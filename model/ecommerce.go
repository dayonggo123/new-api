package model

import (
	"errors"

	"gorm.io/gorm"
)

// EcommerceModelPose 模特姿势配置
type EcommerceModelPose struct {
	Id            int            `json:"id"`
	PoseId        string         `json:"pose_id" gorm:"uniqueIndex;size:64"` // no_model, hold_product, etc.
	Label         string         `json:"label"`
	Description   string         `json:"description" gorm:"type:text"`
	CoverImageUrl string         `json:"cover_image_url"`
	SortOrder     int            `json:"sort_order" gorm:"default:0"`
	Status        int            `json:"status" gorm:"default:1"` // 1=enabled, 2=disabled
	CreatedTime   int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime   int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (p *EcommerceModelPose) Insert() error { return DB.Create(p).Error }

func (p *EcommerceModelPose) Update() error {
	return DB.Model(p).Select("pose_id", "label", "description", "cover_image_url", "sort_order", "status").Updates(p).Error
}

func (p *EcommerceModelPose) Delete() error { return DB.Delete(p).Error }

func GetAllModelPoses(startIdx, num int) (poses []*EcommerceModelPose, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Model(&EcommerceModelPose{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if num > 0 {
		err = tx.Order("sort_order asc, id asc").Limit(num).Offset(startIdx).Find(&poses).Error
	} else {
		err = tx.Order("sort_order asc, id asc").Find(&poses).Error
	}
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return poses, total, nil
}

func GetEnabledModelPoses() ([]*EcommerceModelPose, error) {
	var poses []*EcommerceModelPose
	err := DB.Where("status = ?", 1).Order("sort_order asc, id asc").Find(&poses).Error
	return poses, err
}

func GetModelPoseById(id int) (*EcommerceModelPose, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	pose := EcommerceModelPose{Id: id}
	err := DB.First(&pose, "id = ?", id).Error
	return &pose, err
}

func DeleteModelPoseById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	pose := EcommerceModelPose{Id: id}
	err := DB.Where(pose).First(&pose).Error
	if err != nil {
		return err
	}
	return pose.Delete()
}
