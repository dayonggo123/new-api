package model

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试前置：TaskMain（task_cas_test.go）已初始化内存 SQLite 单连接 DB。
// 这里只补迁移采集队列表。
func ensureCollectTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&CollectLink{}, &CollectDevice{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM collect_links")
		DB.Exec("DELETE FROM collect_devices")
	})
}

func newTestLink(userId int, resolved string) *CollectLink {
	return &CollectLink{
		UserId:      userId,
		Url:         resolved,
		ResolvedUrl: resolved,
		Platform:    "tiktok",
		UrlKey:      CollectLinkURLKey(resolved),
		CreateTime:  time.Now().Unix(),
	}
}

// 幂等：同 user + url_key 重复插入返回 duplicated=true 且 id 相同（规范验收用例 6）
func TestInsertCollectLinkIdempotent(t *testing.T) {
	ensureCollectTables(t)
	link := newTestLink(1, "https://vm.tiktok.com/AAA/")
	id1, dup, err := InsertCollectLink(link)
	require.NoError(t, err)
	assert.False(t, dup)
	assert.Greater(t, id1, int64(0))

	link2 := newTestLink(1, "https://vm.tiktok.com/AAA/")
	id2, dup, err := InsertCollectLink(link2)
	require.NoError(t, err)
	assert.True(t, dup)
	assert.Equal(t, id1, id2)

	// 不同用户同 URL 不算重复
	link3 := newTestLink(2, "https://vm.tiktok.com/AAA/")
	id3, dup, err := InsertCollectLink(link3)
	require.NoError(t, err)
	assert.False(t, dup)
	assert.NotEqual(t, id1, id3)
}

// ack 状态机全路径：成功终态 / 失败重试退回 / 三次失败终态 / 幂等 / 跨设备守卫
func TestProcessCollectAckStateMachine(t *testing.T) {
	ensureCollectTables(t)
	userId := 42
	deviceA := &CollectDevice{Id: 1001, UserId: userId}
	deviceB := &CollectDevice{Id: 1002, UserId: userId}
	now := time.Now().Unix()

	// --- 成功路径 ---
	link := newTestLink(userId, "https://vm.tiktok.com/OK/")
	_, _, err := InsertCollectLink(link)
	require.NoError(t, err)
	claimed, err := ClaimPendingCollectLinks(userId, deviceA.Id, 5, now)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	updated, err := ProcessCollectAck(deviceA, []AckResult{{Id: claimed[0].Id, Status: CollectLinkStatusSuccess}}, now)
	require.NoError(t, err)
	assert.Equal(t, 1, updated)

	var got CollectLink
	require.NoError(t, DB.First(&got, claimed[0].Id).Error)
	assert.Equal(t, CollectLinkStatusSuccess, got.Status)
	assert.Greater(t, got.DoneTime, int64(0)) // v1.1：终态必须写 done_time

	// 终态重复 ack → 幂等忽略
	updated, err = ProcessCollectAck(deviceA, []AckResult{{Id: claimed[0].Id, Status: CollectLinkStatusSuccess}}, now)
	require.NoError(t, err)
	assert.Equal(t, 0, updated)

	// --- 失败重试路径：retry < 3 退回 pending（最多 3 次重试），retry >= 3 终态 ---
	// 规范 §2.3：初始尝试 + 3 次重试 = 共 4 次失败才进入终态
	link2 := newTestLink(userId, "https://vm.tiktok.com/RETRY/")
	_, _, err = InsertCollectLink(link2)
	require.NoError(t, err)

	var lastId int64
	const totalAttempts = CollectLinkMaxRetry + 1
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		claimed, err := ClaimPendingCollectLinks(userId, deviceA.Id, 1, now)
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		lastId = claimed[0].Id

		updated, err := ProcessCollectAck(deviceA, []AckResult{{Id: lastId, Status: CollectLinkStatusFailed, Error: "HTTP 403"}}, now)
		require.NoError(t, err)
		assert.Equal(t, 1, updated)

		var got CollectLink
		require.NoError(t, DB.First(&got, lastId).Error)
		if attempt < totalAttempts {
			// 退回 pending：retry+1、清 device_id、无 done_time
			assert.Equal(t, CollectLinkStatusPending, got.Status)
			assert.Equal(t, attempt, got.Retry)
			assert.Equal(t, int64(0), got.DeviceId)
			assert.Equal(t, int64(0), got.DoneTime)
			assert.Equal(t, "HTTP 403", got.Error)
		} else {
			// retry 达到上限 → 终态 3，必须写 done_time（v1.1 评审 P0 修复点）
			assert.Equal(t, CollectLinkStatusFailed, got.Status)
			assert.Greater(t, got.DoneTime, int64(0))
		}
	}

	// --- 跨设备守卫：B 设备不能 ack A 设备认领的任务 ---
	link3 := newTestLink(userId, "https://vm.tiktok.com/GUARD/")
	_, _, err = InsertCollectLink(link3)
	require.NoError(t, err)
	claimed3, err := ClaimPendingCollectLinks(userId, deviceA.Id, 1, now)
	require.NoError(t, err)
	require.Len(t, claimed3, 1)

	updated, err = ProcessCollectAck(deviceB, []AckResult{{Id: claimed3[0].Id, Status: CollectLinkStatusSuccess}}, now)
	require.NoError(t, err)
	assert.Equal(t, 0, updated)

	var got3 CollectLink
	require.NoError(t, DB.First(&got3, claimed3[0].Id).Error)
	assert.Equal(t, CollectLinkStatusClaimed, got3.Status) // 状态不变
}

// 用例 5：并发领取不重复（内存 SQLite 单连接串行执行，验证语义正确性；
// 生产 MySQL/PG 依赖 SKIP LOCKED / RETURNING 保证）
func TestClaimConcurrentNoDuplicate(t *testing.T) {
	ensureCollectTables(t)
	userId := 7
	const total = 20
	for i := 0; i < total; i++ {
		link := newTestLink(userId, "https://vm.tiktok.com/X"+string(rune('A'+i%26))+string(rune('a'+i/26))+"/")
		_, _, err := InsertCollectLink(link)
		require.NoError(t, err)
	}

	now := time.Now().Unix()
	var wg sync.WaitGroup
	idsCh := make(chan []int64, 10)
	errCh := make(chan error, 10)
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			items, err := ClaimPendingCollectLinks(userId, int64(2000+worker), 5, now)
			if err != nil {
				errCh <- err
				return
			}
			ids := make([]int64, 0, len(items))
			for _, it := range items {
				ids = append(ids, it.Id)
			}
			idsCh <- ids
		}(w)
	}
	wg.Wait()
	close(idsCh)
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	seen := map[int64]int{}
	claimedTotal := 0
	for ids := range idsCh {
		for _, id := range ids {
			seen[id]++
			claimedTotal++
		}
	}
	for id, cnt := range seen {
		assert.Equal(t, 1, cnt, "link %d 被重复领取", id)
	}
	assert.LessOrEqual(t, claimedTotal, total)

	// 返回的 url 必须是 resolved_url
	var first CollectLink
	require.NoError(t, DB.First(&first).Error)
	assert.NotEmpty(t, first.ResolvedUrl)
}

// 僵尸回收：已领取超 30 分钟未回执 → 退回 pending
func TestReclaimZombieCollectLinks(t *testing.T) {
	ensureCollectTables(t)
	userId := 9
	link := newTestLink(userId, "https://vm.tiktok.com/ZOMBIE/")
	_, _, err := InsertCollectLink(link)
	require.NoError(t, err)
	now := time.Now().Unix()
	claimed, err := ClaimPendingCollectLinks(userId, 3001, 1, now-31*60)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	n, err := ReclaimZombieCollectLinks(now - 30*60)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	var got CollectLink
	require.NoError(t, DB.First(&got, claimed[0].Id).Error)
	assert.Equal(t, CollectLinkStatusPending, got.Status)
	assert.Equal(t, int64(0), got.DeviceId)
	assert.Equal(t, int64(0), got.ClaimTime)
}

// 90 天 TTL：done_time>0 且超期的终态记录被清理；
// done_time=0 的记录（v1.1 之前的 bug 数据）不被误删依赖新逻辑：进入终态必写 done_time
func TestCleanupFinishedCollectLinks(t *testing.T) {
	ensureCollectTables(t)
	userId := 11
	now := time.Now().Unix()
	deadline := now - 90*24*3600

	// 旧的成功记录（done_time 超期）→ 应被清理
	old := newTestLink(userId, "https://vm.tiktok.com/OLD/")
	old.Status = CollectLinkStatusSuccess
	old.DoneTime = deadline - 100
	require.NoError(t, DB.Create(old).Error)

	// 新的成功记录 → 保留
	recent := newTestLink(userId, "https://vm.tiktok.com/RECENT/")
	recent.Status = CollectLinkStatusSuccess
	recent.DoneTime = now
	require.NoError(t, DB.Create(recent).Error)

	n, err := CleanupFinishedCollectLinks(deadline)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	var count int64
	require.NoError(t, DB.Model(&CollectLink{}).Where("id = ?", old.Id).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	require.NoError(t, DB.Model(&CollectLink{}).Where("id = ?", recent.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
