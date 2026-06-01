package model

import (
	"gorm.io/gorm"
)

// Popup 每日弹窗公告表
type Popup struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Title     string `json:"title" gorm:"type:varchar(255);not null"`
	Content   string `json:"content" gorm:"type:text;not null"`
	ImageUrl  string `json:"image_url" gorm:"type:varchar(512);default:''"`
	Type      string `json:"type" gorm:"type:varchar(50);default:'announcement'"`
	Enabled   bool   `json:"enabled" gorm:"default:true"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Popup) TableName() string {
	return "popups"
}

// GetActivePopup 获取当前启用的最新一条弹窗
func GetActivePopup() (*Popup, error) {
	var popup Popup
	err := DB.Where("enabled = ?", true).Order("created_at DESC").First(&popup).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &popup, nil
}

// GetPopupById 根据 ID 获取弹窗
func GetPopupById(id int) (*Popup, error) {
	var popup Popup
	err := DB.First(&popup, id).Error
	if err != nil {
		return nil, err
	}
	return &popup, nil
}

// GetAllPopups 获取所有弹窗（后台管理）
func GetAllPopups() ([]*Popup, error) {
	var popups []*Popup
	err := DB.Order("created_at DESC").Find(&popups).Error
	return popups, err
}

// CreatePopup 创建弹窗
func (popup *Popup) Insert() error {
	return DB.Create(popup).Error
}

// UpdatePopup 更新弹窗
func (popup *Popup) Update() error {
	return DB.Save(popup).Error
}

// DeletePopup 删除弹窗
func DeletePopup(id int) error {
	return DB.Delete(&Popup{}, id).Error
}
