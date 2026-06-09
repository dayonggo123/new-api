package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	SeoTitle       string    `json:"seo_title"`
	SeoDescription string    `json:"seo_description"`
	SeoKeywords    string    `json:"seo_keywords"`
	Intro          string    `json:"intro"`
	Faq            []FaqItem `json:"faq"`
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
		"max_tokens":  6000,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	common.SysLog(fmt.Sprintf("[SEO] article=%d model=%s baseURL=%s", article.Id, cfg.SeoAIModel, cfg.SeoAIBaseURL))

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
		body, _ := io.ReadAll(resp.Body)
		common.SysLog(fmt.Sprintf("[SEO] article=%d API error: status=%d body=%s", article.Id, resp.StatusCode, string(body)))
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
	common.SysLog(fmt.Sprintf("[SEO] article=%d raw response: %s", article.Id, content))
	content = extractJSONFromMarkdown(content)

	var result ArticleSEOArticleResult
	if err := common.Unmarshal([]byte(content), &result); err != nil {
		common.SysLog(fmt.Sprintf("[SEO] article=%d json parse error: %v, content=%s", article.Id, err, content))
		return nil, fmt.Errorf("parse seo json failed: %w, content=%s", err, content)
	}

	common.SysLog(fmt.Sprintf("[SEO] article=%d parsed: title=%q intro=%q faq_len=%d", article.Id, result.SeoTitle, result.Intro, len(result.Faq)))

	// fallback: 如果 AI 没返回 intro，用 summary 或 content 前 300 字自动生成
	if result.Intro == "" {
		source := article.Summary
		if source == "" {
			source = article.Content
		}
		if len(source) > 300 {
			source = source[:300]
		}
		// 尝试在句号处截断，避免截断在句子中间
		if idx := strings.LastIndex(source, "。"); idx > 100 {
			source = source[:idx+3] // 包含句号
		}
		result.Intro = strings.TrimSpace(source)
		common.SysLog(fmt.Sprintf("[SEO] article=%d intro fallback applied: %q", article.Id, result.Intro))
	}

	return &result, nil
}

// UpdateArticleSEO 更新文章的 SEO 字段
func UpdateArticleSEO(articleId int, result *ArticleSEOArticleResult) {
	faqJSON, _ := common.Marshal(result.Faq)
	updates := map[string]interface{}{
		"seo_title":       result.SeoTitle,
		"seo_description": result.SeoDescription,
		"seo_keywords":    result.SeoKeywords,
		"intro":           result.Intro,
		"faq":             string(faqJSON),
	}
	common.SysLog(fmt.Sprintf("[SEO] article=%d updating db: intro=%q faq=%q", articleId, result.Intro, string(faqJSON)))
	if err := model.DB.Model(&model.Article{}).Where("id = ?", articleId).Updates(updates).Error; err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("update article seo failed: id=%d err=%v", articleId, err))
	} else {
		common.SysLog(fmt.Sprintf("[SEO] article=%d db updated successfully", articleId))
	}
}

func buildArticleSystemPrompt() string {
	return `You are an expert in SEO and content marketing. 
Given an article's information, generate the following in the SAME LANGUAGE as the article:

1. seo_title: A compelling SEO title (50-60 characters) that includes the main keyword and attracts clicks
2. seo_description: A meta description (150-160 characters) that summarizes the article and encourages clicks
3. seo_keywords: 8-12 SEO keywords separated by commas (include long-tail keywords relevant to the article topic)
4. intro: A rich, engaging introduction paragraph (200-400 characters) that hooks the reader and summarizes the article's value. This will be displayed as a highlighted card on the article page.
5. faq: An array of 3-5 frequently asked questions and answers related to the article topic. Each item should have "question" and "answer" fields. Answers should be concise (50-150 characters).

Return ONLY valid JSON, no markdown, no explanation:
{"seo_title":"...","seo_description":"...","seo_keywords":"kw1, kw2, ...","intro":"...","faq":[{"question":"...","answer":"..."},...]}`
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
