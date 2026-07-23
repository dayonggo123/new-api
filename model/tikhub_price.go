package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// TikHubPriceConfig TikHub 接口收费配置表
type TikHubPriceConfig struct {
	Id          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Endpoint    string    `json:"endpoint" gorm:"uniqueIndex;size:100;not null"` // 接口标识
	Name        string    `json:"name" gorm:"size:100;not null"`                // 接口名称
	Description string    `json:"description" gorm:"type:text"`                   // 接口描述

	// 普通用户价格
	Price float64 `json:"price" gorm:"default:0"` // 单次调用价格(美元)

	// VIP 用户价格
	VipPrice float64 `json:"vip_price" gorm:"default:0"` // VIP 单次调用价格(美元)

	// SVIP 用户价格
	SvipPrice float64 `json:"svip_price" gorm:"default:0"` // SVIP 单次调用价格(美元)

	// 免费条数
	FreeQuota     int  `json:"free_quota" gorm:"default:0"`      // 普通用户免费条数
	VipFreeQuota  int  `json:"vip_free_quota" gorm:"default:0"`   // VIP 用户免费条数
	SvipFreeQuota int  `json:"svip_free_quota" gorm:"default:0"` // SVIP 用户免费条数

	Enabled   bool      `json:"enabled" gorm:"default:true"` // 是否启用收费
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TikHubPriceConfig) TableName() string {
	return "tikhub_price_configs"
}

// GetAllTikHubPriceConfigs 获取所有收费配置
func GetAllTikHubPriceConfigs() ([]*TikHubPriceConfig, error) {
	var configs []*TikHubPriceConfig
	err := DB.Order("id ASC").Find(&configs).Error
	return configs, err
}

// GetEnabledTikHubPriceConfigs 获取已启用的收费配置
func GetEnabledTikHubPriceConfigs() (map[string]*TikHubPriceConfig, error) {
	var configs []*TikHubPriceConfig
	err := DB.Where("enabled = ?", true).Find(&configs).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]*TikHubPriceConfig)
	for _, c := range configs {
		result[c.Endpoint] = c
	}
	return result, nil
}

// TikHubPriceConfigPublic 公开接口返回的价格配置
type TikHubPriceConfigPublic struct {
	Endpoint    string  `json:"endpoint"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"` // 普通用户价格
	Enabled     bool    `json:"enabled"`
	Quota       int     `json:"quota"` // 积分 = price * 100
	Description string  `json:"description"`

	// 等级价格
	VipPrice       float64 `json:"vip_price"`
	SvipPrice      float64 `json:"svip_price"`
	VipQuota       int     `json:"vip_quota"`
	SvipQuota      int     `json:"svip_quota"`

	// 免费条数
	FreeQuota     int `json:"free_quota"`
	VipFreeQuota  int `json:"vip_free_quota"`
	SvipFreeQuota int `json:"svip_free_quota"`
}

// GetTikHubPriceConfigsForPublic 公开接口：获取已启用的收费配置（返回简洁格式）
func GetTikHubPriceConfigsForPublic() ([]TikHubPriceConfigPublic, error) {
	var configs []TikHubPriceConfig
	err := DB.Where("enabled = ?", true).Find(&configs).Error
	if err != nil {
		return nil, err
	}

	result := make([]TikHubPriceConfigPublic, len(configs))
	for i, c := range configs {
		result[i] = TikHubPriceConfigPublic{
			Endpoint:       c.Endpoint,
			Name:           c.Name,
			Price:          c.Price,
			Enabled:        c.Enabled,
			Quota:          int(c.Price * 100),
			Description:    c.Description,
			VipPrice:       c.VipPrice,
			SvipPrice:      c.SvipPrice,
			VipQuota:       int(c.VipPrice * 100),
			SvipQuota:      int(c.SvipPrice * 100),
			FreeQuota:      c.FreeQuota,
			VipFreeQuota:   c.VipFreeQuota,
			SvipFreeQuota:  c.SvipFreeQuota,
		}
	}
	return result, nil
}

// GetTikHubPriceConfigByEndpoint 根据接口标识获取配置
func GetTikHubPriceConfigByEndpoint(endpoint string) (*TikHubPriceConfig, error) {
	var config TikHubPriceConfig
	err := DB.Where("endpoint = ? AND enabled = ?", endpoint, true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Insert 创建收费配置
func (config *TikHubPriceConfig) Insert() error {
	return DB.Create(config).Error
}

// Update 更新收费配置
func (config *TikHubPriceConfig) Update() error {
	return DB.Save(config).Error
}

// DeleteTikHubPriceConfig 删除收费配置
func DeleteTikHubPriceConfig(id int) error {
	return DB.Delete(&TikHubPriceConfig{}, id).Error
}

// GetUserTikHubTier 获取用户的 TikHub 等级
// 返回: "svip", "vip", "common", "admin", "root"
func GetUserTikHubTier(userID int) string {
	// 先检查用户角色
	var user User
	err := DB.Where("id = ?", userID).Select("role").First(&user).Error
	if err != nil {
		return "common"
	}

	// Root 和 Admin 用户免费
	if user.Role == common.RoleRootUser {
		return "root"
	}
	if user.Role == common.RoleAdminUser {
		return "admin"
	}

	// 检查是否有 SVIP 订阅
	var sub UserSubscription
	err = DB.Where("user_id = ? AND status = ? AND end_time > ?", userID, "active", time.Now().Unix()).
		Order("end_time DESC").First(&sub).Error
	if err == nil && sub.PlanId > 0 {
		// 获取订阅计划
		var plan SubscriptionPlan
		err = DB.Where("id = ?", sub.PlanId).First(&plan).Error
		if err == nil {
			// 根据升级组判断等级
			switch plan.UpgradeGroup {
			case "svip":
				return "svip"
			case "vip":
				return "vip"
			}
		}
	}

	return "common"
}

// GetTikHubPriceWithTier 根据用户等级获取价格配置
// 返回: price (实际价格), quota (积分), freeQuota (免费条数), shouldCharge (是否应该扣费)
func (config *TikHubPriceConfig) GetTikHubPriceWithTier(tier string) (price float64, quota int, freeQuota int, shouldCharge bool) {
	switch tier {
	case "root", "admin":
		// 管理员用户免费
		return 0, 0, 0, false
	case "svip":
		price = config.SvipPrice
		freeQuota = config.SvipFreeQuota
	case "vip":
		price = config.VipPrice
		freeQuota = config.VipFreeQuota
	default:
		price = config.Price
		freeQuota = config.FreeQuota
	}

	// 如果价格为0或者免费条数大于0，则免费
	if price <= 0 || freeQuota > 0 {
		return 0, 0, freeQuota, false
	}

	// 计算积分
	quota = int(price * 100)
	return price, quota, freeQuota, true
}

// InitDefaultTikHubPriceConfigs 初始化默认收费配置（美元）
func InitDefaultTikHubPriceConfigs() error {
	// 价格单位：美元 (USD)
	// 实际扣费 = 价格 * 100 积分
	defaultConfigs := []TikHubPriceConfig{
		{Endpoint: "video", Name: "获取单个视频数据", Description: "通过 aweme_id 获取视频数据", Price: 0.01, Enabled: true},
		{Endpoint: "video-by-share-url", Name: "通过分享链接获取视频", Description: "通过分享链接获取视频数据", Price: 0.01, Enabled: true},
		{Endpoint: "video-comments", Name: "获取视频评论", Description: "获取单个视频评论数据", Price: 0.02, Enabled: true},
		{Endpoint: "post-comment", Name: "获取作品评论列表", Description: "获取作品评论列表(Web)", Price: 0.02, Enabled: true},
		{Endpoint: "comment-keywords", Name: "评论关键词分析", Description: "分析视频评论中的热门关键词", Price: 0.02, Enabled: true},
		{Endpoint: "music-chart-list", Name: "音乐排行榜", Description: "获取热门音乐排行榜", Price: 0.01, Enabled: true},
		{Endpoint: "trending-search-words", Name: "每日趋势搜索词", Description: "获取每日趋势搜索关键词", Price: 0.01, Enabled: true},
		{Endpoint: "product", Name: "商品详情", Description: "获取 TikTok 商品详情", Price: 0.02, Enabled: true},
		{Endpoint: "account-health-status", Name: "账号健康状态", Description: "获取创作者账号健康状态(违规积分)", Price: 0.03, Enabled: true},
		{Endpoint: "account-insights-overview", Name: "账号概览", Description: "获取创作者账号表现概览", Price: 0.03, Enabled: true},
		{Endpoint: "video-analytics-summary", Name: "视频概览", Description: "获取创作者视频表现概览", Price: 0.03, Enabled: true},
		{Endpoint: "video-audience-stats", Name: "视频受众分析", Description: "获取视频受众分析数据(性别/年龄/地区分布)", Price: 0.03, Enabled: true},
		{Endpoint: "product-related-videos", Name: "同款商品关联视频", Description: "获取同款商品关联视频列表", Price: 0.02, Enabled: true},
		{Endpoint: "trends-hashtag-list", Name: "热门标签榜单", Description: "获取热门标签排行榜", Price: 0.01, Enabled: true},
		{Endpoint: "hot-selling-products-list", Name: "热卖商品列表", Description: "获取 TikTok Shop 热卖商品列表", Price: 0.01, Enabled: true},
		{Endpoint: "video-metrics", Name: "视频统计数据", Description: "获取视频观看量、点赞、评论、收藏等统计数据", Price: 0.03, Enabled: true},
		{Endpoint: "detect-fake-views", Name: "虚假流量检测", Description: "检测视频虚假流量分析，评估流量质量", Price: 0.05, Enabled: true},
		{Endpoint: "creator-info-milestones", Name: "创作者信息与里程碑", Description: "获取创作者基本信息和成长里程碑", Price: 0.03, Enabled: true},
	}

	for _, c := range defaultConfigs {
		var exist TikHubPriceConfig
		if err := DB.Where("endpoint = ?", c.Endpoint).First(&exist).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := DB.Create(&c).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}
