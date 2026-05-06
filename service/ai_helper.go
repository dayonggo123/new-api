package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const aiHelperTimeout = 60 * time.Second

// AISEOResult AI 生成的 SEO/GEO 内容
type AISEOResult struct {
	Keywords string    `json:"seo_keywords"`
	Intro    string    `json:"intro"`
	Faq      []FaqItem `json:"faq"`
}

type FaqItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// GenerateSEOForPrompt 调用 AI 为提示词生成 SEO 关键词、介绍文案和 FAQ
func GenerateSEOForPrompt(prompt *model.Prompt) (*AISEOResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return nil, fmt.Errorf("seo ai not configured")
	}

	// 构建用户输入
	userContent := buildAIInput(prompt)

	// 构建请求体
	reqBody := map[string]interface{}{
		"model": cfg.SeoAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": buildSystemPrompt()},
			{"role": "user", "content": userContent},
		},
		"temperature": 0.7,
		"max_tokens":  800,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: aiHelperTimeout}
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

	// 解析 JSON（AI 可能包裹在 markdown code block 中）
	content = extractJSONFromMarkdown(content)

	var result AISEOResult
	if err := common.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse seo json failed: %w, content=%s", err, content)
	}

	return &result, nil
}

// UpdatePromptSEO 更新提示词的 SEO 字段
func UpdatePromptSEO(promptId int, result *AISEOResult) {
	faqJSON, _ := common.Marshal(result.Faq)
	updates := map[string]interface{}{
		"seo_keywords": result.Keywords,
		"intro":        result.Intro,
		"faq":          string(faqJSON),
	}
	if err := model.DB.Model(&model.Prompt{}).Where("id = ?", promptId).Updates(updates).Error; err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("update prompt seo failed: id=%d err=%v", promptId, err))
	}
}

func buildSystemPrompt() string {
	return `You are an expert in SEO and GEO (Generative Engine Optimization). 
Given a prompt's information, generate the following in the SAME LANGUAGE as the prompt:

1. seo_keywords: 8-12 SEO keywords separated by commas (include long-tail keywords)
2. intro: A compelling 1-paragraph introduction (max 300 characters) summarizing what this prompt does and its value
3. faq: An array of 3-4 Q&A pairs that AI search engines (like ChatGPT, Perplexity) would find useful

Return ONLY valid JSON, no markdown, no explanation:
{"seo_keywords":"kw1, kw2, ...","intro":"...","faq":[{"question":"...","answer":"..."},...]}`
}

func buildAIInput(prompt *model.Prompt) string {
	input := fmt.Sprintf("Title: %s\n", prompt.Title)
	if prompt.Content != "" {
		content := prompt.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		input += fmt.Sprintf("Content: %s\n", content)
	}
	if prompt.Model != "" {
		input += fmt.Sprintf("Target Model: %s\n", prompt.Model)
	}
	if prompt.Tags != "" {
		input += fmt.Sprintf("Tags: %s\n", prompt.Tags)
	}
	if prompt.Description != "" {
		input += fmt.Sprintf("Description: %s\n", prompt.Description)
	}
	return input
}

func extractJSONFromMarkdown(content string) string {
	// 如果 AI 返回了 ```json ... ``` 包裹的 JSON，提取内部内容
	if len(content) > 7 && content[:7] == "```json" {
		content = content[7:]
		if idx := bytes.Index([]byte(content), []byte("```")); idx != -1 {
			content = content[:idx]
		}
		return trimWhitespace(content)
	}
	if len(content) > 3 && content[:3] == "```" {
		content = content[3:]
		if idx := bytes.Index([]byte(content), []byte("```")); idx != -1 {
			content = content[:idx]
		}
		return trimWhitespace(content)
	}
	return trimWhitespace(content)
}

func trimWhitespace(s string) string {
	return string(bytes.TrimSpace([]byte(s)))
}
