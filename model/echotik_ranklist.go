package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EchotikVideoRanklistSnapshot EchoTik 视频榜单分页缓存快照
// 每行对应一次上游请求的一个分页结果。
type EchotikVideoRanklistSnapshot struct {
	Id                uint64         `json:"id" gorm:"primaryKey;autoIncrement"`
	Date              string         `json:"date" gorm:"type:varchar(10);not null;uniqueIndex:idx_echotik_ranklist_uq,priority:1"`
	Region            string         `json:"region" gorm:"type:varchar(16);not null;uniqueIndex:idx_echotik_ranklist_uq,priority:2"`
	VideoRankField    int            `json:"video_rank_field" gorm:"not null;uniqueIndex:idx_echotik_ranklist_uq,priority:3"`
	RankType          int            `json:"rank_type" gorm:"not null;uniqueIndex:idx_echotik_ranklist_uq,priority:4"`
	ProductCategoryID string         `json:"product_category_id" gorm:"type:varchar(64);default:'';uniqueIndex:idx_echotik_ranklist_uq,priority:5"`
	CreatedByAI       string         `json:"created_by_ai" gorm:"type:varchar(8);default:'';uniqueIndex:idx_echotik_ranklist_uq,priority:6"`
	PageNum           int            `json:"page_num" gorm:"not null;uniqueIndex:idx_echotik_ranklist_uq,priority:7"`
	PageSize          int            `json:"page_size" gorm:"not null;uniqueIndex:idx_echotik_ranklist_uq,priority:8"`

	// 原始数据
	RawResponse string `json:"-" gorm:"type:longtext;not null"` // 上游完整 JSON 响应
	Items       string `json:"-" gorm:"type:longtext"`          // data 数组 JSON，便于后续按 video_id 检索
	ItemCount   int    `json:"-"`                               // data 长度

	// 上游元信息
	UpstreamCode      int    `json:"-"`
	UpstreamMessage   string `json:"-" gorm:"type:varchar(255)"`
	UpstreamRequestID string `json:"-" gorm:"type:varchar(128);index"`

	// 缓存控制
	FetchedAt int64 `json:"-" gorm:"bigint;not null;index"`
	ExpiresAt int64 `json:"-" gorm:"bigint;not null;index"`

	// 时间戳
	CreatedAt int64          `json:"created_at" gorm:"bigint"`
	UpdatedAt int64          `json:"updated_at" gorm:"bigint"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (EchotikVideoRanklistSnapshot) TableName() string {
	return "echotik_video_ranklist_snapshots"
}

// EchotikRanklistKey 用于按 8 维参数唯一查询或写入缓存快照。
type EchotikRanklistKey struct {
	Date              string
	Region            string
	VideoRankField    int
	RankType          int
	ProductCategoryID string
	CreatedByAI       string
	PageNum           int
	PageSize          int
}

// GetEchotikRanklistSnapshot 按参数查询缓存快照（不过期过滤）。
func GetEchotikRanklistSnapshot(key *EchotikRanklistKey) (*EchotikVideoRanklistSnapshot, error) {
	if key == nil {
		return nil, errors.New("echotik ranklist key is nil")
	}

	var snapshot EchotikVideoRanklistSnapshot
	err := DB.Where("date = ?", key.Date).
		Where("region = ?", key.Region).
		Where("video_rank_field = ?", key.VideoRankField).
		Where("rank_type = ?", key.RankType).
		Where("product_category_id = ?", key.ProductCategoryID).
		Where("created_by_ai = ?", key.CreatedByAI).
		Where("page_num = ?", key.PageNum).
		Where("page_size = ?", key.PageSize).
		First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &snapshot, nil
}

// GetFreshEchotikRanklistSnapshot 查询未过期的缓存快照。
func GetFreshEchotikRanklistSnapshot(key *EchotikRanklistKey) (*EchotikVideoRanklistSnapshot, error) {
	if key == nil {
		return nil, errors.New("echotik ranklist key is nil")
	}

	now := time.Now().Unix()
	var snapshot EchotikVideoRanklistSnapshot
	err := DB.Where("date = ?", key.Date).
		Where("region = ?", key.Region).
		Where("video_rank_field = ?", key.VideoRankField).
		Where("rank_type = ?", key.RankType).
		Where("product_category_id = ?", key.ProductCategoryID).
		Where("created_by_ai = ?", key.CreatedByAI).
		Where("page_num = ?", key.PageNum).
		Where("page_size = ?", key.PageSize).
		Where("expires_at > ?", now).
		First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &snapshot, nil
}

// UpsertEchotikRanklistSnapshot 使用 ON CONFLICT UPDATE 语义写入或覆盖缓存快照。
func UpsertEchotikRanklistSnapshot(snapshot *EchotikVideoRanklistSnapshot) error {
	if snapshot == nil {
		return errors.New("echotik ranklist snapshot is nil")
	}

	now := time.Now().Unix()
	if snapshot.CreatedAt == 0 {
		snapshot.CreatedAt = now
	}
	snapshot.UpdatedAt = now

	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "date"},
			{Name: "region"},
			{Name: "video_rank_field"},
			{Name: "rank_type"},
			{Name: "product_category_id"},
			{Name: "created_by_ai"},
			{Name: "page_num"},
			{Name: "page_size"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"raw_response",
			"items",
			"item_count",
			"upstream_code",
			"upstream_message",
			"upstream_request_id",
			"fetched_at",
			"expires_at",
			"updated_at",
		}),
	}).Create(snapshot).Error
}

// DeleteEchotikRanklistSnapshotsBefore 物理删除 fetched_at 早于 cutoff 的快照。
func DeleteEchotikRanklistSnapshotsBefore(cutoff int64) (int64, error) {
	result := DB.Unscoped().Where("fetched_at < ?", cutoff).Delete(&EchotikVideoRanklistSnapshot{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
