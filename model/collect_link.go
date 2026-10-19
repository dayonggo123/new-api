package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm/clause"
)

// 采集队列状态机
// 0 待领取 →(GET /pending 原子领取)→ 1 已领取 →(ack)→ 2 成功 / 3 失败
// ack 失败且 retry<3 → 退回 0（retry+1，清空 device_id）
// ack 失败且 retry>=3 → 终态 3（写 done_time）
// 僵尸回收：status=1 且 claim_time 超 30 分钟未回执 → 退回 0
// 注意：进入任何终态（2 或 3）都必须写 done_time，否则 90 天 TTL 清理永远扫不到
const (
	CollectLinkStatusPending = 0
	CollectLinkStatusClaimed = 1
	CollectLinkStatusSuccess = 2
	CollectLinkStatusFailed  = 3
)

const CollectLinkMaxRetry = 3

type CollectLink struct {
	Id          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index:idx_collect_link_user_status,priority:1;index:idx_collect_link_user_create,priority:1;uniqueIndex:uk_collect_link_user_dedup,priority:1;not null"`
	Url         string `json:"url" gorm:"type:text;not null"`
	ResolvedUrl string `json:"resolved_url" gorm:"type:varchar(512)"`
	Platform    string `json:"platform" gorm:"type:varchar(16);not null;default:''"`
	UrlKey      string `json:"-" gorm:"type:varchar(64);uniqueIndex:uk_collect_link_user_dedup,priority:2;not null"`
	Status      int    `json:"status" gorm:"index:idx_collect_link_user_status,priority:2;not null;default:0"`
	DeviceId    int64  `json:"device_id"`
	Retry       int    `json:"retry" gorm:"not null;default:0"`
	Error       string `json:"error" gorm:"type:varchar(512);not null;default:''"`
	CreateTime  int64  `json:"create_time" gorm:"bigint;not null;index:idx_collect_link_user_create,priority:2"`
	ClaimTime   int64  `json:"claim_time" gorm:"bigint;not null;default:0"`
	DoneTime    int64  `json:"done_time" gorm:"bigint;not null;default:0"`
}

func (CollectLink) TableName() string {
	return "collect_links"
}

func CollectLinkURLKey(resolvedUrl string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(resolvedUrl))))
	return hex.EncodeToString(sum[:])
}

// isDuplicateKeyError 判断唯一索引冲突。
// 注意：本项目 GORM 未开启 TranslateError，gorm.ErrDuplicatedKey 不会生效，需按各驱动错误串识别。
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Duplicate entry") || // MySQL
		strings.Contains(s, "UNIQUE constraint failed") || // SQLite
		strings.Contains(s, "duplicate key value violates unique constraint") // PostgreSQL
}

// InsertCollectLink 幂等插入：命中 (user_id, url_key) 唯一索引时返回已有记录
func InsertCollectLink(link *CollectLink) (existingId int64, duplicated bool, err error) {
	err = DB.Create(link).Error
	if err == nil {
		return link.Id, false, nil
	}
	if !isDuplicateKeyError(err) {
		return 0, false, err
	}
	var existing CollectLink
	if findErr := DB.Select("id").Where("user_id = ? AND url_key = ?", link.UserId, link.UrlKey).First(&existing).Error; findErr != nil {
		return 0, false, err
	}
	return existing.Id, true, nil
}

// CountUserPendingCollectLinks 该用户待领取积压数
func CountUserPendingCollectLinks(userId int) (int64, error) {
	var count int64
	err := DB.Model(&CollectLink{}).Where("user_id = ? AND status = ?", userId, CollectLinkStatusPending).Count(&count).Error
	return count, err
}

// CountUserRecentCollectSubmits 滚动 24 小时提交数（配额口径：rolling window，不用自然日）
func CountUserRecentCollectSubmits(userId int, since int64) (int64, error) {
	var count int64
	err := DB.Model(&CollectLink{}).Where("user_id = ? AND create_time >= ?", userId, since).Count(&count).Error
	return count, err
}

// ClaimPendingCollectLinks 原子领取，三套实现：
//   - PostgreSQL：单条 UPDATE ... RETURNING
//   - MySQL 8：同事务 SELECT ... FOR UPDATE SKIP LOCKED → UPDATE（MySQL 不支持 UPDATE ... RETURNING）
//   - SQLite：事务内 SELECT + 带 status 守卫的 UPDATE（单写者天然串行）
//
// 禁止先 SELECT 再 UPDATE（无锁），否则并发轮询会重复领取同一条。
func ClaimPendingCollectLinks(userId int, deviceId int64, limit int, now int64) ([]CollectLink, error) {
	if common.UsingPostgreSQL {
		return claimPendingCollectLinksPostgreSQL(userId, deviceId, limit, now)
	}
	if common.UsingMySQL {
		return claimPendingCollectLinksMySQL(userId, deviceId, limit, now)
	}
	return claimPendingCollectLinksSQLite(userId, deviceId, limit, now)
}

func claimPendingCollectLinksPostgreSQL(userId int, deviceId int64, limit int, now int64) ([]CollectLink, error) {
	items := make([]CollectLink, 0)
	// PostgreSQL 专用语法（RETURNING），列名为小写无需引号
	err := DB.Raw(`UPDATE collect_links SET status = ?, device_id = ?, claim_time = ?
		WHERE id IN (
			SELECT id FROM collect_links
			WHERE user_id = ? AND status = ?
			ORDER BY id ASC LIMIT ?
		)
		RETURNING id, resolved_url AS url, platform`,
		CollectLinkStatusClaimed, deviceId, now,
		userId, CollectLinkStatusPending, limit,
	).Scan(&items).Error
	return items, err
}

func claimPendingCollectLinksMySQL(userId int, deviceId int64, limit int, now int64) ([]CollectLink, error) {
	items := make([]CollectLink, 0)
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Select("id", "resolved_url AS url", "platform").
		Where("user_id = ? AND status = ?", userId, CollectLinkStatusPending).
		Order("id ASC").Limit(limit).
		Find(&items).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.Id)
		}
		err = tx.Model(&CollectLink{}).Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":     CollectLinkStatusClaimed,
				"device_id":  deviceId,
				"claim_time": now,
			}).Error
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return items, nil
}

func claimPendingCollectLinksSQLite(userId int, deviceId int64, limit int, now int64) ([]CollectLink, error) {
	items := make([]CollectLink, 0)
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	err := tx.Select("id", "resolved_url AS url", "platform").
		Where("user_id = ? AND status = ?", userId, CollectLinkStatusPending).
		Order("id ASC").Limit(limit).
		Find(&items).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.Id)
		}
		// status 守卫：并发事务已认领的行会被跳过，配合 SQLite 单写者串行，不会重复领取
		res := tx.Model(&CollectLink{}).
			Where("id IN ? AND status = ?", ids, CollectLinkStatusPending).
			Updates(map[string]interface{}{
				"status":     CollectLinkStatusClaimed,
				"device_id":  deviceId,
				"claim_time": now,
			})
		if res.Error != nil {
			tx.Rollback()
			return nil, res.Error
		}
		if res.RowsAffected != int64(len(items)) {
			tx.Rollback()
			return nil, fmt.Errorf("claim raced, expected %d rows, updated %d", len(items), res.RowsAffected)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return items, nil
}

// AckResult 桌面端回执单项
type AckResult struct {
	Id     int64  `json:"id"`
	Status int    `json:"status"` // 只接受 2（成功）或 3（失败）
	Error  string `json:"error"`
}

// ProcessCollectAck 处理回执批次，返回实际更新的条数。
// 规则：
//   - 已终态（2/3）的 id 再次 ack → 忽略（幂等）
//   - 仅认领该记录的设备可回执（device_id 守卫）
//   - 成功 → status=2，done_time=now，清 error
//   - 失败且 retry<3 → 退回 status=0，retry+1，清空 device_id/claim_time（换机由僵尸回收兜底）
//   - 失败且 retry>=3 → status=3 终态，done_time=now
func ProcessCollectAck(device *CollectDevice, results []AckResult, now int64) (int, error) {
	updated := 0
	for _, r := range results {
		if r.Status != CollectLinkStatusSuccess && r.Status != CollectLinkStatusFailed {
			continue
		}
		// 规范 §4.1：error 超 512 截断
		if len(r.Error) > 512 {
			r.Error = r.Error[:512]
		}
		var link CollectLink
		err := DB.Select("id", "status", "device_id", "retry").
			Where("id = ? AND user_id = ?", r.Id, device.UserId).
			First(&link).Error
		if err != nil {
			continue // 不存在或他人记录，忽略
		}
		if link.Status == CollectLinkStatusSuccess || link.Status == CollectLinkStatusFailed {
			continue // 终态幂等
		}
		if link.Status != CollectLinkStatusClaimed || link.DeviceId != device.Id {
			continue // 非本设备认领（含已被僵尸回收退回 pending 的）
		}
		updates := map[string]interface{}{}
		if r.Status == CollectLinkStatusSuccess {
			updates["status"] = CollectLinkStatusSuccess
			updates["done_time"] = now
			updates["error"] = ""
		} else if link.Retry < CollectLinkMaxRetry {
			updates["status"] = CollectLinkStatusPending
			updates["retry"] = link.Retry + 1
			updates["device_id"] = nil
			updates["claim_time"] = 0
			updates["error"] = r.Error
		} else {
			updates["status"] = CollectLinkStatusFailed
			updates["done_time"] = now
			updates["error"] = r.Error
		}
		res := DB.Model(&CollectLink{}).Where("id = ? AND status = ? AND device_id = ?",
			link.Id, CollectLinkStatusClaimed, device.Id).Updates(updates)
		if res.Error != nil {
			return updated, res.Error
		}
		if res.RowsAffected > 0 {
			updated++
		}
	}
	return updated, nil
}

// ReclaimZombieCollectLinks 僵尸回收：已领取超 30 分钟未回执 → 退回待领取
func ReclaimZombieCollectLinks(deadline int64) (int64, error) {
	res := DB.Model(&CollectLink{}).
		Where("status = ? AND claim_time < ?", CollectLinkStatusClaimed, deadline).
		Updates(map[string]interface{}{
			"status":     CollectLinkStatusPending,
			"device_id":  nil,
			"claim_time": 0,
		})
	return res.RowsAffected, res.Error
}

// CleanupFinishedCollectLinks 清理 90 天前进入终态的记录
func CleanupFinishedCollectLinks(deadline int64) (int64, error) {
	res := DB.Where("status IN ? AND done_time > 0 AND done_time < ?",
		[]int{CollectLinkStatusSuccess, CollectLinkStatusFailed}, deadline).
		Delete(&CollectLink{})
	return res.RowsAffected, res.Error
}
