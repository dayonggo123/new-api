package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// SEOOptimizationQueueItem SEO 优化队列项
// 用于低 CTR 回流、低分内容整改、手动加入的优化任务
type SEOOptimizationQueueItem struct {
	Id          int            `json:"id" gorm:"primaryKey"`
	RecordID    int            `json:"record_id" gorm:"index"`
	ContentType string         `json:"content_type" gorm:"index"` // article / prompt / keyword
	Title       string         `json:"title"`
	Keyword     string         `json:"keyword"`
	Reason      string         `json:"reason"` // low_ctr / low_score / manual
	ScoreBefore int            `json:"score_before"`
	ScoreAfter  int            `json:"score_after"`
	Status      string         `json:"status" gorm:"default:'pending'"` // pending / processing / optimized / dismissed
	Extra       string         `json:"extra" gorm:"type:text"`          // JSON
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (item *SEOOptimizationQueueItem) Insert() error {
	return DB.Create(item).Error
}

func (item *SEOOptimizationQueueItem) Update() error {
	return DB.Save(item).Error
}

// AddSEOOptimizationQueueItem 添加优化队列项（自动去重：相同 record_id + content_type + reason + status=pending 只保留一条）
func AddSEOOptimizationQueueItem(item *SEOOptimizationQueueItem) error {
	if item.RecordID <= 0 && item.Keyword == "" {
		return errors.New("record_id or keyword is required")
	}
	if item.ContentType == "" {
		return errors.New("content_type is required")
	}
	if item.Status == "" {
		item.Status = "pending"
	}

	var existing SEOOptimizationQueueItem
	err := DB.Where("record_id = ? AND content_type = ? AND reason = ? AND status = ?", item.RecordID, item.ContentType, item.Reason, "pending").
		First(&existing).Error
	if err == nil {
		// 已存在待处理项，更新信息即可
		existing.Title = item.Title
		existing.ScoreBefore = item.ScoreBefore
		existing.Extra = item.Extra
		existing.UpdatedTime = item.UpdatedTime
		return existing.Update()
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return item.Insert()
}

// ListSEOOptimizationQueueItems 查询优化队列
func ListSEOOptimizationQueueItems(status string, limit int, offset int) ([]*SEOOptimizationQueueItem, int64, error) {
	query := DB.Model(&SEOOptimizationQueueItem{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	var items []*SEOOptimizationQueueItem
	err := query.Order("created_time desc").Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

// GetSEOOptimizationQueueItemById 按 ID 获取队列项
func GetSEOOptimizationQueueItemById(id int) (*SEOOptimizationQueueItem, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	item := SEOOptimizationQueueItem{Id: id}
	err := DB.First(&item, "id = ?", id).Error
	return &item, err
}

// UpdateSEOOptimizationQueueItemStatus 更新队列项状态与优化后分数
func UpdateSEOOptimizationQueueItemStatus(id int, status string, scoreAfter int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	updates := map[string]interface{}{
		"status":       status,
		"score_after":  scoreAfter,
		"updated_time": common.GetTimestamp(),
	}
	return DB.Model(&SEOOptimizationQueueItem{}).Where("id = ?", id).Updates(updates).Error
}
