package model

import (
	"errors"

	"gorm.io/gorm"
)

// EcommerceCaseCategory 案例分类配置
type EcommerceCaseCategory struct {
	Id            int            `json:"id"`
	CategoryId    string         `json:"category_id" gorm:"uniqueIndex;size:64"` // clothing, electronics, etc.
	CategoryName  string         `json:"category_name"`
	CoverImageUrl string         `json:"cover_image_url"`
	RequiresModel bool           `json:"requires_model" gorm:"default:false"`
	SortOrder     int            `json:"sort_order" gorm:"default:0"`
	Status        int            `json:"status" gorm:"default:1"` // 1=enabled, 2=disabled
	CreatedTime   int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime   int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (c *EcommerceCaseCategory) Insert() error { return DB.Create(c).Error }

func (c *EcommerceCaseCategory) Update() error {
	return DB.Model(c).Select("category_id", "category_name", "cover_image_url", "requires_model", "sort_order", "status").Updates(c).Error
}

func (c *EcommerceCaseCategory) Delete() error { return DB.Delete(c).Error }

func GetAllCaseCategories(startIdx, num int) (categories []*EcommerceCaseCategory, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Model(&EcommerceCaseCategory{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if num > 0 {
		err = tx.Order("sort_order asc, id asc").Limit(num).Offset(startIdx).Find(&categories).Error
	} else {
		err = tx.Order("sort_order asc, id asc").Find(&categories).Error
	}
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

func GetEnabledCaseCategories() ([]*EcommerceCaseCategory, error) {
	var categories []*EcommerceCaseCategory
	err := DB.Where("status = ?", 1).Order("sort_order asc, id asc").Find(&categories).Error
	return categories, err
}

func GetCaseCategoryById(id int) (*EcommerceCaseCategory, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	category := EcommerceCaseCategory{Id: id}
	err := DB.First(&category, "id = ?", id).Error
	return &category, err
}

func DeleteCaseCategoryById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	category := EcommerceCaseCategory{Id: id}
	err := DB.Where(category).First(&category).Error
	if err != nil {
		return err
	}
	return category.Delete()
}

// EcommerceCaseDetail 案例详情配置
type EcommerceCaseDetail struct {
	Id              int            `json:"id"`
	CategoryId      string         `json:"category_id" gorm:"index;size:64"`
	PlatformId      string         `json:"platform_id" gorm:"index;size:64"` // taobao, jd, etc.
	PlatformName    string         `json:"platform_name"`
	VisualFeatures  string         `json:"visual_features" gorm:"type:text"` // JSON array
	Composition     string         `json:"composition" gorm:"type:text"`     // JSON array
	Lighting        string         `json:"lighting" gorm:"type:text"`
	BackgroundStyle string         `json:"background_style" gorm:"type:text"`
	CaseReference   string         `json:"case_reference" gorm:"type:text"`
	CreatedTime     int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime     int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (d *EcommerceCaseDetail) Insert() error { return DB.Create(d).Error }

func (d *EcommerceCaseDetail) Update() error {
	return DB.Model(d).Select("category_id", "platform_id", "platform_name", "visual_features", "composition", "lighting", "background_style", "case_reference").Updates(d).Error
}

func (d *EcommerceCaseDetail) Delete() error { return DB.Delete(d).Error }

func GetCaseDetails(startIdx, num int, categoryId, platformId string) (details []*EcommerceCaseDetail, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&EcommerceCaseDetail{})
	if categoryId != "" {
		query = query.Where("category_id = ?", categoryId)
	}
	if platformId != "" {
		query = query.Where("platform_id = ?", platformId)
	}

	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if num > 0 {
		err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&details).Error
	} else {
		err = query.Order("id desc").Find(&details).Error
	}
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return details, total, nil
}

func GetCaseDetailById(id int) (*EcommerceCaseDetail, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	detail := EcommerceCaseDetail{Id: id}
	err := DB.First(&detail, "id = ?", id).Error
	return &detail, err
}

func GetCaseDetailByCategoryAndPlatform(categoryId, platformId string) (*EcommerceCaseDetail, error) {
	if categoryId == "" {
		return nil, errors.New("category_id is empty")
	}
	if platformId == "" {
		return nil, errors.New("platform_id is empty")
	}
	var detail EcommerceCaseDetail
	err := DB.Where("category_id = ? AND platform_id = ?", categoryId, platformId).First(&detail).Error
	return &detail, err
}

func DeleteCaseDetailById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	detail := EcommerceCaseDetail{Id: id}
	err := DB.Where(detail).First(&detail).Error
	if err != nil {
		return err
	}
	return detail.Delete()
}
