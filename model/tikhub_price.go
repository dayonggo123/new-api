package model

import (
	"time"

	"gorm.io/gorm"
)

// TikHubPriceConfig TikHub 接口收费配置表
type TikHubPriceConfig struct {
	Id          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Endpoint    string    `json:"endpoint" gorm:"uniqueIndex;size:100;not null"` // 接口标识
	Name        string    `json:"name" gorm:"size:100;not null"`                // 接口名称
	Description string    `json:"description" gorm:"type:text"`                   // 接口描述
	Price       float64  `json:"price" gorm:"default:0"`                     // 单次调用价格(积分)
	Enabled     bool     `json:"enabled" gorm:"default:true"`                    // 是否启用收费
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
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

// InitDefaultTikHubPriceConfigs 初始化默认收费配置
func InitDefaultTikHubPriceConfigs() error {
	defaultConfigs := []TikHubPriceConfig{
		{Endpoint: "video", Name: "获取单个视频数据", Description: "通过 aweme_id 获取视频数据", Price: 1, Enabled: true},
		{Endpoint: "video-by-share-url", Name: "通过分享链接获取视频", Description: "通过分享链接获取视频数据", Price: 1, Enabled: true},
		{Endpoint: "comment-keywords", Name: "评论关键词分析", Description: "分析视频评论中的热门关键词", Price: 2, Enabled: true},
		{Endpoint: "music-chart-list", Name: "音乐排行榜", Description: "获取热门音乐排行榜", Price: 1, Enabled: true},
		{Endpoint: "trending-search-words", Name: "每日趋势搜索词", Description: "获取每日趋势搜索关键词", Price: 1, Enabled: true},
		{Endpoint: "product", Name: "商品详情", Description: "获取 TikTok 商品详情", Price: 2, Enabled: true},
		{Endpoint: "account-health-status", Name: "账号健康状态", Description: "获取创作者账号健康状态(违规积分)", Price: 3, Enabled: true},
		{Endpoint: "account-insights-overview", Name: "账号概览", Description: "获取创作者账号表现概览", Price: 3, Enabled: true},
		{Endpoint: "video-analytics-summary", Name: "视频概览", Description: "获取创作者视频表现概览", Price: 3, Enabled: true},
		{Endpoint: "product-related-videos", Name: "同款商品关联视频", Description: "获取同款商品关联视频列表", Price: 2, Enabled: true},
		{Endpoint: "trends-hashtag-list", Name: "热门标签榜单", Description: "获取热门标签排行榜", Price: 1, Enabled: true},
		{Endpoint: "hot-selling-products-list", Name: "热卖商品列表", Description: "获取 TikTok Shop 热卖商品列表", Price: 1, Enabled: true},
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
