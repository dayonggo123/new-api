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

const articleSEOAITimeout = 60 * time.Second

// ArticleSEOArticleResult AI 生成的文章 SEO 内容
type ArticleSEOArticleResult struct {
	SeoTitle       string `json:"seo_title"`
	SeoDescription string `json:"seo_description"`
	SeoKeywords    string `json:"seo_keywords"`
}

// GenerateSEOForArticle 调用 AI 为文章生成 SEO 元数据
func GenerateSEOForArticle(article *model.Article) (*ArticleSEOArticleResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return nil, fmt.Errorf("seo ai not configured")
	}

	// 读取 article-seo Skill 模板（如果存在）
	systemPrompt := buildArticleSystemPrompt()
	userPromptTemplate := buildArticleUserPromptTemplate()
	if skill, err := model.GetSkillBySkillId("article-seo"); err == nil && skill.SystemPromptTemplate != "" {
		systemPrompt = skill.SystemPromptTemplate
		if skill.UserPromptTemplate != "" {
			userPromptTemplate = skill.UserPromptTemplate
		}
	}

	// 构建用户输入
	userContent := renderArticleSEOPrompt(userPromptTemplate, article)

	reqBody := map[string]interface{}{
		"model": cfg.SeoAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		"temperature": 0.7,
		"max_tokens":  4000,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: articleSEOAITimeout}
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

	content := apiResp.Choices[0].Message.Content
	content = extractJSONFromMarkdown(content)

	var result ArticleSEOArticleResult
	if err := common.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse seo json failed: %w, content=%s", err, content)
	}

	return &result, nil
}

// UpdateArticleSEO 更新文章的 SEO 字段
func UpdateArticleSEO(articleId int, result *ArticleSEOArticleResult) {
	updates := map[string]interface{}{
		"seo_title":       result.SeoTitle,
		"seo_description": result.SeoDescription,
		"seo_keywords":    result.SeoKeywords,
	}
	if err := model.DB.Model(&model.Article{}).Where("id = ?", articleId).Updates(updates).Error; err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("update article seo failed: id=%d err=%v", articleId, err))
	}
}

func buildArticleSystemPrompt() string {
	return `You are an expert in SEO and content marketing. 
Given an article's information, generate the following in the SAME LANGUAGE as the article:

1. seo_title: A compelling SEO title (50-60 characters) that includes the main keyword and attracts clicks
2. seo_description: A meta description (150-160 characters) that summarizes the article and encourages clicks
3. seo_keywords: 8-12 SEO keywords separated by commas (include long-tail keywords relevant to the article topic)

Return ONLY valid JSON, no markdown, no explanation:
{"seo_title":"...","seo_description":"...","seo_keywords":"kw1, kw2, ..."}`
}

func buildArticleUserPromptTemplate() string {
	return `Title: {{title}}
Summary: {{summary}}
Content Preview: {{content}}
Author: {{author}}
Tags: {{tags}}`
}

func renderArticleSEOPrompt(template string, article *model.Article) string {
	content := template
	content = strings.ReplaceAll(content, "{{title}}", article.Title)

	summary := article.Summary
	if len(summary) > 500 {
		summary = summary[:500] + "..."
	}
	content = strings.ReplaceAll(content, "{{summary}}", summary)

	preview := article.Content
	if len(preview) > 800 {
		preview = preview[:800] + "..."
	}
	content = strings.ReplaceAll(content, "{{content}}", preview)

	content = strings.ReplaceAll(content, "{{author}}", article.Author)
	content = strings.ReplaceAll(content, "{{tags}}", article.Tags)

	return content
}
