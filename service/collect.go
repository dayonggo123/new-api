package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

// 采集队列运营参数（规范 §4.3 建议值，可按运营调整）
const (
	CollectDailySubmitQuota  = 200  // 单用户滚动 24h 提交上限
	CollectMaxPendingQueue   = 500  // 单用户 pending 积压上限（超出 409）
	CollectMaxAckBatch       = 100  // ack 单次最多条数（超出 400 BATCH_TOO_LARGE）
	CollectMaxURLLen         = 4096 // 脏文本截断长度
	CollectMaxResolvedURLLen = 512
	CollectRollingWindowSecs = 86400 // 滚动 24h（不用自然日：跨境用户活跃时段跨零点）
	CollectTokenFailLimit    = 5     // 令牌校验失败 5 次/分钟/IP
	CollectTokenFailWindow   = time.Minute
)

var (
	ErrCollectInvalidURL    = errors.New("未从输入中找到 tiktok/douyin 链接")
	ErrCollectInvalidToken  = errors.New("设备令牌无效")
	ErrCollectDeviceRevoked = errors.New("设备令牌已吊销")
	ErrCollectQueueFull     = errors.New("待采集积压过多，请先打开桌面端")
	ErrCollectRateLimited   = errors.New("超过单日提交配额（200 条/24h），请明天再来")
	ErrCollectBatchTooLarge = errors.New("ack 批次过大（上限 100 条）")
)

// 规范 §4.1：从脏文本中提取第一条 tiktok/douyin 链接。
// (?i) 大小写不敏感；尾部标点剥离规则与规范一致。
var collectURLRe = regexp.MustCompile(`(?i)https?://(?:[\w-]+\.)*(?:tiktok|douyin)\.com/[^\s<>"']+`)

func ExtractCollectURL(raw string) (string, bool) {
	match := collectURLRe.FindString(raw)
	if match == "" {
		return "", false
	}
	// 去掉尾部标点：.,;:!?)]}"'
	resolved := strings.TrimRight(match, ".,;:!?)]}'\"")
	if resolved == "" {
		return "", false
	}
	return resolved, true
}

// DetectPlatform 规范 §4.2
func DetectPlatform(resolvedURL string) string {
	lower := strings.ToLower(resolvedURL)
	switch {
	case strings.Contains(lower, "tiktok.com"):
		return "tiktok"
	case strings.Contains(lower, "douyin.com"):
		return "douyin"
	default:
		return "unknown"
	}
}

// ValidateCollectDevice 设备令牌校验：哈希比对，不返回明文
func ValidateCollectDevice(token string) (*model.CollectDevice, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrCollectInvalidToken
	}
	device, err := model.GetCollectDeviceByTokenHash(model.HashCollectToken(token))
	if err != nil {
		return nil, ErrCollectInvalidToken
	}
	if device.Revoked {
		return nil, ErrCollectDeviceRevoked
	}
	return device, nil
}

// SubmitCollectLink 提交链接（幂等：同 user + url_key 重复提交返回 duplicated=true）
func SubmitCollectLink(device *model.CollectDevice, raw string) (int64, bool, error) {
	if len(raw) > CollectMaxURLLen {
		raw = raw[:CollectMaxURLLen]
	}
	resolved, ok := ExtractCollectURL(raw)
	if !ok {
		return 0, false, ErrCollectInvalidURL
	}
	// 规范 §4.1：resolved_url 超 512 视为 INVALID_URL（不截断——截断会产生错误链接）
	if len(resolved) > CollectMaxResolvedURLLen {
		return 0, false, ErrCollectInvalidURL
	}
	now := time.Now().Unix()

	pending, err := model.CountUserPendingCollectLinks(device.UserId)
	if err != nil {
		return 0, false, err
	}
	if pending >= CollectMaxPendingQueue {
		return 0, false, ErrCollectQueueFull
	}

	recent, err := model.CountUserRecentCollectSubmits(device.UserId, now-CollectRollingWindowSecs)
	if err != nil {
		return 0, false, err
	}
	if recent >= CollectDailySubmitQuota {
		return 0, false, ErrCollectRateLimited
	}

	link := &model.CollectLink{
		UserId:      device.UserId,
		Url:         raw,
		ResolvedUrl: resolved,
		Platform:    DetectPlatform(resolved),
		UrlKey:      model.CollectLinkURLKey(resolved),
		CreateTime:  now,
	}
	return model.InsertCollectLink(link)
}

// GetPendingCollectItems 原子领取待采集（响应里 Url 字段即 resolved_url）
func GetPendingCollectItems(device *model.CollectDevice, limit int) ([]model.CollectLink, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	now := time.Now().Unix()
	go model.TouchCollectDeviceLastSeen(device.Id, now)
	return model.ClaimPendingCollectLinks(device.UserId, device.Id, limit, now)
}

// AckCollectResults 回执（幂等：终态 id 重复 ack 直接忽略）
func AckCollectResults(device *model.CollectDevice, results []dto.CollectAckResult) (int, error) {
	if len(results) > CollectMaxAckBatch {
		return 0, ErrCollectBatchTooLarge
	}
	ackResults := make([]model.AckResult, 0, len(results))
	for _, r := range results {
		ackResults = append(ackResults, model.AckResult{Id: r.Id, Status: r.Status, Error: r.Error})
	}
	return model.ProcessCollectAck(device, ackResults, time.Now().Unix())
}

// CreateCollectDevice 网页端生成设备令牌（session 鉴权，controller 负责）
func CreateCollectDevice(userId int, name string) (*model.CollectDevice, string, error) {
	name = sanitizeCollectDeviceName(name)
	return model.CreateCollectDevice(userId, name)
}

func sanitizeCollectDeviceName(name string) string {
	name = strings.TrimSpace(name)
	// 过滤控制字符，防日志注入
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if len(name) > 64 {
		name = name[:64]
	}
	if name == "" {
		name = "未命名设备"
	}
	return name
}

// StartCollectJanitor 周期任务：僵尸回收（5 分钟）+ 90 天 TTL 清理（24 小时）
// 时间戳一律取服务端时间，不采信客户端时钟
func StartCollectJanitor() {
	go func() {
		for {
			if n, err := model.ReclaimZombieCollectLinks(time.Now().Unix() - 30*60); err != nil {
				common.SysError("collect zombie reclaim error: " + err.Error())
			} else if n > 0 {
				common.SysLog(fmt.Sprintf("collect: reclaimed %d zombie links", n))
			}
			time.Sleep(5 * time.Minute)
		}
	}()
	go func() {
		for {
			if n, err := model.CleanupFinishedCollectLinks(time.Now().Unix() - 90*24*3600); err != nil {
				common.SysError("collect TTL cleanup error: " + err.Error())
			} else if n > 0 {
				common.SysLog(fmt.Sprintf("collect: cleaned %d expired links", n))
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}

// ---- 令牌校验失败限流：5 次/分钟/IP ----

type collectTokenFailureLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newCollectTokenFailureLimiter() *collectTokenFailureLimiter {
	return &collectTokenFailureLimiter{hits: make(map[string][]time.Time)}
}

var collectTokenLimiter = newCollectTokenFailureLimiter()

// Allow 该 IP 最近一分钟内失败次数是否未超阈值
func (l *collectTokenFailureLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(ip, time.Now())) < CollectTokenFailLimit
}

// Record 记录一次失败
func (l *collectTokenFailureLimiter) Record(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.hits[ip] = append(l.prune(ip, now), now)
}

func (l *collectTokenFailureLimiter) prune(ip string, now time.Time) []time.Time {
	cutoff := now.Add(-CollectTokenFailWindow)
	hits := l.hits[ip]
	kept := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.hits[ip] = kept
	return kept
}

// CollectTokenLimiterAllow / CollectTokenLimiterRecord 暴露给 controller
func CollectTokenLimiterAllow(ip string) bool { return collectTokenLimiter.Allow(ip) }
func CollectTokenLimiterRecord(ip string)     { collectTokenLimiter.Record(ip) }
