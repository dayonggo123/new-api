package model

// MarketingBanner 运营横幅 Banner 表
type MarketingBanner struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Priority         int    `json:"priority" gorm:"default:0;index"`
	Enabled          bool   `json:"enabled" gorm:"default:true"`
	StartAt          int64  `json:"start_at" gorm:"default:0;index"`
	EndAt            int64  `json:"end_at" gorm:"default:0;index"`
	MaxDismissHours  int    `json:"max_dismiss_hours" gorm:"default:24"`
	Content          string `json:"content" gorm:"type:text;not null"`
	CreatedAt        int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (MarketingBanner) TableName() string {
	return "marketing_banners"
}

// BannerContent 按语言编码的内容
type BannerContent struct {
	Text           string `json:"text"`
	Cta            string `json:"cta"`
	ActionType     string `json:"action_type"`
	ActionPayload  string `json:"action_payload"`
}

// GetActiveBanners 获取当前生效的 banner 列表（按优先级降序）
func GetActiveBanners(now int64) ([]*MarketingBanner, error) {
	var banners []*MarketingBanner
	err := DB.Where("enabled = ? AND start_at <= ? AND (end_at = 0 OR end_at >= ?)", true, now, now).
		Order("priority DESC, created_at DESC").
		Find(&banners).Error
	if err != nil {
		return nil, err
	}
	return banners, nil
}

// GetAllBanners 获取所有 banner（后台管理）
func GetAllBanners() ([]*MarketingBanner, error) {
	var banners []*MarketingBanner
	err := DB.Order("priority DESC, created_at DESC").Find(&banners).Error
	return banners, err
}

// GetBannerById 根据 ID 获取 banner
func GetBannerById(id int) (*MarketingBanner, error) {
	var banner MarketingBanner
	err := DB.First(&banner, id).Error
	if err != nil {
		return nil, err
	}
	return &banner, nil
}

// Insert 创建 banner
func (banner *MarketingBanner) Insert() error {
	return DB.Create(banner).Error
}

// Update 更新 banner
func (banner *MarketingBanner) Update() error {
	return DB.Save(banner).Error
}

// DeleteBanner 删除 banner
func DeleteBanner(id int) error {
	return DB.Delete(&MarketingBanner{}, id).Error
}
