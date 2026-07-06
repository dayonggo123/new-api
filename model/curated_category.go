package model

// CuratedCategory 一键同款模板分类表
type CuratedCategory struct {
	Id        int    `json:"-" gorm:"primaryKey;autoIncrement"`
	Key       string `json:"key" gorm:"size:64;uniqueIndex"`
	Name      string `json:"name" gorm:"not null"`
	IconUrl   string `json:"iconUrl" gorm:"column:icon_url"`
	SortOrder int    `json:"sortOrder" gorm:"column:sort_order;default:0;index"`
	Enabled   bool   `json:"enabled" gorm:"default:true"`
	CreatedAt int64  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt int64  `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (CuratedCategory) TableName() string {
	return "curated_categories"
}

// Insert 创建分类
func (category *CuratedCategory) Insert() error {
	return DB.Create(category).Error
}

// Update 保存分类
func (category *CuratedCategory) Update() error {
	return DB.Save(category).Error
}

// Delete 删除分类
func (category *CuratedCategory) Delete() error {
	return DB.Delete(category).Error
}

// GetCuratedCategoryById 根据自增 ID 获取分类
func GetCuratedCategoryById(id int) (*CuratedCategory, error) {
	var category CuratedCategory
	err := DB.First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetCuratedCategoryByKey 根据业务 key 获取分类
func GetCuratedCategoryByKey(key string) (*CuratedCategory, error) {
	var category CuratedCategory
	err := DB.Where("key = ?", key).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetCuratedCategories 获取已启用分类列表
func GetCuratedCategories() ([]*CuratedCategory, error) {
	var categories []*CuratedCategory
	err := DB.Where("enabled = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&categories).Error
	return categories, err
}

// AdminListCuratedCategories 管理后台获取全部分类
func AdminListCuratedCategories() ([]*CuratedCategory, error) {
	var categories []*CuratedCategory
	err := DB.Order("sort_order ASC, id ASC").Find(&categories).Error
	return categories, err
}
