package service

import "fmt"

const (
	curatedTemplateCacheNamespace   = "new-api:curated_template:v1"
	curatedCategoryCacheNamespace   = "new-api:curated_category:v1"
	curatedTemplateListCacheNamespace = "new-api:curated_template_list:v1"
)

// CuratedTemplateCacheKey 生成模板详情缓存 key
func CuratedTemplateCacheKey(templateId string) string {
	return fmt.Sprintf("%s:template:%s", curatedTemplateCacheNamespace, templateId)
}

// CuratedCategoryListCacheKey 生成分类列表缓存 key
func CuratedCategoryListCacheKey() string {
	return fmt.Sprintf("%s:categories:enabled", curatedCategoryCacheNamespace)
}

// CuratedTemplateListCacheKey 生成模板列表查询缓存 key
func CuratedTemplateListCacheKey(category, keyword, sortBy string, page, pageSize int, includeDetails bool) string {
	return fmt.Sprintf("%s:list:%s:%s:%s:%d:%d:%t", curatedTemplateListCacheNamespace, category, keyword, sortBy, page, pageSize, includeDetails)
}

// InvalidateCuratedTemplateCache 使模板相关缓存失效（P1 接入 cachex）
func InvalidateCuratedTemplateCache() error {
	return nil
}

// InvalidateCuratedCategoryCache 使分类列表缓存失效（P1 接入 cachex）
func InvalidateCuratedCategoryCache() error {
	return nil
}
