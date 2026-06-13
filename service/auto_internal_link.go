package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// InternalLinkSuggestion 内链建议
type InternalLinkSuggestion struct {
	TargetID    int    `json:"target_id"`
	TargetType  string `json:"target_type"` // article / prompt
	TargetTitle string `json:"target_title"`
	TargetSlug  string `json:"target_slug"`
	AnchorText  string `json:"anchor_text"`
	Relevance   int    `json:"relevance"` // 0-100 相关度
	Reason      string `json:"reason"`
}

// SuggestInternalLinks 为指定内容推荐内链
func SuggestInternalLinks(recordType string, recordID int, limit int) ([]InternalLinkSuggestion, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	// 获取源内容
	var sourceKeywords []string
	var sourceTitle string

	switch recordType {
	case "article":
		article, err := model.GetArticleById(recordID)
		if err != nil {
			return nil, err
		}
		sourceTitle = article.Title
		sourceKeywords = extractKeywords(article)
	case "prompt":
		prompt, err := model.GetPromptById(recordID)
		if err != nil {
			return nil, err
		}
		sourceTitle = prompt.Title
		sourceKeywords = extractKeywordsFromPrompt(prompt.Prompt)
	default:
		return nil, nil
	}

	if len(sourceKeywords) == 0 {
		return nil, nil
	}

	// 搜索相关文章
	var suggestions []InternalLinkSuggestion

	// 搜索相关文章（排除自己）
	articles, _, err := model.GetPublicArticles(0, "", 0, 50)
	if err == nil {
		for _, article := range articles {
			if recordType == "article" && article.Id == recordID {
				continue
			}
			score := calculateRelevance(sourceKeywords, sourceTitle, article.Title, article.SeoKeywords, article.Tags)
			if score > 30 {
				suggestions = append(suggestions, InternalLinkSuggestion{
					TargetID:    article.Id,
					TargetType:  "article",
					TargetTitle: article.Title,
					TargetSlug:  article.Slug,
					AnchorText:  generateAnchorText(sourceTitle, article.Title),
					Relevance:   score,
					Reason:      generateReason("article", article.Title, score),
				})
			}
		}
	}

	// 搜索相关 Prompt（排除自己）
	prompts, _, err := model.GetPublicPrompts(0, "", 0, 50, "id", "asc")
	if err == nil {
		for _, prompt := range prompts {
			if recordType == "prompt" && prompt.Id == recordID {
				continue
			}
			score := calculateRelevance(sourceKeywords, sourceTitle, prompt.Title, prompt.SeoKeywords, prompt.Tags)
			if score > 30 {
				suggestions = append(suggestions, InternalLinkSuggestion{
					TargetID:    prompt.Id,
					TargetType:  "prompt",
					TargetTitle: prompt.Title,
					TargetSlug:  prompt.Slug,
					AnchorText:  generateAnchorText(sourceTitle, prompt.Title),
					Relevance:   score,
					Reason:      generateReason("prompt", prompt.Title, score),
				})
			}
		}
	}

	// 按相关度排序
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].Relevance > suggestions[i].Relevance {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// 截取前 limit 个
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

// extractKeywords 从文章中提取关键词
func extractKeywords(article *model.Article) []string {
	var keywords []string
	if article.SeoKeywords != "" {
		keywords = append(keywords, splitKeywords(article.SeoKeywords)...)
	}
	if article.Title != "" {
		keywords = append(keywords, extractWords(article.Title)...)
	}
	if article.Tags != "" {
		keywords = append(keywords, splitKeywords(article.Tags)...)
	}
	return keywords
}

// extractKeywordsFromPrompt 从 Prompt 中提取关键词
func extractKeywordsFromPrompt(prompt *model.Prompt) []string {
	var keywords []string
	if prompt.SeoKeywords != "" {
		keywords = append(keywords, splitKeywords(prompt.SeoKeywords)...)
	}
	if prompt.Title != "" {
		keywords = append(keywords, extractWords(prompt.Title)...)
	}
	if prompt.Tags != "" {
		keywords = append(keywords, splitKeywords(prompt.Tags)...)
	}
	if prompt.Model != "" {
		keywords = append(keywords, extractWords(prompt.Model)...)
	}
	return keywords
}


// calculateRelevance 计算两个内容的相关度
func calculateRelevance(sourceKeywords []string, sourceTitle, targetTitle, targetSEO, targetTags string) int {
	score := 0
	maxScore := 100

	// 标题匹配（最高权重）
	sourceTitleLower := strings.ToLower(sourceTitle)
	targetTitleLower := strings.ToLower(targetTitle)

	if sourceTitleLower == targetTitleLower {
		return 0 // 相同标题不算相关
	}

	// 计算标题词重叠
	sourceTitleWords := extractWords(sourceTitle)
	targetTitleWords := extractWords(targetTitle)
	titleOverlap := countOverlap(sourceTitleWords, targetTitleWords)
	if len(sourceTitleWords) > 0 {
		score += (titleOverlap * 100 / len(sourceTitleWords)) * 40 / 100 // 标题权重 40%
	}

	// SEO 关键词匹配
	if targetSEO != "" {
		targetSEOWords := splitKeywords(targetSEO)
		seoOverlap := countOverlap(sourceKeywords, targetSEOWords)
		if len(sourceKeywords) > 0 {
			score += (seoOverlap * 100 / len(sourceKeywords)) * 30 / 100 // SEO 权重 30%
		}
	}

	// 标签匹配
	if targetTags != "" {
		targetTagWords := splitKeywords(targetTags)
		tagOverlap := countOverlap(sourceKeywords, targetTagWords)
		if len(sourceKeywords) > 0 {
			score += (tagOverlap * 100 / len(sourceKeywords)) * 20 / 100 // 标签权重 20%
		}
	}

	// 语义相关：检查关键词是否在目标标题中出现
	for _, kw := range sourceKeywords {
		if len(kw) > 3 && strings.Contains(targetTitleLower, strings.ToLower(kw)) {
			score += 10
			break
		}
	}

	if score > maxScore {
		score = maxScore
	}

	return score
}

// countOverlap 计算两个词列表的重叠数
func countOverlap(a, b []string) int {
	count := 0
	seen := make(map[string]bool)
	for _, w := range b {
		seen[strings.ToLower(w)] = true
	}
	for _, w := range a {
		if seen[strings.ToLower(w)] {
			count++
		}
	}
	return count
}

// generateAnchorText 生成锚文本
func generateAnchorText(sourceTitle, targetTitle string) string {
	// 简单策略：使用目标标题的核心词
	words := extractWords(targetTitle)
	if len(words) >= 3 {
		return strings.Join(words[:3], " ")
	}
	return targetTitle
}

// generateReason 生成推荐理由
func generateReason(targetType, targetTitle string, score int) string {
	typeStr := "文章"
	if targetType == "prompt" {
		typeStr = "提示词"
	}

	if score >= 80 {
		return fmt.Sprintf("高度相关 %s《%s》，主题完全匹配", typeStr, targetTitle)
	}
	if score >= 60 {
		return fmt.Sprintf("相关 %s《%s》，关键词覆盖度高", typeStr, targetTitle)
	}
	if score >= 40 {
		return fmt.Sprintf("中度相关 %s《%s》，可作为补充阅读", typeStr, targetTitle)
	}
	return fmt.Sprintf("相关 %s《%s》", typeStr, targetTitle)
}
