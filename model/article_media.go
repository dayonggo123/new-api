package model

import (
	"encoding/base64"
	"errors"
	"time"
)

// ArticleMedia 文章媒体文件（存储于数据库，不依赖文件系统）
type ArticleMedia struct {
	Id          int    `json:"id"`
	ArticleId   int    `json:"article_id" gorm:"index"`
	MediaType   string `json:"media_type"` // cover_image | content_image
	MimeType    string `json:"mime_type"`  // image/png, image/jpeg, ...
	Data        string `json:"-" gorm:"type:longtext"` // base64 encoded, exclude from json
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

func (am *ArticleMedia) Insert() error {
	if am.CreatedTime == 0 {
		am.CreatedTime = time.Now().Unix()
	}
	return DB.Create(am).Error
}

func GetArticleMediaById(id int) (*ArticleMedia, error) {
	var am ArticleMedia
	err := DB.First(&am, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &am, nil
}

func GetArticleMediaByArticleIdAndType(articleId int, mediaType string) ([]*ArticleMedia, error) {
	var ams []*ArticleMedia
	err := DB.Where("article_id = ? AND media_type = ?", articleId, mediaType).Find(&ams).Error
	if err != nil {
		return nil, err
	}
	return ams, nil
}

func DeleteArticleMediaById(id int) error {
	return DB.Delete(&ArticleMedia{Id: id}).Error
}

func DeleteArticleMediaByArticleId(articleId int) error {
	return DB.Where("article_id = ?", articleId).Delete(&ArticleMedia{}).Error
}

// DecodeData returns the raw bytes from base64 data
func (am *ArticleMedia) DecodeData() ([]byte, error) {
	if am.Data == "" {
		return nil, errors.New("no data")
	}
	return base64.StdEncoding.DecodeString(am.Data)
}
