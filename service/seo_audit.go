package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// SEOAuditResult SEO 审计结果
type SEOAuditResult struct {
	OverallScore    int              `json:"overall_score"`    // 0-100 总分
	RecordID        int              `json:"record_id"`
	RecordType      string           `json:"record_type"`      // article / prompt
	Title           string           `json:"title"`
	DimensionScores []DimensionScore `json:"dimension_scores"` // 各维度得分
	Issues          []SEOIssue       `json:"issues"`           // 发现的问题
	Suggestions     []string         `json:"suggestions"`      // 改进建议
	// 兼容 article_seo_audit.go 的字段
	Categories     []string `json:"categories"`      // 内容分类
	CriticalIssues []string `json:"critical_issues"` // 关键问题
	QuickWins      []string `json:"quick_wins"`      // 快速优化项
}

// DimensionScore 维度评分
type DimensionScore struct {
	Name        string `json:"name"`
	Score       int    `json:"score"`        // 0-100
	Weight      int    `json:"weight"`       // 权重
	Status      string `json:"status"`       // pass / warning / fail
	Description string `json:"description"`
}

// SEOIssue SEO 问题
type SEOIssue struct {
	Type        string `json:"type"`        // error / warning / info
	Field       string `json:"field"`       // 问题字段
	Message     string `json:"message"`     // 问题描述
	Suggestion  string `json:"suggestion"`  // 修复建议
	AutoFixable bool   `json:"auto_fixable"` // 是否可自动修复
}

// AuditSEO 对指定内容进行 SEO 审计
func AuditSEO(recordType string, recordID int) (*SEOAuditResult, error) {
	switch recordType {
	case "article":
		return auditArticle(recordID)
	case "prompt":
		return auditPrompt(recordID)
	default:
		return nil, fmt.Errorf("unsupported record type: %s", recordType)
	}
}

// auditArticle 审计文章 SEO
func auditArticle(recordID int) (*SEOAuditResult, error) {
	article, err := model.GetArticleById(recordID)
	if err != nil {
		return nil, err
	}

	result := &SEOAuditResult{
		RecordID:   recordID,
		RecordType: "article",
		Title:      article.Title,
		Issues:     []SEOIssue{},
		Suggestions: []string{},
	}

	dimensions := []DimensionScore{}

	// 1. Title 审计
	titleScore, titleIssues := auditTitle(article.Title)
	dimensions = append(dimensions, DimensionScore{
		Name:        "Title",
		Score:       titleScore,
		Weight:      20,
		Status:      scoreStatus(titleScore),
		Description: "页面标题长度和关键词覆盖",
	})
	result.Issues = append(result.Issues, titleIssues...)

	// 2. Content 审计
	contentScore, contentIssues := auditContent(article.Content, article.Title)
	dimensions = append(dimensions, DimensionScore{
		Name:        "Content",
		Score:       contentScore,
		Weight:      25,
		Status:      scoreStatus(contentScore),
		Description: "内容质量和长度",
	})
	result.Issues = append(result.Issues, contentIssues...)

	// 3. SEO Fields 审计
	seoScore, seoIssues := auditSEOFields(article.SeoKeywords, article.Intro, article.Faq)
	dimensions = append(dimensions, DimensionScore{
		Name:        "SEO Fields",
		Score:       seoScore,
		Weight:      20,
		Status:      scoreStatus(seoScore),
		Description: "SEO 关键词、介绍、FAQ",
	})
	result.Issues = append(result.Issues, seoIssues...)

	// 4. Slug 审计
	slugScore, slugIssues := auditSlug(article.Slug, article.Title)
	dimensions = append(dimensions, DimensionScore{
		Name:        "URL/Slug",
		Score:       slugScore,
		Weight:      10,
		Status:      scoreStatus(slugScore),
		Description: "URL 友好度和关键词",
	})
	result.Issues = append(result.Issues, slugIssues...)

	// 5. Multi-language 审计
	i18nScore, i18nIssues := auditI18n(article.I18n, article.SeoI18n, article.GeoBlocksI18n)
	dimensions = append(dimensions, DimensionScore{
		Name:        "Multi-language",
		Score:       i18nScore,
		Weight:      15,
		Status:      scoreStatus(i18nScore),
		Description: "多语言覆盖情况",
	})
	result.Issues = append(result.Issues, i18nIssues...)

	// 6. GEO Blocks 审计
	geoScore, geoIssues := auditGeoBlocks(article.GeoBlocks)
	dimensions = append(dimensions, DimensionScore{
		Name:        "GEO Blocks",
		Score:       geoScore,
		Weight:      10,
		Status:      scoreStatus(geoScore),
		Description: "GEO 结构化内容",
	})
	result.Issues = append(result.Issues, geoIssues...)

	result.DimensionScores = dimensions
	result.OverallScore = calculateOverallScore(dimensions)

	// 生成改进建议
	result.Suggestions = generateSuggestions(result.Issues)

	return result, nil
}

// auditPrompt 审计 Prompt SEO
func auditPrompt(recordID int) (*SEOAuditResult, error) {
	prompt, err := model.GetPromptById(recordID)
	if err != nil {
		return nil, err
	}

	result := &SEOAuditResult{
		RecordID:   recordID,
		RecordType: "prompt",
		Title:      prompt.Title,
		Issues:     []SEOIssue{},
		Suggestions: []string{},
	}

	dimensions := []DimensionScore{}

	// 1. Title 审计
	titleScore, titleIssues := auditTitle(prompt.Title)
	dimensions = append(dimensions, DimensionScore{
		Name:        "Title",
		Score:       titleScore,
		Weight:      20,
		Status:      scoreStatus(titleScore),
		Description: "页面标题长度和关键词覆盖",
	})
	result.Issues = append(result.Issues, titleIssues...)

	// 2. Content 审计
	contentScore, contentIssues := auditPromptContent(prompt.Content, prompt.Title)
	dimensions = append(dimensions, DimensionScore{
		Name:        "Content",
		Score:       contentScore,
		Weight:      25,
		Status:      scoreStatus(contentScore),
		Description: "提示词内容质量",
	})
	result.Issues = append(result.Issues, contentIssues...)

	// 3. SEO Fields 审计
	seoScore, seoIssues := auditSEOFields(prompt.SeoKeywords, prompt.Intro, prompt.Faq)
	dimensions = append(dimensions, DimensionScore{
		Name:        "SEO Fields",
		Score:       seoScore,
		Weight:      20,
		Status:      scoreStatus(seoScore),
		Description: "SEO 关键词、介绍、FAQ",
	})
	result.Issues = append(result.Issues, seoIssues...)

	// 4. Slug 审计
	slugScore, slugIssues := auditSlug(prompt.Slug, prompt.Title)
	dimensions = append(dimensions, DimensionScore{
		Name:        "URL/Slug",
		Score:       slugScore,
		Weight:      10,
		Status:      scoreStatus(slugScore),
		Description: "URL 友好度和关键词",
	})
	result.Issues = append(result.Issues, slugIssues...)

	// 5. Multi-language 审计
	i18nScore, i18nIssues := auditI18n(prompt.I18n, prompt.TitleI18n, prompt.GeoBlocksI18n)
	dimensions = append(dimensions, DimensionScore{
		Name:        "Multi-language",
		Score:       i18nScore,
		Weight:      15,
		Status:      scoreStatus(i18nScore),
		Description: "多语言覆盖情况",
	})
	result.Issues = append(result.Issues, i18nIssues...)

	// 6. GEO Blocks 审计
	geoScore, geoIssues := auditGeoBlocks(prompt.GeoBlocks)
	dimensions = append(dimensions, DimensionScore{
		Name:        "GEO Blocks",
		Score:       geoScore,
		Weight:      10,
		Status:      scoreStatus(geoScore),
		Description: "GEO 结构化内容",
	})
	result.Issues = append(result.Issues, geoIssues...)

	result.DimensionScores = dimensions
	result.OverallScore = calculateOverallScore(dimensions)
	result.Suggestions = generateSuggestions(result.Issues)

	return result, nil
}

// ==================== 审计维度实现 ====================

func auditTitle(title string) (int, []SEOIssue) {
	score := 100
	var issues []SEOIssue

	if title == "" {
		score = 0
		issues = append(issues, SEOIssue{
			Type:        "error",
			Field:       "title",
			Message:     "标题为空",
			Suggestion:  "填写一个包含目标关键词的吸引人的标题",
			AutoFixable: false,
		})
		return score, issues
	}

	if len(title) < 10 {
		score -= 30
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "title",
			Message:     fmt.Sprintf("标题过短 (%d 字符)，建议 20-60 字符", len(title)),
			Suggestion:  "扩展标题，加入更多描述性词语",
			AutoFixable: false,
		})
	} else if len(title) > 70 {
		score -= 20
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "title",
			Message:     fmt.Sprintf("标题过长 (%d 字符)，建议 20-60 字符", len(title)),
			Suggestion:  "缩短标题，确保核心关键词在前 60 字符内",
			AutoFixable: false,
		})
	}

	if strings.Contains(title, "|") || strings.Contains(title, "-") {
		score += 5 // 品牌分隔符加分
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, issues
}

func auditContent(content, title string) (int, []SEOIssue) {
	score := 100
	var issues []SEOIssue

	if content == "" {
		score = 0
		issues = append(issues, SEOIssue{
			Type:        "error",
			Field:       "content",
			Message:     "内容为空",
			Suggestion:  "添加实质性内容",
			AutoFixable: false,
		})
		return score, issues
	}

	wordCount := len(strings.Fields(content))
	if wordCount < 300 {
		score -= 40
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "content",
			Message:     fmt.Sprintf("内容过短 (%d 词)，建议至少 500 词", wordCount),
			Suggestion:  "扩展内容深度，添加更多细节和例子",
			AutoFixable: false,
		})
	} else if wordCount < 800 {
		score -= 15
		issues = append(issues, SEOIssue{
			Type:        "info",
			Field:       "content",
			Message:     fmt.Sprintf("内容较短 (%d 词)，建议 1000+ 词以获得更好排名", wordCount),
			Suggestion:  "考虑扩展内容",
			AutoFixable: false,
		})
	} else if wordCount >= 1500 {
		score += 10
	}

	// 检查标题关键词是否在内容中出现
	titleWords := extractWords(title)
	contentLower := strings.ToLower(content)
	for _, word := range titleWords {
		if len(word) > 3 && strings.Contains(contentLower, strings.ToLower(word)) {
			score += 5
			break
		}
	}

	if score > 100 {
		score = 100
	}

	return score, issues
}

func auditPromptContent(content, title string) (int, []SEOIssue) {
	score := 100
	var issues []SEOIssue

	if content == "" {
		score = 0
		issues = append(issues, SEOIssue{
			Type:        "error",
			Field:       "content",
			Message:     "提示词内容为空",
			Suggestion:  "添加提示词文本",
			AutoFixable: false,
		})
		return score, issues
	}

	wordCount := len(strings.Fields(content))
	if wordCount < 50 {
		score -= 30
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "content",
			Message:     fmt.Sprintf("提示词过短 (%d 词)，建议至少 100 词", wordCount),
			Suggestion:  "添加更多细节描述",
			AutoFixable: false,
		})
	}

	if score > 100 {
		score = 100
	}

	return score, issues
}

func auditSEOFields(keywords, intro, faq string) (int, []SEOIssue) {
	score := 100
	var issues []SEOIssue

	// SEO Keywords
	if keywords == "" {
		score -= 30
		issues = append(issues, SEOIssue{
			Type:        "error",
			Field:       "seo_keywords",
			Message:     "未设置 SEO 关键词",
			Suggestion:  "使用 AI 生成 SEO 关键词",
			AutoFixable: true,
		})
	} else {
		kwCount := len(splitKeywords(keywords))
		if kwCount < 5 {
			score -= 10
			issues = append(issues, SEOIssue{
				Type:        "info",
				Field:       "seo_keywords",
				Message:     fmt.Sprintf("SEO 关键词较少 (%d 个)，建议 8-12 个", kwCount),
				Suggestion:  "扩展关键词列表",
				AutoFixable: true,
			})
		}
	}

	// Intro
	if intro == "" {
		score -= 25
		issues = append(issues, SEOIssue{
			Type:        "error",
			Field:       "intro",
			Message:     "未设置介绍文案",
			Suggestion:  "使用 AI 生成介绍文案",
			AutoFixable: true,
		})
	} else if len(intro) < 50 {
		score -= 10
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "intro",
			Message:     "介绍文案过短",
			Suggestion:  "扩展介绍文案到 100-300 字符",
			AutoFixable: false,
		})
	}

	// FAQ
	if faq == "" {
		score -= 15
		issues = append(issues, SEOIssue{
			Type:        "info",
			Field:       "faq",
			Message:     "未设置 FAQ",
			Suggestion:  "使用 AI 生成 FAQ 以提升 GEO",
			AutoFixable: true,
		})
	}

	if score < 0 {
		score = 0
	}

	return score, issues
}

func auditSlug(slug, title string) (int, []SEOIssue) {
	score := 100
	var issues []SEOIssue

	if slug == "" {
		score -= 40
		issues = append(issues, SEOIssue{
			Type:        "error",
			Field:       "slug",
			Message:     "未设置 URL slug",
			Suggestion:  "生成 URL 友好的 slug",
			AutoFixable: true,
		})
		return score, issues
	}

	if len(slug) > 80 {
		score -= 15
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "slug",
			Message:     fmt.Sprintf("Slug 过长 (%d 字符)，建议 < 60", len(slug)),
			Suggestion:  "缩短 slug",
			AutoFixable: false,
		})
	}

	// 检查 slug 是否包含关键词
	slugLower := strings.ToLower(slug)
	titleWords := extractWords(title)
	keywordInSlug := false
	for _, word := range titleWords {
		if len(word) > 3 && strings.Contains(slugLower, strings.ToLower(word)) {
			keywordInSlug = true
			break
		}
	}
	if !keywordInSlug {
		score -= 15
		issues = append(issues, SEOIssue{
			Type:        "info",
			Field:       "slug",
			Message:     "Slug 未包含标题关键词",
			Suggestion:  "在 slug 中加入核心关键词",
			AutoFixable: false,
		})
	}

	return score, issues
}

func auditI18n(i18n, titleI18n, geoI18n string) (int, []SEOIssue) {
	score := 100
	var issues []SEOIssue

	// 检查是否有英文翻译
	hasEn := false
	if titleI18n != "" {
		var titleMap map[string]string
		if err := common.Unmarshal([]byte(titleI18n), &titleMap); err == nil {
			if _, ok := titleMap["en"]; ok && titleMap["en"] != "" {
				hasEn = true
			}
		}
	}

	if !hasEn {
		score -= 50
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "i18n",
			Message:     "缺少英文翻译（多语言 SEO 基础）",
			Suggestion:  "使用批量翻译功能生成英文版本",
			AutoFixable: true,
		})
	}

	// 检查是否完全翻译（12 种语言）
	if i18n != "" {
		var contentMap map[string]string
		if err := common.Unmarshal([]byte(i18n), &contentMap); err == nil {
			if len(contentMap) < 5 {
				score -= 20
				issues = append(issues, SEOIssue{
					Type:        "info",
					Field:       "i18n",
					Message:     fmt.Sprintf("仅翻译 %d 种语言，建议覆盖 12 种语言", len(contentMap)),
					Suggestion:  "使用自动翻译队列完成全部翻译",
					AutoFixable: true,
				})
			} else if len(contentMap) >= 10 {
				score += 10
			}
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, issues
}

func auditGeoBlocks(geoBlocks string) (int, []SEOIssue) {
	score := 100
	var issues []SEOIssue

	if geoBlocks == "" {
		score -= 50
		issues = append(issues, SEOIssue{
			Type:        "warning",
			Field:       "geo_blocks",
			Message:     "未设置 GEO 结构化内容",
			Suggestion:  "使用 AI 生成 GEO 结构化内容（适用场景/使用步骤/使用技巧）",
			AutoFixable: true,
		})
		return score, issues
	}

	// 检查内容是否丰富
	if len(geoBlocks) < 200 {
		score -= 20
		issues = append(issues, SEOIssue{
			Type:        "info",
			Field:       "geo_blocks",
			Message:     "GEO 结构化内容较短",
			Suggestion:  "扩展 GEO 内容以增加搜索引擎可见性",
			AutoFixable: false,
		})
	}

	return score, issues
}

// ==================== 辅助函数 ====================

func scoreStatus(score int) string {
	if score >= 80 {
		return "pass"
	}
	if score >= 60 {
		return "warning"
	}
	return "fail"
}

func calculateOverallScore(dimensions []DimensionScore) int {
	if len(dimensions) == 0 {
		return 0
	}

	totalWeight := 0
	weightedScore := 0

	for _, d := range dimensions {
		totalWeight += d.Weight
		weightedScore += d.Score * d.Weight
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedScore / totalWeight
}

func generateSuggestions(issues []SEOIssue) []string {
	var suggestions []string
	seen := make(map[string]bool)

	for _, issue := range issues {
		if issue.Type == "error" || issue.Type == "warning" {
			msg := issue.Suggestion
			if !seen[msg] {
				suggestions = append(suggestions, msg)
				seen[msg] = true
			}
		}
	}

	return suggestions
}

// splitKeywords 复用内链服务中的分割逻辑
func splitKeywords(s string) []string {
	var result []string
	for _, sep := range []string{",", "，", "|", " / ", " /"} {
		if strings.Contains(s, sep) {
			parts := strings.Split(s, sep)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					result = append(result, p)
				}
			}
			return result
		}
	}
	return strings.Fields(s)
}

// extractWords 复用内链服务中的提取逻辑
func extractWords(s string) []string {
	words := strings.Fields(s)
	var result []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		w = strings.Trim(w, ".,!?;:\"'()[]{}")
		if len(w) > 2 {
			result = append(result, strings.ToLower(w))
		}
	}
	return result
}

// SEOAuditCategory 兼容旧 Prompt SEO 审计的维度分类
type SEOAuditCategory struct {
	Score       int      `json:"score"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

// AuditPromptSEO 对 Prompt 进行 SEO 审计（供外部调用，如翻译完成后触发）
func AuditPromptSEO(prompt *model.Prompt) (*SEOAuditResult, error) {
	if prompt == nil {
		return nil, fmt.Errorf("prompt is nil")
	}
	return auditPrompt(prompt.Id)
}
