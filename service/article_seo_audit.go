package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const articleSEOAuditTimeout = 90 * time.Second

// AuditArticleSEO 调用 AI 对文章的 SEO 内容进行审计
func AuditArticleSEO(article *model.Article) (*SEOAuditResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return nil, fmt.Errorf("seo ai not configured")
	}

	// 读取 article-seo-audit Skill 模板
	systemPrompt := buildDefaultArticleSEOAuditSystemPrompt()
	userPromptTemplate := buildDefaultArticleSEOAuditUserPrompt()

	if skill, err := model.GetSkillBySkillId("article-seo-audit"); err == nil && skill.SystemPromptTemplate != "" {
		systemPrompt = skill.SystemPromptTemplate
		if skill.UserPromptTemplate != "" {
			userPromptTemplate = skill.UserPromptTemplate
		}
	}

	userContent := renderArticleSEOAuditPrompt(userPromptTemplate, article)

	reqBody := map[string]interface{}{
		"model": cfg.SeoAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		"temperature": 0.3,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: articleSEOAuditTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.SeoAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.SeoAIApiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai api returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := common.DecodeJson(resp.Body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse ai response failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("ai response empty")
	}

	content := extractJSONFromMarkdown(apiResp.Choices[0].Message.Content)

	var result SEOAuditResult
	if err := common.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse audit json failed: %w, content=%s", err, content)
	}

	if result.OverallScore < 0 {
		result.OverallScore = 0
	}
	if result.OverallScore > 100 {
		result.OverallScore = 100
	}

	// 异步保存审计结果
	go saveArticleSEOAuditResult(article.Id, &result)

	return &result, nil
}

func saveArticleSEOAuditResult(articleId int, result *SEOAuditResult) {
	categoriesJSON, _ := common.Marshal(result.Categories)
	criticalJSON, _ := common.Marshal(result.CriticalIssues)
	quickWinsJSON, _ := common.Marshal(result.QuickWins)

	audit := &model.ArticleSEOAudit{
		ArticleId:      articleId,
		OverallScore:   result.OverallScore,
		Categories:     string(categoriesJSON),
		CriticalIssues: string(criticalJSON),
		QuickWins:      string(quickWinsJSON),
		CreatedAt:      time.Now().Unix(),
	}
	if err := model.CreateArticleSEOAudit(audit); err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("save article seo audit failed: article=%d err=%v", articleId, err))
	}
}

func buildDefaultArticleSEOAuditSystemPrompt() string {
	return `You are an expert SEO auditor specializing in blog/article SEO. Audit a single article page's SEO content across 5 dimensions.

Return ONLY valid JSON, no markdown, no explanation.`
}

func buildDefaultArticleSEOAuditUserPrompt() string {
	return `Audit the following SEO content for this article page:

## Article Information
Title: {{title}}
Summary: {{summary}}
Content Preview: {{content}}
Author: {{author}}
Tags: {{tags}}

## Current SEO Content
SEO Title: {{seo_title}}
SEO Description: {{seo_description}}
SEO Keywords: {{seo_keywords}}

## Audit Rules

### 1. Completeness (0-100)
- All 3 fields (seo_title, seo_description, seo_keywords) must be present and non-empty
- Each missing/empty field deducts 33 points

### 2. Keyword Quality (0-100)
- Relevance: keywords must match the article's topic and content
- Quantity: 5-12 keywords is optimal; too few (<3) or too many (>20) deducts points
- Specificity: avoid overly generic words like "article", "blog", "news"
- Long-tail keywords score higher

### 3. Title Quality (0-100)
- Length: 50-60 characters is optimal
- Keyword inclusion: title should naturally include at least 1 primary keyword
- Attractiveness: should encourage clicks (use numbers, questions, or strong words)
- Uniqueness: avoid generic titles

### 4. Description Quality (0-100)
- Length: 150-160 characters is optimal
- Keyword inclusion: description should include at least 2 keywords naturally
- Call-to-action: ideally includes a soft CTA or value proposition
- Summary quality: accurately reflects article content

### 5. Structured Data / Technical (0-100)
- Keywords are properly comma-separated
- No duplicate keywords
- Title and description are distinct (not identical)
- Keywords appear in both title and description

Return ONLY valid JSON in this exact format:
{"overall_score":0-100,"categories":{"completeness":{"score":0-100,"issues":["..."],"suggestions":["..."]},"keyword_quality":{"score":0-100,"issues":["..."],"suggestions":["..."]},"title_quality":{"score":0-100,"issues":["..."],"suggestions":["..."]},"description_quality":{"score":0-100,"issues":["..."],"suggestions":["..."]},"technical":{"score":0-100,"issues":["..."],"suggestions":["..."]}},"critical_issues":["..."],"quick_wins":["..."]}`
}

func renderArticleSEOAuditPrompt(template string, article *model.Article) string {
	content := article.Content
	if len(content) > 500 {
		content = content[:500] + "..."
	}

	summary := article.Summary
	if summary == "" {
		summary = "(empty)"
	}

	result := strings.ReplaceAll(template, "{{title}}", article.Title)
	result = strings.ReplaceAll(result, "{{summary}}", summary)
	result = strings.ReplaceAll(result, "{{content}}", content)
	result = strings.ReplaceAll(result, "{{author}}", article.Author)
	result = strings.ReplaceAll(result, "{{tags}}", article.Tags)
	result = strings.ReplaceAll(result, "{{seo_title}}", article.SeoTitle)
	result = strings.ReplaceAll(result, "{{seo_description}}", article.SeoDescription)
	result = strings.ReplaceAll(result, "{{seo_keywords}}", article.SeoKeywords)
	return result
}
