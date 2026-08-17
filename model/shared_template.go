package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"gorm.io/gorm"
)

// SharedTemplate 工作流模板云端分享表
type SharedTemplate struct {
	Id            int            `json:"-" gorm:"primaryKey;autoIncrement"`
	TemplateId    string         `json:"id" gorm:"column:template_id;size:32;uniqueIndex"`
	Name          string         `json:"name" gorm:"size:200;not null"`
	Description   string         `json:"description" gorm:"type:text"`
	Category      string         `json:"category" gorm:"size:32;not null;index:idx_sht_status_category"`
	AuthorId      int            `json:"authorId" gorm:"column:author_id;not null;index"`
	AuthorName    string         `json:"authorName" gorm:"column:author_name;size:100;not null"`
	Status        string         `json:"status" gorm:"size:20;not null;default:'pending';index:idx_sht_status_category"`
	// Hidden 管理员隐藏标记：hidden=true 时下游（模板市场列表/详情/使用）不展示，数据保留可恢复
	Hidden        bool           `json:"hidden" gorm:"column:hidden;default:false"`
	RejectReason  string         `json:"rejectReason,omitempty" gorm:"column:reject_reason;type:text"`
	PlanJson      string         `json:"planJson" gorm:"column:plan_json;type:longtext;not null"`
	PlanVersion   int            `json:"planVersion" gorm:"column:plan_version;default:3"`
	AppMinVersion string         `json:"appMinVersion,omitempty" gorm:"column:app_min_version;size:20"`
	ManifestJson  string         `json:"manifestJson,omitempty" gorm:"column:manifest_json;type:longtext"`
	AssetCount    int            `json:"assetCount" gorm:"column:asset_count;default:0"`
	ImageCount    int            `json:"imageCount" gorm:"column:image_count;default:0"`
	VideoCount    int            `json:"videoCount" gorm:"column:video_count;default:0"`
	TotalSize     int64          `json:"totalSize" gorm:"column:total_size;default:0"`
	HasAssets     bool           `json:"hasAssets" gorm:"column:has_assets;default:false"`
	// ThumbnailUrl 模板封面 URL。使用 TEXT 而非 VARCHAR：
	// R2 presigned URL（含 AWS 签名参数）长度可超过 500 字符（实测 513+），
	// VARCHAR(500) 在 MySQL 上会报 Error 1406 Data too long。
	// 新分享入库前会被规范化为 r2://bucket/key 短路径（见 service.ShareTemplate），
	// 此列同时兼容历史完整 URL 与第三方 CDN URL。
	ThumbnailUrl  string         `json:"thumbnailUrl,omitempty" gorm:"column:thumbnail_url;type:text"`
	ThumbnailType string         `json:"thumbnailType,omitempty" gorm:"column:thumbnail_type;size:20"`
	UseCount      int            `json:"useCount" gorm:"column:use_count;default:0;index"`
	CreatedAt     int64          `json:"createdAt" gorm:"column:created_at;autoCreateTime;index"`
	UpdatedAt     int64          `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
	ApprovedAt    int64          `json:"approvedAt,omitempty" gorm:"column:approved_at;default:0"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (SharedTemplate) TableName() string {
	return "shared_templates"
}

// ========== 状态常量 ==========

const (
	SharedTemplateStatusPending  = "pending"
	SharedTemplateStatusApproved = "approved"
	SharedTemplateStatusRejected = "rejected"
)

// ========== CRUD ==========

func (t *SharedTemplate) Insert() error {
	return DB.Create(t).Error
}

func (t *SharedTemplate) Update() error {
	return DB.Save(t).Error
}

func (t *SharedTemplate) SoftDelete() error {
	return DB.Delete(t).Error
}

// ========== 查询 ==========

func GetSharedTemplateByTemplateId(templateId string) (*SharedTemplate, error) {
	var t SharedTemplate
	err := DB.Where("template_id = ?", templateId).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func GetSharedTemplates(query *dto.SharedTemplateListQuery, page, pageSize int) ([]*SharedTemplate, int64, error) {
	var templates []*SharedTemplate
	var total int64

	db := DB.Model(&SharedTemplate{}).Where("status = ? AND hidden = ?", SharedTemplateStatusApproved, false)

	if query.Category != "" {
		db = db.Where("category = ?", query.Category)
	}
	if query.Keyword != "" {
		keyword := "%" + strings.TrimSpace(query.Keyword) + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	switch query.Sort {
	case "newest":
		db = db.Order("created_at DESC")
	case "popular":
		db = db.Order("use_count DESC, created_at DESC")
	default:
		db = db.Order("created_at DESC")
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

func GetSharedTemplatesByAuthor(authorId, page, pageSize int) ([]*SharedTemplate, int64, error) {
	var templates []*SharedTemplate
	var total int64

	db := DB.Model(&SharedTemplate{}).Where("author_id = ?", authorId).Order("created_at DESC")

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

func GetPendingSharedTemplates(page, pageSize int) ([]*SharedTemplate, int64, error) {
	var templates []*SharedTemplate
	var total int64

	db := DB.Model(&SharedTemplate{}).Where("status = ?", SharedTemplateStatusPending).Order("created_at ASC")

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

func AdminListSharedTemplates(statusFilter string, page, pageSize int) ([]*SharedTemplate, int64, error) {
	var templates []*SharedTemplate
	var total int64

	db := DB.Model(&SharedTemplate{})
	if statusFilter != "" {
		db = db.Where("status = ?", statusFilter)
	}
	db = db.Order("created_at DESC")

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

func UpdateSharedTemplateStatus(templateId, status, rejectReason string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": common.GetTimestamp(),
	}
	if status == SharedTemplateStatusRejected && rejectReason != "" {
		updates["reject_reason"] = rejectReason
	} else {
		updates["reject_reason"] = ""
	}
	if status == SharedTemplateStatusApproved {
		updates["approved_at"] = common.GetTimestamp()
	}
	return DB.Model(&SharedTemplate{}).Where("template_id = ?", templateId).Updates(updates).Error
}

// UpdateSharedTemplateHidden 设置模板隐藏状态（隐藏后下游不展示，数据保留）
func UpdateSharedTemplateHidden(templateId string, hidden bool) error {
	return DB.Model(&SharedTemplate{}).Where("template_id = ?", templateId).
		Updates(map[string]interface{}{
			"hidden":     hidden,
			"updated_at": common.GetTimestamp(),
		}).Error
}

// DeleteSharedTemplatePermanent 彻底删除模板（物理删除，不可恢复），
// 并同步清理其使用记录。审计日志保留以便追溯。
func DeleteSharedTemplatePermanent(templateId string) error {
	if err := DB.Unscoped().Where("template_id = ?", templateId).Delete(&SharedTemplate{}).Error; err != nil {
		return err
	}
	return DB.Where("template_id = ?", templateId).Delete(&SharedTemplateUse{}).Error
}

func IncrementSharedTemplateUseCount(templateId string) error {
	return DB.Model(&SharedTemplate{}).Where("template_id = ?", templateId).
		UpdateColumn("use_count", gorm.Expr("use_count + 1")).Error
}

// ========== SharedTemplateUse ==========

type SharedTemplateUse struct {
	Id         int    `json:"-" gorm:"primaryKey;autoIncrement"`
	TemplateId string `json:"templateId" gorm:"column:template_id;size:32;not null;index"`
	UserId     int    `json:"userId" gorm:"column:user_id;not null;uniqueIndex:idx_sht_template_user"`
	UsedAt     int64  `json:"usedAt" gorm:"column:used_at;autoCreateTime"`
}

func (SharedTemplateUse) TableName() string {
	return "shared_template_uses"
}

// FindOrCreateTemplateUse inserts a usage record, silently skipping if already exists.
// Uses FirstOrCreate to be cross-DB compatible (SQLite / MySQL / PostgreSQL).
func FindOrCreateTemplateUse(templateId string, userId int) error {
	use := SharedTemplateUse{
		TemplateId: templateId,
		UserId:     userId,
	}
	return DB.Where("template_id = ? AND user_id = ?", templateId, userId).
		FirstOrCreate(&use).Error
}

// ========== SharedTemplateAuditLog ==========

type SharedTemplateAuditLog struct {
	Id         int    `json:"-" gorm:"primaryKey;autoIncrement"`
	TemplateId string `json:"templateId" gorm:"column:template_id;size:32;not null;index"`
	AdminId    int    `json:"adminId" gorm:"column:admin_id;not null"`
	AdminName  string `json:"adminName" gorm:"column:admin_name;size:128"`
	Action     string `json:"action" gorm:"size:20;not null"`
	Reason     string `json:"reason,omitempty" gorm:"type:text"`
	CreatedAt  int64  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
}

func (SharedTemplateAuditLog) TableName() string {
	return "shared_template_audit_logs"
}

func (l *SharedTemplateAuditLog) Insert() error {
	return DB.Create(l).Error
}
