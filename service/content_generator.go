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

const contentGenTimeout = 180 * time.Second

// ContentGenerationType 内容生成类型
type ContentGenerationType string

const (
	ContentTypeArticle ContentGenerationType = "article"
	ContentTypePrompt  ContentGenerationType = "prompt"
)

// ContentGenerationRequest 内容生成请求
type ContentGenerationRequest struct {
	Type           ContentGenerationType `json:"type" binding:"required"` // article 或 prompt
	Keywords       []string              `json:"keywords" binding:"required"`
	Language       string                `json:"language"`                // 默认 "en"
	AutoSEO        bool                  `json:"auto_seo"`                // 自动生成 SEO 字段
	AutoGEO        bool                  `json:"auto_geo"`                // 自动生成 GEO 结构化内容
	AutoTranslate  bool                  `json:"auto_translate"`          // 自动翻译 12 种语言
	AutoPublish    bool                  `json:"auto_publish"`            // 自动发布（status = 1）
}

// ContentGenerationResult 内容生成结果
type ContentGenerationResult struct {
	Type         string `json:"type"`
	RecordID     int    `json:"record_id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Slug         string `json:"slug"`
	Status       string `json:"status"`       // completed / failed
	ErrorMessage string `json:"error_message,omitempty"`
}

// GenerateContent 根据关键词生成内容并自动保存
func GenerateContent(req *ContentGenerationRequest) (*ContentGenerationResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return nil, fmt.Errorf("ai content generation not configured: please configure SEO AI settings first")
	}

	lang := req.Language
	if lang == "" {
		lang = "en"
	}

	switch req.Type {
	case ContentTypeArticle:
		return generateArticleContent(req, lang)
	case ContentTypePrompt:
		return generatePromptContent(req, lang)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", req.Type)
	}
}

// generateArticleContent 生成文章并保存
func generateArticleContent(req *ContentGenerationRequest, lang string) (*ContentGenerationResult, error) {
	keywordsStr := strings.Join(req.Keywords, ", ")
	mainKeyword := req.Keywords[0]

	// Step 1: AI 生成文章
	articleResult, err := callAIForArticle(mainKeyword, keywordsStr, lang)
	if err != nil {
		return &ContentGenerationResult{
			Type:         string(ContentTypeArticle),
			Status:       "failed",
			ErrorMessage: err.Error(),
		}, err
	}

	// Step 2: 生成 slug
	slug := model.GenerateSlug(articleResult.Title)

	// Step 3: 创建文章记录
	article := &model.Article{
		Title:         articleResult.Title,
		Slug:          slug,
		Content:       articleResult.Content,
		Summary:       articleResult.Summary,
		Author:        "AI Generator",
		Status:        0, // 草稿
		CreatedTime:   common.GetTimestamp(),
		UpdatedTime:   common.GetTimestamp(),
		SeoKeywords:   articleResult.SeoKeywords,
		Intro:         articleResult.Intro,
	}

	if err := model.DB.Create(article).Error; err != nil {
		return &ContentGenerationResult{
			Type:         string(ContentTypeArticle),
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("save article failed: %v", err),
		}, err
	}

	result := &ContentGenerationResult{
		Type:     string(ContentTypeArticle),
		RecordID: article.Id,
		Title:    article.Title,
		Content:  article.Content,
		Slug:     article.Slug,
		Status:   "completed",
	}

	// Step 4: 自动 SEO（如果启用）
	if req.AutoSEO {
		go func() {
			if seoResult, err := GenerateSEOForArticle(article); err == nil {
				updates := map[string]interface{}{
					"seo_keywords": seoResult.SeoKeywords,
					"intro":        seoResult.Intro,
					"faq":          seoResult.Faq,
				}
				model.DB.Model(&model.Article{}).Where("id = ?", article.Id).Updates(updates)
			}
		}()
	}

	// Step 5: 自动 GEO（如果启用）
	if req.AutoGEO {
		go func() {
			_ = GenerateArticleGeoBlocks(article.Id)
		}()
	}

	// Step 6: 自动翻译（如果启用）
	if req.AutoTranslate {
		go StartAutoTranslate("article", article.Id)
	}

	// Step 7: 自动发布（如果启用）
	if req.AutoPublish {
		model.DB.Model(&model.Article{}).Where("id = ?", article.Id).Update("status", 1)
	}

	return result, nil
}

// generatePromptContent 生成 Prompt 并保存
func generatePromptContent(req *ContentGenerationRequest, lang string) (*ContentGenerationResult, error) {
	keywordsStr := strings.Join(req.Keywords, ", ")
	mainKeyword := req.Keywords[0]

	// Step 1: AI 生成 Prompt
	promptResult, err := callAIForPrompt(mainKeyword, keywordsStr, lang)
	if err != nil {
		return &ContentGenerationResult{
			Type:         string(ContentTypePrompt),
			Status:       "failed",
			ErrorMessage: err.Error(),
		}, err
	}

	// Step 2: 生成 slug
	slug := model.GenerateSlug(promptResult.Title)

	// Step 3: 创建 Prompt 记录
	prompt := &model.Prompt{
		Title:         promptResult.Title,
		Slug:          slug,
		Content:       promptResult.Content,
		Description:   promptResult.Description,
		Author:        "AI Generator",
		Status:        1, // 直接启用
		CreatedTime:   common.GetTimestamp(),
		UpdatedTime:   common.GetTimestamp(),
	}

	if err := model.DB.Create(prompt).Error; err != nil {
		return &ContentGenerationResult{
			Type:         string(ContentTypePrompt),
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("save prompt failed: %v", err),
		}, err
	}

	result := &ContentGenerationResult{
		Type:     string(ContentTypePrompt),
		RecordID: prompt.Id,
		Title:    prompt.Title,
		Content:  prompt.Content,
		Slug:     prompt.Slug,
		Status:   "completed",
	}

	// Step 4: 自动 SEO（如果启用）
	if req.AutoSEO {
		go func() {
			if seoResult, err := GenerateSEOForPrompt(prompt); err == nil {
				UpdatePromptSEO(prompt.Id, seoResult)
			}
		}()
	}

	// Step 5: 自动 GEO（如果启用）
	if req.AutoGEO {
		go func() {
			_ = GeneratePromptGeoBlocks(prompt.Id)
		}()
	}

	// Step 6: 自动翻译（如果启用）
	if req.AutoTranslate {
		go StartAutoTranslate("prompt", prompt.Id)
	}

	return result, nil
}

// ==================== AI 调用 ====================

// articleAIResult AI 生成的文章结果
type articleAIResult struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Summary     string `json:"summary"`
	SeoKeywords string `json:"seo_keywords"`
	Intro       string `json:"intro"`
}

// promptAIResult AI 生成的 Prompt 结果
type promptAIResult struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// callAIForArticle 调用 AI 生成文章
func callAIForArticle(mainKeyword, allKeywords, lang string) (*articleAIResult, error) {
	cfg := operation_setting.GetSEOSetting()

	systemPrompt := fmt.Sprintf(`You are an expert SEO content writer specializing in AI video creation and prompt engineering. Write a comprehensive, SEO-optimized article in %s.

Requirements:
1. Write 2000-3000 words of high-quality content
2. Use the main keyword naturally throughout (1-2%% density)
3. Include H2 and H3 headings for structure
4. Write in a helpful, authoritative tone
5. Include practical examples and actionable advice
6. End with a strong call-to-action
7. Target audience: video creators, marketers, AI enthusiasts

Return ONLY valid JSON with this exact structure:
{"title":"...","content":"...","summary":"...","seo_keywords":"kw1, kw2, kw3, kw4, kw5","intro":"..."}

- title: compelling, click-worthy title under 60 chars, include main keyword
- content: full article HTML/markdown, with ## headings
- summary: 1-paragraph summary (max 200 chars)
- seo_keywords: 8-12 keywords separated by commas
- intro: compelling intro paragraph (max 300 chars)`, lang)

	userPrompt := fmt.Sprintf(`Write an SEO-optimized article targeting the keyword: "%s"

Related keywords to include naturally: %s

The article should be for harse.tv — an AI creative workspace offering node-based canvas video creation and AI video prompt library (supports Sora, Kling, Veo, Runway, and more).

Focus on providing genuine value to readers. Don't just promote harse.tv — educate and inform.`, mainKeyword, allKeywords)

	return callAIForContent(systemPrompt, userPrompt, cfg)
}

// callAIForPrompt 调用 AI 生成 Prompt
func callAIForPrompt(mainKeyword, allKeywords, lang string) (*promptAIResult, error) {
	cfg := operation_setting.GetSEOSetting()

	systemPrompt := fmt.Sprintf(`You are an expert prompt engineer. Create a high-quality AI video generation prompt in %s.

Requirements:
1. The prompt should be detailed, specific, and produce excellent results
2. Include clear instructions for AI video models (Sora, Kling, Veo, etc.)
3. The description should explain what the prompt does and how to use it
4. Include an example output description
5. Make it practical and ready-to-use

Return ONLY valid JSON with this exact structure:
{"title":"...","content":"...","description":"...","example":"..."}

- title: descriptive title (max 100 chars)
- content: the full prompt text (detailed, 200-500 words)
- description: what this prompt does and best use cases (max 300 chars)
- example: description of what the AI would generate with this prompt (max 200 chars)`, lang)

	userPrompt := fmt.Sprintf(`Create an AI video generation prompt for the topic: "%s"

Related concepts: %s

This prompt will be published on harse.tv's prompt library — a collection of curated AI video prompts for creators.

Make the prompt detailed enough to produce high-quality, consistent results across different AI video models.`, mainKeyword, allKeywords)

	// 复用文章生成的 AI 调用逻辑，然后映射到 prompt 结构
	rawResult, err := callAIForContent(systemPrompt, userPrompt, cfg)
	if err != nil {
		return nil, err
	}

	return &promptAIResult{
		Title:       rawResult.Title,
		Content:     rawResult.Content,
		Description: rawResult.Summary,
		Example:     rawResult.Intro,
	}, nil
}

// callAIForContent 通用 AI 调用
func callAIForContent(systemPrompt, userPrompt string, cfg *operation_setting.SEOSetting) (*articleAIResult, error) {
	reqBody := map[string]interface{}{
		"model": cfg.SeoAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.6,
		"max_tokens":  4000,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: contentGenTimeout}
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

	var result articleAIResult
	if err := common.Unmarshal([]byte(content), &result); err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("parse content gen json failed: %v, content=%s", err, content))
		// 降级处理：尝试提取文本
		return fallbackExtractArticle(content)
	}

	return &result, nil
}

// fallbackExtractArticle 降级提取文章内容
func fallbackExtractArticle(content string) (*articleAIResult, error) {
	// 简单提取标题（第一行）
	lines := strings.Split(content, "\n")
	var title string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			title = line
			break
		}
	}
	if title == "" {
		title = "AI Generated Content"
	}

	return &articleAIResult{
		Title:       title,
		Content:     content,
		Summary:     "AI generated content",
		SeoKeywords: "AI video, prompt, generation",
		Intro:       "",
	}, nil
}
