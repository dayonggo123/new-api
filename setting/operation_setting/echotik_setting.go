package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type EchotikSetting struct {
	EchotikEnabled  bool   `json:"echotik_enabled"`  // 是否启用 EchoTik 接口代理
	EchotikBaseURL  string `json:"echotik_base_url"` // EchoTik API 基础地址，如 https://open.echotik.live
	EchotikUsername string `json:"echotik_username"` // EchoTik 用户名
	EchotikPassword string `json:"echotik_password"` // EchoTik 密码

	// 缓存总开关与 TTL（秒）
	EchotikCacheEnabled            bool `json:"echotik_cache_enabled"`              // 是否启用本地缓存
	EchotikCacheTodayTTLSeconds    int  `json:"echotik_cache_today_ttl_seconds"`    // 当天数据 TTL
	EchotikCacheLast7DaysTTLSeconds int `json:"echotik_cache_last_7_days_ttl_seconds"` // 近 7 天数据 TTL
	EchotikCacheOlderTTLSeconds    int  `json:"echotik_cache_older_ttl_seconds"`    // 7 天前数据 TTL

	// 数据保留
	EchotikCacheRetentionDays int `json:"echotik_cache_retention_days"` // 快照物理保留天数

	// 定时预热同步
	EchotikSyncEnabled             bool     `json:"echotik_sync_enabled"`               // 是否启用定时预热
	EchotikSyncFrequencyHours      int      `json:"echotik_sync_frequency_hours"`       // 同步周期（小时）
	EchotikSyncQPS                 int      `json:"echotik_sync_qps"`                   // 同步限流（请求/秒）
	EchotikSyncRegions             []string `json:"echotik_sync_regions"`               // 预同步区域列表
	EchotikSyncRankFields          []int    `json:"echotik_sync_rank_fields"`           // 预同步 video_rank_field 列表
	EchotikSyncRankTypes           []int    `json:"echotik_sync_rank_types"`            // 预同步 rank_type 列表
	EchotikSyncProductCategoryIDs  []string `json:"echotik_sync_product_category_ids"`  // 预同步商品类目列表，空字符串表示默认
	EchotikSyncCreatedByAIOptions  []string `json:"echotik_sync_created_by_ai_options"` // 预同步 created_by_ai 列表
	EchotikSyncMaxPages            int      `json:"echotik_sync_max_pages"`             // 每组合预同步最大页数
	EchotikSyncPageSize            int      `json:"echotik_sync_page_size"`             // 预同步每页条数
	EchotikSyncDateDays            int      `json:"echotik_sync_date_days"`             // 预同步最近 N 天（包含今天）
}

var echotikSetting = EchotikSetting{
	EchotikEnabled:  false,
	EchotikBaseURL:  "https://open.echotik.live",
	EchotikUsername: "",
	EchotikPassword: "",

	EchotikCacheEnabled:             true,
	EchotikCacheTodayTTLSeconds:     3600,    // 1 小时
	EchotikCacheLast7DaysTTLSeconds: 21600,   // 6 小时
	EchotikCacheOlderTTLSeconds:     2592000, // 30 天

	EchotikCacheRetentionDays: 1, // 1 天

	EchotikSyncEnabled:            true,
	EchotikSyncFrequencyHours:     24,
	EchotikSyncQPS:                1,
	EchotikSyncRegions:            []string{"US", "MX", "BR", "VN", "TH", "PH", "MY", "ID", "SG", "JP", "GB", "ES", "DE", "IT", "FR"},
	EchotikSyncRankFields:         []int{1, 2},
	EchotikSyncRankTypes:          []int{1, 2, 3},
	EchotikSyncProductCategoryIDs: []string{""},
	EchotikSyncCreatedByAIOptions: []string{""},
	EchotikSyncMaxPages:           1,
	EchotikSyncPageSize:           10,
	EchotikSyncDateDays:           3,
}

func init() {
	config.GlobalConfig.Register("echotik_setting", &echotikSetting)
}

func GetEchotikSetting() *EchotikSetting {
	return &echotikSetting
}
