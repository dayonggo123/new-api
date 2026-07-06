package dto

import "encoding/json"

// CuratedTemplateListQuery 模板列表公开查询参数
type CuratedTemplateListQuery struct {
	Category       string `form:"category"`
	Keyword        string `form:"keyword"`
	Page           int    `form:"page"`
	PageSize       int    `form:"pageSize"`
	SortBy         string `form:"sortBy"`
	IncludeDetails bool   `form:"includeDetails"`
}

// AdminUpsertTemplateRequest 管理后台创建/更新模板请求
type AdminUpsertTemplateRequest struct {
	TemplateId      string          `json:"id"`
	Title           string          `json:"title"`
	Category        string          `json:"category"`
	CoverImageUrl   string          `json:"coverImageUrl"`
	PreviewMediaUrl string          `json:"previewMediaUrl"`
	Description     string          `json:"description"`
	Prompt          string          `json:"prompt"`
	InputSlots      json.RawMessage `json:"inputSlots"`
	Params          json.RawMessage `json:"params"`
	ExecutionPlan   json.RawMessage `json:"executionPlan"`
	EstimatedPrice  float64         `json:"estimatedPrice"`
	SortOrder       int             `json:"sortOrder"`
	Enabled         bool            `json:"enabled"`
	HotScore        int             `json:"hotScore"`
}

// AdminUpdateTemplateStatusRequest 管理后台更新模板状态请求
type AdminUpdateTemplateStatusRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// AdminUpsertCategoryRequest 管理后台创建/更新分类请求
type AdminUpsertCategoryRequest struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	IconUrl   string `json:"iconUrl"`
	SortOrder int    `json:"sortOrder"`
	Enabled   bool   `json:"enabled"`
}
