package dto

import "encoding/json"

// CuratedTemplateListItem 模板列表单条记录
type CuratedTemplateListItem struct {
	Id              string          `json:"id"`
	Title           string          `json:"title"`
	Category        string          `json:"category"`
	CoverImageUrl   string          `json:"coverImageUrl"`
	PreviewMediaUrl string          `json:"previewMediaUrl"`
	Description     string          `json:"description"`
	EstimatedPrice  float64         `json:"estimatedPrice"`
	InputSlots      json.RawMessage `json:"inputSlots"`
	Params          json.RawMessage `json:"params"`
	ExecutionPlan   json.RawMessage `json:"executionPlan,omitempty"`
	SortOrder       int             `json:"sortOrder"`
	Enabled         bool            `json:"enabled"`
	HotScore        int             `json:"hotScore"`
	CreatedAt       int64           `json:"createdAt"`
	UpdatedAt       int64           `json:"updatedAt"`
}

// CuratedTemplateListResponse 模板列表分页响应
type CuratedTemplateListResponse struct {
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
	List     []CuratedTemplateListItem `json:"list"`
}

// CuratedTemplateDetailResponse 模板详情响应（在列表项基础上增加 prompt）
type CuratedTemplateDetailResponse struct {
	CuratedTemplateListItem
	Prompt string `json:"prompt"`
}

// CuratedCategoryResponse 分类响应
type CuratedCategoryResponse struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	IconUrl   string `json:"iconUrl"`
	SortOrder int    `json:"sortOrder"`
	Enabled   bool   `json:"enabled"`
}

// CuratedCategoryListResponse 分类列表响应
type CuratedCategoryListResponse struct {
	Categories []CuratedCategoryResponse `json:"categories"`
}
