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
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const articleWriteAITimeout = 120 * time.Second

// ArticleWriteRequest AI 写文章的请求参数
type ArticleWriteRequest struct {
	Title        string `json:"title"`
	Prompt       string `json:"prompt"`
	ReferenceURL string `json:"reference_url"`
	Language     string `json:"language"`
}

// ArticleWriteResult AI 生成的文章内容
type ArticleWriteResult struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	Summary       string `json:"summary"`
	Tags          string `json:"tags"`
	CoverImageUrl string `json:"cover_image_url"`
	Author        string `json:"author"`
}

// GenerateArticle 调用 AI 根据用户输入生成完整文章
func GenerateArticle(req *ArticleWriteRequest) (*ArticleWriteResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return nil, fmt.Errorf("ai writing not configured: please configure SEO AI settings first")
	}

	systemPrompt := buildArticleWriteSystemPrompt()
	userPromptTemplate := buildArticleWriteUserPromptTemplate()

	// 读取 article-write Skill 模板（如果存在）
	if skill, err := model.GetSkillBySkillId("article-write"); err == nil && skill.SystemPromptTemplate != "" {
		systemPrompt = skill.SystemPromptTemplate
		if skill.UserPromptTemplate != "" {
			userPromptTemplate = skill.UserPromptTemplate
		}
	}

	userContent := renderArticleWritePrompt(userPromptTemplate, req)

	reqBody := map[string]interface{}{
		"model": cfg.SeoAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		"temperature": 0.7,
		"max_tokens":  8000,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: articleWriteAITimeout}
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.SeoAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.SeoAIApiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai api returned status %d: %s", resp.StatusCode, string(body))
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

	var result ArticleWriteResult
	if err := common.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse article json failed: %w, content=%s", err, content)
	}

	return &result, nil
}

func buildArticleWriteSystemPrompt() string {
	return `You are an expert content writer and SEO specialist. Your task is to write a complete, high-quality article based on the user's requirements.

Requirements:
1. Write the article in the requested language
2. Use Markdown format for the content (headings, lists, bold, code blocks, tables, etc.)
3. The article should be comprehensive, well-structured, and engaging (at least 800 words if possible)
4. Include a compelling title, well-organized sections with H2/H3 headings, and practical insights
5. The summary should be 1-2 sentences capturing the essence of the article
6. Tags should be 5-10 relevant keywords separated by commas
7. Cover image URL: provide a descriptive image search phrase (e.g., "futuristic AI robot workspace") or leave empty
8. Author: provide a plausible author name or "Editorial Team"

Return ONLY valid JSON, no markdown, no explanation:
{"title":"...","content":"...","summary":"...","tags":"...","cover_image_url":"...","author":"..."}`
}

func buildArticleWriteUserPromptTemplate() string {
	return `Please write an article with the following requirements:

Language: {{language}}
Title hint: {{title}}
Writing requirements: {{prompt}}
Reference article URL: {{reference_url}}

Generate a complete article with title, markdown content, summary, tags, cover image URL (descriptive phrase or empty), and author name.`
}

func renderArticleWritePrompt(template string, req *ArticleWriteRequest) string {
	content := template
	lang := req.Language
	if lang == "" {
		lang = "zh"
	}
	content = strings.ReplaceAll(content, "{{language}}", lang)
	content = strings.ReplaceAll(content, "{{title}}", req.Title)
	content = strings.ReplaceAll(content, "{{prompt}}", req.Prompt)
	content = strings.ReplaceAll(content, "{{reference_url}}", req.ReferenceURL)
	return content
}
