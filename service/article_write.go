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

const articleWriteAITimeout = 180 * time.Second

// ArticleWriteRequest AI 写文章的请求参数
type ArticleWriteRequest struct {
	Title        string `json:"title"`
	Prompt       string `json:"prompt"`
	ReferenceURL string `json:"reference_url"`
	Language     string `json:"language"`
}

// ArticleWriteResult AI 生成的文章内容（含 SEO + GEO）
type ArticleWriteResult struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	Summary       string `json:"summary"`
	Tags          string `json:"tags"`
	CoverImageUrl string `json:"cover_image_url"`
	Author        string `json:"author"`
	SeoTitle      string `json:"seo_title"`
	SeoDescription string `json:"seo_description"`
	SeoKeywords   string `json:"seo_keywords"`
	GeoKeywords   string `json:"geo_keywords"`
}

// GenerateArticle 调用 AI 根据用户输入生成 SEO+GEO 优化的完整文章
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
		"temperature": 0.6,
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
	return `You are an expert SEO content strategist and GEO (Generative Engine Optimization) specialist. Your task is to produce a complete, search-engine-optimized article.

Workflow you MUST follow:
1. Analyze the user's title/prompt/reference URL and extract the CORE topic and user intent.
2. Conduct keyword research: identify 1 primary keyword, 3-5 secondary keywords, and 5-8 long-tail keywords.
3. Build an article outline using H2/H3 headings that naturally incorporate the keywords.
4. Write the full article in Markdown (min 1000 words) with:
   - An engaging introduction that includes the primary keyword in the first 100 words
   - Well-structured H2/H3 sections using secondary keywords in headings
   - Keyword density: primary keyword 1-2%, secondary keywords 0.5-1%
   - At least one ordered or unordered list for featured snippets
   - A FAQ section at the end with 3-5 Q&A pairs (critical for GEO)
   - Bold key phrases and entities for AI engine comprehension
5. Generate all SEO and GEO metadata.

Return ONLY valid JSON. No markdown wrappers, no explanations.

JSON format:
{
  "title": "Compelling title (50-60 chars)",
  "content": "Full markdown article with H2/H3, lists, bold, code blocks if needed, and FAQ block at the end",
  "summary": "1-2 sentence summary (150-160 chars)",
  "tags": "5-10 keywords separated by commas",
  "cover_image_url": "Descriptive image search phrase or empty",
  "author": "Author name or Editorial Team",
  "seo_title": "SEO title 50-60 chars with primary keyword",
  "seo_description": "Meta description 150-160 chars with CTA",
  "seo_keywords": "8-12 SEO keywords separated by commas (include long-tail)",
  "geo_keywords": "5-8 GEO keywords for AI search engines (question-based, entity-focused)"
}`
}

func buildArticleWriteUserPromptTemplate() string {
	return `Please write an SEO+GEO optimized article with the following requirements:

Language: {{language}}
Title hint: {{title}}
Writing requirements / prompt: {{prompt}}
Reference article URL: {{reference_url}}

Follow the system workflow strictly:
1. Extract keywords first
2. Plan outline with keyword-rich headings
3. Write the full article (Markdown, min 1000 words)
4. Include a FAQ section at the end for GEO
5. Output all fields in the required JSON format.`
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
