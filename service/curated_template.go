package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	defaultCuratedPageSize = 20
	maxCuratedPageSize     = 100
)

// GetCuratedTemplates 获取公开模板列表
func GetCuratedTemplates(query *dto.CuratedTemplateListQuery) (*dto.CuratedTemplateListResponse, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = defaultCuratedPageSize
	}
	if pageSize > maxCuratedPageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxCuratedPageSize)
	}

	templates, total, err := model.GetCuratedTemplates(query, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CuratedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, toCuratedTemplateListItem(t, query.IncludeDetails))
	}

	return &dto.CuratedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// GetCuratedTemplateByTemplateId 获取单个模板详情
func GetCuratedTemplateByTemplateId(templateId string) (*dto.CuratedTemplateDetailResponse, error) {
	template, err := model.GetCuratedTemplateByTemplateId(templateId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}
	if !template.Enabled {
		return nil, fmt.Errorf("template not found")
	}

	item := toCuratedTemplateListItem(template, true)
	return &dto.CuratedTemplateDetailResponse{
		CuratedTemplateListItem: item,
		Prompt:                  template.Prompt,
	}, nil
}

// GetCuratedTemplateExecutionPlan 获取模板执行计划
func GetCuratedTemplateExecutionPlan(templateId string) (json.RawMessage, error) {
	template, err := model.GetCuratedTemplateByTemplateId(templateId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}
	if !template.Enabled {
		return nil, fmt.Errorf("template not found")
	}
	if len(template.ExecutionPlan) == 0 {
		return json.RawMessage("{}"), nil
	}
	return template.ExecutionPlan, nil
}

// AdminListCuratedTemplates 管理后台获取模板列表
func AdminListCuratedTemplates(page, pageSize int) (*dto.CuratedTemplateListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultCuratedPageSize
	}
	if pageSize > maxCuratedPageSize {
		pageSize = maxCuratedPageSize
	}

	templates, total, err := model.AdminListCuratedTemplates(page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CuratedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, toCuratedTemplateListItem(t, true))
	}

	return &dto.CuratedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// AdminCreateTemplate 管理后台创建模板
func AdminCreateTemplate(req *dto.AdminUpsertTemplateRequest) (*model.CuratedTemplate, error) {
	if req.TemplateId == "" || req.Title == "" {
		return nil, fmt.Errorf("template id and title are required")
	}

	existing, err := model.GetCuratedTemplateByTemplateId(req.TemplateId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("template id already exists")
	}

	template := &model.CuratedTemplate{
		TemplateId:      req.TemplateId,
		Title:           req.Title,
		Category:        req.Category,
		CoverImageUrl:   req.CoverImageUrl,
		PreviewMediaUrl: req.PreviewMediaUrl,
		Description:     req.Description,
		Prompt:          req.Prompt,
		InputSlots:      req.InputSlots,
		Params:          req.Params,
		ExecutionPlan:   req.ExecutionPlan,
		EstimatedPrice:  req.EstimatedPrice,
		SortOrder:       req.SortOrder,
		Enabled:         req.Enabled,
		HotScore:        req.HotScore,
	}

	if err := template.Insert(); err != nil {
		return nil, err
	}
	_ = InvalidateCuratedTemplateCache()
	return template, nil
}

// AdminUpdateTemplate 管理后台更新模板
func AdminUpdateTemplate(id int, req *dto.AdminUpsertTemplateRequest) error {
	if id <= 0 {
		return fmt.Errorf("invalid template id")
	}

	template, err := model.GetCuratedTemplateById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("template not found")
		}
		return err
	}

	if req.TemplateId != "" && req.TemplateId != template.TemplateId {
		existing, err := model.GetCuratedTemplateByTemplateId(req.TemplateId)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existing != nil {
			return fmt.Errorf("template id already exists")
		}
		template.TemplateId = req.TemplateId
	}

	template.Title = req.Title
	template.Category = req.Category
	template.CoverImageUrl = req.CoverImageUrl
	template.PreviewMediaUrl = req.PreviewMediaUrl
	template.Description = req.Description
	template.Prompt = req.Prompt
	template.InputSlots = req.InputSlots
	template.Params = req.Params
	template.ExecutionPlan = req.ExecutionPlan
	template.EstimatedPrice = req.EstimatedPrice
	template.SortOrder = req.SortOrder
	template.Enabled = req.Enabled
	template.HotScore = req.HotScore

	if err := template.Update(); err != nil {
		return err
	}
	_ = InvalidateCuratedTemplateCache()
	return nil
}

// AdminDeleteTemplate 管理后台删除模板
func AdminDeleteTemplate(id int) error {
	template, err := model.GetCuratedTemplateById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("template not found")
		}
		return err
	}
	if err := template.Delete(); err != nil {
		return err
	}
	_ = InvalidateCuratedTemplateCache()
	return nil
}

// AdminUpdateTemplateStatus 管理后台更新模板启用状态
func AdminUpdateTemplateStatus(id int, enabled bool) error {
	if err := model.AdminUpdateStatus(id, enabled); err != nil {
		return err
	}
	_ = InvalidateCuratedTemplateCache()
	return nil
}

// toCuratedTemplateListItem 将模型转换为列表项 DTO
func toCuratedTemplateListItem(template *model.CuratedTemplate, includeDetails bool) dto.CuratedTemplateListItem {
	item := dto.CuratedTemplateListItem{
		Id:              template.TemplateId,
		Title:           template.Title,
		Category:        template.Category,
		CoverImageUrl:   template.CoverImageUrl,
		PreviewMediaUrl: template.PreviewMediaUrl,
		Description:     template.Description,
		EstimatedPrice:  template.EstimatedPrice,
		InputSlots:      template.InputSlots,
		Params:          template.Params,
		SortOrder:       template.SortOrder,
		Enabled:         template.Enabled,
		HotScore:        template.HotScore,
		CreatedAt:       template.CreatedAt,
		UpdatedAt:       template.UpdatedAt,
	}
	if includeDetails {
		item.ExecutionPlan = template.ExecutionPlan
	}
	return item
}
