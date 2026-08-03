package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	defaultSharedTemplatePageSize = 20
	maxSharedTemplatePageSize     = 50
)

// ========== 参数校验 ==========

// validCategories 分类白名单
var validSharedTemplateCategories = map[string]bool{
	"ecommerce":  true,
	"portrait":   true,
	"landscape":  true,
	"commercial": true,
	"creative":   true,
	"other":      true,
}

// validateSharedTemplateCategory 校验分类
func validateSharedTemplateCategory(category string) bool {
	return validSharedTemplateCategories[category]
}

// validatePlanJson 校验 planJson 为合法 JSON 且 steps 非空
func validatePlanJson(planJson string) error {
	var plan map[string]interface{}
	if err := json.Unmarshal([]byte(planJson), &plan); err != nil {
		return fmt.Errorf("planJson is not valid JSON")
	}

	// 检查 steps 字段
	steps, ok := plan["steps"]
	if !ok {
		return fmt.Errorf("planJson missing 'steps' field")
	}
	stepsArr, ok := steps.([]interface{})
	if !ok || len(stepsArr) == 0 {
		return fmt.Errorf("planJson has no valid steps")
	}

	// 检查 templateVersion >= 3
	if version, ok := plan["templateVersion"].(float64); ok {
		if int(version) < 3 {
			return fmt.Errorf("template version too old, minimum required: 3")
		}
	}

	return nil
}

// ========== 公开接口 ==========

// GetSharedTemplates 获取公开模板列表
func GetSharedTemplates(query *dto.SharedTemplateListQuery) (*dto.SharedTemplateListResponse, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.GetSharedTemplates(query, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, toSharedTemplateListItem(t))
	}

	return &dto.SharedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// GetSharedTemplateDetail 获取模板详情（只返回 approved 的）
func GetSharedTemplateDetail(templateId string) (*dto.SharedTemplateDetail, error) {
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}
	if template.Status != model.SharedTemplateStatusApproved {
		return nil, fmt.Errorf("template not found")
	}
	return toSharedTemplateDetail(template), nil
}

// ========== 用户接口 ==========

// ShareTemplate 用户分享模板
func ShareTemplate(userId int, userName string, req *dto.SharedTemplateShareRequest) (*dto.SharedTemplateCreateResponse, error) {
	// 1. 参数校验
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)

	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Name) > 200 {
		return nil, fmt.Errorf("name too long, max 200 characters")
	}
	if len(req.Description) > 500 {
		return nil, fmt.Errorf("description too long, max 500 characters")
	}
	if !validateSharedTemplateCategory(req.Category) {
		return nil, fmt.Errorf("invalid category")
	}
	if req.PlanJson == "" {
		return nil, fmt.Errorf("planJson is required")
	}

	// 2. 校验 planJson
	if err := validatePlanJson(req.PlanJson); err != nil {
		return nil, err
	}

	// 3. 生成模板 ID
	templateId := common.GetUUID()
	if len(templateId) > 32 {
		templateId = templateId[:32]
	}

	// 4. 设置默认值
	planVersion := req.PlanVersion
	if planVersion <= 0 {
		planVersion = 3
	}

	// 5. 保存到数据库
	template := &model.SharedTemplate{
		TemplateId:    templateId,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		AuthorId:      userId,
		AuthorName:    userName,
		Status:        model.SharedTemplateStatusPending,
		PlanJson:      req.PlanJson,
		PlanVersion:   planVersion,
		AppMinVersion: req.AppMinVersion,
	}

	if err := template.Insert(); err != nil {
		return nil, fmt.Errorf("failed to create template: %v", err)
	}

	return &dto.SharedTemplateCreateResponse{
		Id:        templateId,
		Status:    model.SharedTemplateStatusPending,
		CreatedAt: template.CreatedAt,
	}, nil
}

// GetMySharedTemplates 获取用户自己的模板列表
func GetMySharedTemplates(userId, page, pageSize int) (*dto.SharedTemplateMineListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.GetSharedTemplatesByAuthor(userId, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateMineItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, dto.SharedTemplateMineItem{
			Id:           t.TemplateId,
			Name:         t.Name,
			Status:       t.Status,
			RejectReason: t.RejectReason,
			Category:     t.Category,
			UseCount:     t.UseCount,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}

	return &dto.SharedTemplateMineListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// DeleteSharedTemplate 删除模板（仅作者）
func DeleteSharedTemplate(templateId string, userId int) error {
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return err
	}

	// 权限校验
	if template.AuthorId != userId {
		return fmt.Errorf("not authorized to delete this template")
	}

	return template.SoftDelete()
}

// RecordSharedTemplateUse 记录模板使用
func RecordSharedTemplateUse(templateId string, userId int) error {
	// 检查模板存在且已发布
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return err
	}
	if template.Status != model.SharedTemplateStatusApproved {
		return fmt.Errorf("template not available")
	}

	// 尝试插入使用记录（FirstOrCreate 跨 DB 兼容，UNIQUE 约束防重复计数）
	if err := model.FindOrCreateTemplateUse(templateId, userId); err != nil {
		return err
	}

	// 增加使用计数
	return model.IncrementSharedTemplateUseCount(templateId)
}

// ========== 管理员接口 ==========

// GetPendingSharedTemplates 获取待审核列表
func GetPendingSharedTemplates(page, pageSize int) (*dto.SharedTemplateListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.GetPendingSharedTemplates(page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		item := toSharedTemplateListItem(t)
		// 待审核接口返回完整 planJson（放在详情里）
		items = append(items, item)
	}

	return &dto.SharedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// AdminListSharedTemplates 管理后台获取全部模板
func AdminListSharedTemplates(query *dto.AdminSharedTemplateListQuery) (*dto.SharedTemplateListResponse, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.AdminListSharedTemplates(query.Status, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, toSharedTemplateListItem(t))
	}

	return &dto.SharedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// GetPendingSharedTemplateDetail 获取待审核模板详情（含完整 planJson）
func GetPendingSharedTemplateDetail(templateId string) (*dto.SharedTemplateDetail, error) {
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}
	return toSharedTemplateDetail(template), nil
}

// AuditSharedTemplate 审核模板
func AuditSharedTemplate(templateId string, adminId int, adminName string, req *dto.AuditRequest) (*dto.AuditResponse, error) {
	// 1. 校验 action
	action := strings.TrimSpace(req.Action)
	if action != "approve" && action != "reject" {
		return nil, fmt.Errorf("invalid action, must be 'approve' or 'reject'")
	}

	// 2. reject 时 reason 必填
	reason := strings.TrimSpace(req.Reason)
	if action == "reject" && reason == "" {
		return nil, fmt.Errorf("reason is required when rejecting")
	}
	if len(reason) > 500 {
		return nil, fmt.Errorf("reason too long, max 500 characters")
	}

	// 3. 获取模板
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}

	// 4. 状态校验
	if template.Status != model.SharedTemplateStatusPending {
		return nil, fmt.Errorf("template is not in pending status, current: %s", template.Status)
	}

	// 5. 更新状态
	newStatus := model.SharedTemplateStatusApproved
	if action == "reject" {
		newStatus = model.SharedTemplateStatusRejected
	}
	if err := model.UpdateSharedTemplateStatus(templateId, newStatus, reason); err != nil {
		return nil, fmt.Errorf("failed to update template status: %v", err)
	}

	// 6. 记录审核日志
	auditLog := &model.SharedTemplateAuditLog{
		TemplateId: templateId,
		AdminId:    adminId,
		AdminName:  adminName,
		Action:     action,
		Reason:     reason,
	}
	if err := auditLog.Insert(); err != nil {
		// 审核日志写入失败不阻塞主流程
		common.SysLog(fmt.Sprintf("failed to write audit log for template %s: %v", templateId, err))
	}

	return &dto.AuditResponse{
		Id:        templateId,
		Status:    newStatus,
		UpdatedAt: common.GetTimestamp(),
	}, nil
}

// ========== 内部转换函数 ==========

// toSharedTemplateListItem 将 model 转为列表项 DTO
func toSharedTemplateListItem(t *model.SharedTemplate) dto.SharedTemplateListItem {
	item := dto.SharedTemplateListItem{
		Id:           t.TemplateId,
		Name:         t.Name,
		Description:  t.Description,
		ThumbnailUrl: t.ThumbnailUrl,
		Category:     t.Category,
		AuthorId:     t.AuthorId,
		AuthorName:   t.AuthorName,
		Status:       t.Status,
		UseCount:     t.UseCount,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.HasAssets {
		item.AssetInfo = &dto.SharedTemplateAssetInfo{
			HasAssets:  t.HasAssets,
			AssetCount: t.AssetCount,
			TotalSize:  t.TotalSize,
			ImageCount: t.ImageCount,
			VideoCount: t.VideoCount,
		}
	}
	return item
}

// toSharedTemplateDetail 将 model 转为详情 DTO
func toSharedTemplateDetail(t *model.SharedTemplate) *dto.SharedTemplateDetail {
	return &dto.SharedTemplateDetail{
		SharedTemplateListItem: toSharedTemplateListItem(t),
		PlanJson:               t.PlanJson,
		PlanVersion:            t.PlanVersion,
		AppMinVersion:          t.AppMinVersion,
		RejectReason:           t.RejectReason,
		ApprovedAt:             t.ApprovedAt,
	}
}
