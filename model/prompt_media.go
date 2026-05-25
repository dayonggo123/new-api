package model

import (
	"encoding/base64"
	"errors"
	"time"
)

// PromptMedia 提示词库媒体文件（存储于数据库，不依赖文件系统）
type PromptMedia struct {
	Id          int    `json:"id"`
	PromptId    int    `json:"prompt_id" gorm:"index"`
	MediaType   string `json:"media_type"`   // cover_image | video
	MimeType    string `json:"mime_type"`    // image/png, video/mp4, ...
	Data        string `json:"-" gorm:"type:longtext"` // base64 encoded, exclude from json
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

func (pm *PromptMedia) Insert() error {
	if pm.CreatedTime == 0 {
		pm.CreatedTime = time.Now().Unix()
	}
	return DB.Create(pm).Error
}

func GetPromptMediaById(id int) (*PromptMedia, error) {
	var pm PromptMedia
	err := DB.First(&pm, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func GetPromptMediaByPromptIdAndType(promptId int, mediaType string) (*PromptMedia, error) {
	var pm PromptMedia
	err := DB.Where("prompt_id = ? AND media_type = ?", promptId, mediaType).First(&pm).Error
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func DeletePromptMediaById(id int) error {
	return DB.Delete(&PromptMedia{Id: id}).Error
}

func DeletePromptMediaByPromptId(promptId int) error {
	return DB.Where("prompt_id = ?", promptId).Delete(&PromptMedia{}).Error
}

// DecodeData returns the raw bytes from base64 data
func (pm *PromptMedia) DecodeData() ([]byte, error) {
	if pm.Data == "" {
		return nil, errors.New("no data")
	}
	return base64.StdEncoding.DecodeString(pm.Data)
}
