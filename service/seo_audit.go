package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const seoAuditTimeout = 90 * time.Second

// SEOAuditCategory 单个审计维度结果
type SEOAuditCategory struct {
	Score       int      `json:"score"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

// SEOAuditResult SEO 审计总结果
type SEOAuditResult struct {
	OverallScore   int                         `json:"overall_score"`
	Categories     map[string]SEOAuditCategory `json:"categories"`
	CriticalIssues []string                    `json:"critical_issues"`
	QuickWins      []string                    `json:"quick_wins"`
}

// AuditPromptSEO 调用 AI 对 Prompt 的 SEO 内容进行审计
func AuditPromptSEO(prompt *model.Prompt) (*SEOAuditResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return nil, fmt.Errorf("seo ai not configured")
	}

	// 读取 seo-audit Skill 模板
	systemPrompt := buildDefaultSEOAuditSystemPrompt()
	userPromptTemplate := buildDefaultSEOAuditUserPrompt()

	if skill, err := model.GetSkillBySkillId("seo-audit"); err == nil && skill.SystemPromptTemplate != "" {
		systemPrompt = skill.SystemPromptTemplate
		if skill.UserPromptTemplate != "" {
			userPromptTemplate = skill.UserPromptTemplate
		}
	}

	// 替换模板变量
	userContent := renderSEOAuditPrompt(userPromptTemplate, prompt)

	reqBody := map[string]interface{}{
		"model": cfg.SeoAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		"temperature": 0.3,
		"max_tokens":  4000,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: seoAuditTimeout}
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

	// 确保 overall_score 在 0-100 之间
	if result.OverallScore < 0 {
		result.OverallScore = 0
	}
	if result.OverallScore > 100 {
		result.OverallScore = 100
	}

	return &result, nil
}

func buildDefaultSEOAuditSystemPrompt() string {
	return `You are an expert SEO auditor specializing in AI prompt marketplace SEO. Audit a single prompt page's SEO content across 5 dimensions.

Return ONLY valid JSON, no markdown, no explanation.`
}

func buildDefaultSEOAuditUserPrompt() string {
	return `Audit the following SEO content for this AI prompt page:

## Prompt Information
Title: {{title}}
Content: {{content}}
Description: {{description}}
Model: {{model}}
Tags: {{tags}}

## Current SEO Content
SEO Keywords: {{seo_keywords}}
Intro: {{intro}}
FAQ: {{faq}}

## Audit Rules

### 1. Completeness (0-100)
- All 3 fields (keywords, intro, faq) must be present and non-empty
- Each missing/empty field deducts 33 points

### 2. Keyword Quality (0-100)
- Relevance: keywords must match the prompt's topic and content
- Quantity: 5-12 keywords is optimal; too few (<3) or too many (>20) deducts points
- Specificity: avoid overly generic words like "AI", "tool", "best"
- Long-tail keywords score higher

### 3. Intro Quality (0-100)
- Length: 80-300 characters is optimal
- Keyword inclusion: intro should naturally include at least 2 keywords
- Value proposition: should clearly explain what the prompt does and why users should try it
- Call-to-action: ideally includes a soft CTA

### 4. FAQ Quality (0-100)
- Must be valid JSON array with question/answer objects
- 3-5 Q&A pairs is optimal
- Questions must be relevant to the prompt topic
- Answers should be concise (50-200 chars) and helpful
- Include questions that AI search engines would ask

### 5. Structured Data (0-100)
- FAQ must be valid JSON
- Each item must have "question" and "answer" fields
- Compatible with Schema.org FAQPage format
- No nested objects or invalid types

Return ONLY valid JSON in this exact format:
{"overall_score":0-100,"categories":{"completeness":{"score":0-100,"issues":["..."],"suggestions":["..."]},"keyword_quality":{"score":0-100,"issues":["..."],"suggestions":["..."]},"intro_quality":{"score":0-100,"issues":["..."],"suggestions":["..."]},"faq_quality":{"score":0-100,"issues":["..."],"suggestions":["..."]},"structured_data":{"score":0-100,"issues":["..."],"suggestions":["..."]}},"critical_issues":["..."],"quick_wins":["..."]}`
}

func renderSEOAuditPrompt(template string, prompt *model.Prompt) string {
	content := prompt.Content
	if len(content) > 800 {
		content = content[:800] + "..."
	}

	faq := prompt.Faq
	if faq == "" {
		faq = "(empty)"
	}

	result := strings.ReplaceAll(template, "{{title}}", prompt.Title)
	result = strings.ReplaceAll(result, "{{content}}", content)
	result = strings.ReplaceAll(result, "{{description}}", prompt.Description)
	result = strings.ReplaceAll(result, "{{model}}", prompt.Model)
	result = strings.ReplaceAll(result, "{{tags}}", prompt.Tags)
	result = strings.ReplaceAll(result, "{{seo_keywords}}", prompt.SeoKeywords)
	result = strings.ReplaceAll(result, "{{intro}}", prompt.Intro)
	result = strings.ReplaceAll(result, "{{faq}}", faq)
	return result
}
