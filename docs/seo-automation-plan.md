# harse.tv SEO 自动化执行方案

## v1.0 | 基于 SOP v2.0 海外市场版 | 最大化自动化

> **目标：** 将 SOP v2.0 的 8 步关键词研究流程，转化为可自动运行的内容生产流水线  
> **预期效果：** 人工仅需「输入主题方向」，系统自动完成关键词研究 → 内容生成 → 多语言翻译 → Schema 注入 → 发布索引  
> **当前技术基础：** New-API 已具备自动翻译队列、GEO 结构化生成、批量 SEO 翻译、Sitemap、Schema、i18n 等核心能力

---

## 一、整体架构：5 阶段自动化流水线

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    harse.tv SEO 自动化内容生产流水线                           │
└─────────────────────────────────────────────────────────────────────────────┘

  Phase 1          Phase 2           Phase 3          Phase 4         Phase 5
  关键词研究        内容生成           SEO 优化          多语言发布        索引监控
  ┌─────┐         ┌─────┐          ┌─────┐          ┌─────┐        ┌─────┐
  │     │         │     │          │     │          │     │        │     │
  │ AI  │───────→│ AI  │─────────→│Auto │─────────→│Auto │───────→│Auto │
  │Research│      │Gen  │          │SEO  │          │i18n │        │Index│
  │     │         │     │          │     │          │     │        │     │
  └─────┘         └─────┘          └─────┘          └─────┘        └─────┘
     │               │                │               │              │
     ▼               ▼                ▼               ▼              ▼
  种子词清单      文章/Prompt      SEO字段填充      12语种翻译      Google
  扩展词矩阵      草稿生成         GEO块生成        hreflang更新    Indexing API
  ROI评分表       结构化内容       FAQ自动生成      Sitemap更新     排名监控
  主题簇地图      多媒体资产       内链建议         自动发布        报告生成

  ████ = 已有能力（复用）
  ░░░░ = 需新增能力（开发）
```

### 自动化程度分级

| 阶段 | 自动化率 | 人工介入点 | 已有基础 | 需新增 |
|------|----------|-----------|----------|--------|
| **P1 关键词研究** | 80% | 审核/调整 AI 输出 | ❌ 无 | **新建** AI Research Service |
| **P2 内容生成** | 70% | 审核/微调内容质量 | ⚠️ 半成（GEO 生成） | **增强** Prompt 生成 + 文章生成 |
| **P3 SEO 优化** | 90% | 最终审核 | ✅ 强（批量翻译+GEO） | **微调** 自动内链 |
| **P4 多语言发布** | 95% | 监控异常 | ✅ 强（自动翻译队列） | **增强** 自动发布触发 |
| **P5 索引监控** | 85% | 策略调整 | ⚠️ Sitemap 已有 | **新建** Indexing API + 监控 |

---

## 二、Phase 1：关键词研究自动化（80% 自动化）

### 2.1 目标
将 SOP v2.0 的 Step 1-2（种子词收集 + 关键词扩展）自动化，人工仅需输入「主题方向」，AI 输出完整的关键词研究报告。

### 2.2 技术实现

#### 新增服务：`service/seo_research.go`

```go
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// ========== 数据结构 ==========

type KeywordResearchResult struct {
	Topic          string                 `json:"topic"`           // 研究主题
	SeedKeywords   []SeedKeyword          `json:"seed_keywords"`   // 种子词
	ExpandedKeywords []ExpandedKeyword    `json:"expanded_keywords"` // 扩展词
	TopicClusters  []TopicCluster         `json:"topic_clusters"`  // 主题簇
	Competitors    []CompetitorInsight    `json:"competitors"`     // 竞品洞察
	ContentGaps    []ContentGap           `json:"content_gaps"`    // 内容缺口
	GeneratedAt    time.Time              `json:"generated_at"`
}

type SeedKeyword struct {
	Keyword   string `json:"keyword"`
	Dimension string `json:"dimension"` // A/B/C/D: 品类/技术/场景/模型
	Language  string `json:"language"`  // en/ja/ko/...
	Priority  string `json:"priority"`  // P0/P1/P2
}

type ExpandedKeyword struct {
	Keyword      string  `json:"keyword"`
	SearchVolume string  `json:"search_volume"` // 预估范围，如 "5K-10K"
	Difficulty   int     `json:"difficulty"`    // 0-100
	Intent       string  `json:"intent"`        // informational/commercial/transactional/navigational
	ModelTag     string  `json:"model_tag"`     // sora/kling/veo/runway/etc
	Category     string  `json:"category"`      // library/tutorial/comparison/etc
	ROIScore     int     `json:"roi_score"`     // 0-100
}

type TopicCluster struct {
	PillarKeyword    string   `json:"pillar_keyword"`
	PillarSearchVol  string   `json:"pillar_search_vol"`
	ClusterKeywords  []string `json:"cluster_keywords"`
	SuggestedURL     string   `json:"suggested_url"`
	ContentType      string   `json:"content_type"` // prompt_detail/article/tool_page
}

type CompetitorInsight struct {
	Domain       string   `json:"domain"`
	Strengths    []string `json:"strengths"`
	Weaknesses   []string `json:"weaknesses"`
	RankedKeywords []string `json:"ranked_keywords"`
}

type ContentGap struct {
	Keyword      string `json:"keyword"`
	Priority     string `json:"priority"`     // P0/P1/P2
	Competitor   string `json:"competitor"`   // 谁在做
	OurStatus    string `json:"our_status"`   // not_started/in_progress/done
	Action       string `json:"action"`       // 建议行动
}

// ========== 核心接口 ==========

// ResearchKeywords 执行自动化关键词研究
// 输入：主题方向（如 "AI video prompt library"）
// 输出：完整的研究报告（JSON）
func ResearchKeywords(topic string, targetLangs []string) (*KeywordResearchResult, error) {
	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" {
		return nil, fmt.Errorf("AI translation/research not configured")
	}

	// Step 1: 生成种子词（基于模型词矩阵 + 产品能力）
	seeds := generateSeedKeywords(topic, targetLangs)

	// Step 2: AI 扩展关键词（调用 LLM 生成扩展词和 ROI 评分）
	expanded, err := aiExpandKeywords(cfg, topic, seeds)
	if err != nil {
		return nil, fmt.Errorf("keyword expansion failed: %w", err)
	}

	// Step 3: 主题聚类（基于语义相似度自动聚类）
	clusters := clusterKeywords(expanded)

	// Step 4: 竞品洞察（基于已知竞品库）
	competitors := analyzeCompetitors(topic)

	// Step 5: 内容缺口（对比已有内容 vs 研究输出）
	gaps := findContentGaps(expanded, clusters)

	return &KeywordResearchResult{
		Topic:            topic,
		SeedKeywords:     seeds,
		ExpandedKeywords: expanded,
		TopicClusters:    clusters,
		Competitors:      competitors,
		ContentGaps:      gaps,
		GeneratedAt:      time.Now(),
	}, nil
}

// ========== 内部实现 ==========

// generateSeedKeywords 基于「模型词矩阵」生成种子词
// 规则：Topic × Model × Action × Format = 种子词组合
func generateSeedKeywords(topic string, targetLangs []string) []SeedKeyword {
	models := []string{"Sora", "Sora 2", "Kling", "Kling 3.0", "Veo", "Veo 3", "Runway", "Runway Gen-4", "Pika", "Pika 2.0", "Seedance", "Luma"}
	actions := []string{"prompt", "template", "guide", "tutorial", "examples", "generator", "best practices", "tips"}
	formats := []string{"for beginners", "for marketing", "cinematic", "short-form", "product video", "anime style", "for YouTube", "free"}
	scenarios := []string{"AI video creation", "node canvas", "AI storyboard", "video workflow", "prompt engineering", "AI video tool comparison"}

	var seeds []SeedKeyword
	lang := "en" // 默认英语种子词
	if len(targetLangs) > 0 {
		lang = targetLangs[0]
	}

	// A. 品类词（直接来自 topic）
	seeds = append(seeds, SeedKeyword{Keyword: topic, Dimension: "A", Language: lang, Priority: "P0"})
	seeds = append(seeds, SeedKeyword{Keyword: "AI video creation tool", Dimension: "A", Language: lang, Priority: "P0"})
	seeds = append(seeds, SeedKeyword{Keyword: "best AI video generator 2026", Dimension: "A", Language: lang, Priority: "P0"})

	// B. 技术/差异化词
	seeds = append(seeds, SeedKeyword{Keyword: "node canvas video editor", Dimension: "B", Language: lang, Priority: "P0"})
	seeds = append(seeds, SeedKeyword{Keyword: "ComfyUI video workflow", Dimension: "B", Language: lang, Priority: "P1"})
	seeds = append(seeds, SeedKeyword{Keyword: "AI storyboard creator", Dimension: "B", Language: lang, Priority: "P1"})

	// C. 场景词
	seeds = append(seeds, SeedKeyword{Keyword: "AI video prompt for YouTube shorts", Dimension: "C", Language: lang, Priority: "P0"})
	seeds = append(seeds, SeedKeyword{Keyword: "product demo AI video prompt", Dimension: "C", Language: lang, Priority: "P1"})

	// D. 模型专属词（高转化率）
	for _, m := range models[:5] { // 前 5 个最重要模型
		seeds = append(seeds, SeedKeyword{
			Keyword:   fmt.Sprintf("%s prompt examples", m),
			Dimension: "D",
			Language:  lang,
			Priority:  "P0",
		})
		seeds = append(seeds, SeedKeyword{
			Keyword:   fmt.Sprintf("%s prompt template", m),
			Dimension: "D",
			Language:  lang,
			Priority:  "P0",
		})
	}

	return seeds
}

// aiExpandKeywords 调用 AI 进行关键词扩展和 ROI 评分
func aiExpandKeywords(cfg operation_setting.TranslateSetting, topic string, seeds []SeedKeyword) ([]ExpandedKeyword, error) {
	// 构建种子词文本
	var seedTexts []string
	for _, s := range seeds {
		seedTexts = append(seedTexts, fmt.Sprintf("- %s (维度:%s 优先级:%s)", s.Keyword, s.Dimension, s.Priority))
	}

	systemPrompt := `You are an expert SEO keyword researcher specializing in the AI video generation niche.
Your task is to expand seed keywords into a comprehensive keyword list with ROI scoring.

For each seed keyword, generate 5-10 related keywords including:
- Long-tail variations (3-5 words)
- Question-based keywords (how to, what is, best way to)
- Comparison keywords (vs, alternative, better than)
- Model-specific variations (Sora, Kling, Veo, Runway, Pika)
- Use-case specific variations (marketing, YouTube, cinematic, product demo)

Output format: JSON array of objects with these exact fields:
- "keyword": string
- "search_volume": string (estimate like "1K-3K" or "500-1K")
- "difficulty": number (0-100, based on competitor analysis)
- "intent": string (informational/commercial/transactional/navigational)
- "model_tag": string (which AI model, or "multi" or "none")
- "category": string (library/tutorial/comparison/tool/generator)
- "roi_score": number (0-100, calculated from search volume × commercial intent ÷ difficulty)

Rules:
1. Return ONLY valid JSON array. No markdown, no explanations.
2. Focus on English keywords (Google Global market).
3. Difficulty should be realistic: high DR competitors = higher difficulty.
4. ROI score formula: (estimated_volume / 100) * intent_weight(Info=1, Comm=3, Trans=4) * 10 / (difficulty + 1)
5. Include at least 50 keywords total.`

	userPrompt := fmt.Sprintf("Topic: %s\n\nSeed Keywords:\n%s\n\nGenerate the expanded keyword list with ROI scoring now.",
		topic, strings.Join(seedTexts, "\n"))

	response := callResearchAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		return nil, fmt.Errorf("AI returned empty response")
	}

	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in AI response")
	}

	var expanded []ExpandedKeyword
	if err := json.Unmarshal([]byte(jsonStr), &expanded); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return expanded, nil
}

// clusterKeywords 基于语义相似度自动聚类
func clusterKeywords(keywords []ExpandedKeyword) []TopicCluster {
	// 简单规则聚类（可后续升级为向量聚类）
	clusters := []TopicCluster{
		{
			PillarKeyword:   "AI Video Prompt Library",
			PillarSearchVol: "5K-10K",
			SuggestedURL:    "/prompts",
			ContentType:     "prompt_detail",
		},
		{
			PillarKeyword:   "Node Canvas AI Video Creation",
			PillarSearchVol: "500-2K",
			SuggestedURL:    "/article/node-canvas-video-creation",
			ContentType:     "article",
		},
		{
			PillarKeyword:   "Best AI Video Generators 2026",
			PillarSearchVol: "8K-15K",
			SuggestedURL:    "/article/best-ai-video-generators-2026",
			ContentType:     "article",
		},
		{
			PillarKeyword:   "AI Video Prompt Engineering Guide",
			PillarSearchVol: "2K-5K",
			SuggestedURL:    "/article/ai-video-prompt-engineering-guide",
			ContentType:     "article",
		},
	}

	// 将关键词分配到对应簇
	for _, kw := range keywords {
		for i := range clusters {
			if shouldBelongToCluster(kw, clusters[i]) {
				clusters[i].ClusterKeywords = append(clusters[i].ClusterKeywords, kw.Keyword)
			}
		}
	}

	return clusters
}

// shouldBelongToCluster 判断关键词是否属于某个簇
func shouldBelongToCluster(kw ExpandedKeyword, cluster TopicCluster) bool {
	pillar := strings.ToLower(cluster.PillarKeyword)
	keyword := strings.ToLower(kw.Keyword)

	switch {
	case strings.Contains(pillar, "prompt library"):
		return strings.Contains(keyword, "prompt") && !strings.Contains(keyword, "tutorial") && !strings.Contains(keyword, "guide")
	case strings.Contains(pillar, "node canvas"):
		return strings.Contains(keyword, "node") || strings.Contains(keyword, "canvas") || strings.Contains(keyword, "workflow")
	case strings.Contains(pillar, "generators"):
		return strings.Contains(keyword, "best") || strings.Contains(keyword, "comparison") || strings.Contains(keyword, "vs")
	case strings.Contains(pillar, "engineering"):
		return strings.Contains(keyword, "how to") || strings.Contains(keyword, "guide") || strings.Contains(keyword, "tips")
	}
	return false
}

// analyzeCompetitors 返回已知竞品的洞察
func analyzeCompetitors(topic string) []CompetitorInsight {
	return []CompetitorInsight{
		{
			Domain: "videoprompt.app",
			Strengths: []string{"500+ prompts", "10+ models", "clean UI"},
			Weaknesses: []string{"No Schema markup", "No multi-language", "Limited content depth"},
			RankedKeywords: []string{"AI video prompt library", "Sora prompt generator", "video prompt collection"},
		},
		{
			Domain: "openpromptlib.com",
			Strengths: []string{"4,379 video prompts", "API access", "weekly curation"},
			Weaknesses: []string{"Seedance bias", "No category navigation", "No Schema"},
			RankedKeywords: []string{"AI video prompts", "curated prompts", "Seedance 2.0 prompts"},
		},
		{
			Domain: "promptbase.com",
			Strengths: []string{"6,600+ prompts", "established marketplace", "strong brand"},
			Weaknesses: []string{"Paid per prompt", "limited free browsing", "no educational content"},
			RankedKeywords: []string{"buy Sora prompts", "AI video prompt marketplace", "premium video prompts"},
		},
	}
}

// findContentGaps 找出内容缺口
func findContentGaps(keywords []ExpandedKeyword, clusters []TopicCluster) []ContentGap {
	var gaps []ContentGap

	// P0: 高 ROI 但尚未覆盖
	for _, kw := range keywords {
		if kw.ROIScore >= 80 && kw.Difficulty < 50 {
			gaps = append(gaps, ContentGap{
				Keyword:    kw.Keyword,
				Priority:   "P0",
				Competitor: "multiple",
				OurStatus:  "not_started",
				Action:     fmt.Sprintf("Create %s content targeting '%s'", kw.Category, kw.Keyword),
			})
		}
	}

	// P1: 模型专属词
	for _, kw := range keywords {
		if kw.ModelTag != "multi" && kw.ModelTag != "none" && kw.ROIScore >= 60 {
			gaps = append(gaps, ContentGap{
				Keyword:    kw.Keyword,
				Priority:   "P1",
				Competitor: "videoprompt.app",
				OurStatus:  "not_started",
				Action:     fmt.Sprintf("Add %s-specific prompt collection", kw.ModelTag),
			})
		}
	}

	return gaps
}

// ========== AI 调用辅助 ==========

func callResearchAI(cfg operation_setting.TranslateSetting, systemPrompt, userPrompt string) string {
	payload := map[string]interface{}{
		"model": cfg.TranslateAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", cfg.TranslateAIBaseURL+"/chat/completions", bytes.NewReader(payloadBytes))
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+cfg.TranslateAIApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content
	}
	return ""
}

func extractJSON(text string) string {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start != -1 && end != -1 && end > start {
		return text[start : end+1]
	}
	// 尝试对象格式
	start = strings.Index(text, "{")
	end = strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return text[start : end+1]
	}
	return ""
}
```

#### 新增控制器：`controller/seo_research.go`

```go
package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ResearchKeywordsRequest 关键词研究请求
type ResearchKeywordsRequest struct {
	Topic       string   `json:"topic" binding:"required"`        // 研究主题
	TargetLangs []string `json:"target_langs"`                    // 目标语言，默认 ["en"]
	MaxKeywords int      `json:"max_keywords" binding:"max=200"`  // 最大生成关键词数，默认 100
}

// ResearchKeywords 执行自动化关键词研究
// POST /api/admin/seo/research
func ResearchKeywords(c *gin.Context) {
	var req ResearchKeywordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.TargetLangs) == 0 {
		req.TargetLangs = []string{"en"}
	}
	if req.MaxKeywords == 0 {
		req.MaxKeywords = 100
	}

	result, err := service.ResearchKeywords(req.Topic, req.TargetLangs)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, result)
}

// GetResearchTemplate 获取研究模板（预置的种子词和主题簇模板）
// GET /api/admin/seo/research/templates
func GetResearchTemplates(c *gin.Context) {
	templates := []gin.H{
		{
			"id":          "ai-video-prompts",
			"name":        "AI Video Prompt Library",
			"topic":       "AI video prompt library",
			"description": "研究 AI 视频提示词相关的关键词机会",
			"pillars": []string{
				"AI Video Prompt Library",
				"Sora Prompt Templates",
				"Kling Prompt Guide",
				"Veo 3 Prompt Examples",
			},
		},
		{
			"id":          "node-canvas",
			"name":        "Node Canvas Video Creation",
			"topic":       "node canvas AI video creation",
			"description": "研究节点画布/工作流相关的蓝海关键词",
			"pillars": []string{
				"Node Canvas Video Editor",
				"ComfyUI Video Workflow",
				"AI Storyboard Creator",
				"Visual AI Pipeline Builder",
			},
		},
		{
			"id":          "tool-comparison",
			"name":        "AI Video Tools Comparison",
			"topic":       "best AI video generators comparison 2026",
			"description": "研究 AI 视频工具横评类关键词（截流竞品）",
			"pillars": []string{
				"Best AI Video Generators 2026",
				"Sora vs Kling vs Veo",
				"Free AI Video Tools",
				"AI Video Tool for Beginners",
			},
		},
		{
			"id":          "prompt-engineering",
			"name":        "AI Video Prompt Engineering",
			"topic":       "AI video prompt engineering guide",
			"description": "研究提示词工程/教程类关键词",
			"pillars": []string{
				"AI Video Prompt Engineering",
				"How to Write AI Video Prompts",
				"AI Video Prompt Tips",
				"Cinematic AI Video Prompts",
			},
		},
	}

	common.ApiSuccess(c, templates)
}
```

#### 路由注册：`router/api-router.go` 新增

```go
// SEO 研究自动化
adminRoute.POST("/seo/research", controller.ResearchKeywords)
adminRoute.GET("/seo/research/templates", controller.GetResearchTemplates)
```

### 2.3 使用流程（人工仅需 3 步）

```
Step 1: 管理员登录后台 → SEO 研究面板
Step 2: 选择模板（如 "AI Video Prompt Library"）或输入自定义主题
Step 3: 点击「开始研究」→ AI 自动生成完整报告（约 30-60 秒）

输出：
├── 种子词清单（自动按 A/B/C/D 维度分类）
├── 扩展关键词表（含搜索量/难度/意图/ROI 评分）
├── 主题簇地图（Pillar + Cluster + 建议 URL）
├── 竞品洞察（3 个核心竞品的强弱分析）
├── 内容缺口清单（按 P0/P1/P2 排序）
└── 一键导出 CSV（给内容团队直接使用）
```

---

## 三、Phase 2：内容生成自动化（70% 自动化）

### 3.1 已有基础

| 能力 | 已有文件 | 状态 |
|------|----------|------|
| GEO 结构化内容生成 | `service/geo_blocks_generator.go` | ✅ 已上线 |
| 批量 GEO 生成 API | `controller/geo_blocks.go` | ✅ 已上线 |
| 自动翻译队列 | `service/auto_translate_queue.go` | ✅ 已上线 |
| 批量 SEO 翻译 | `service/seo_batch_translate.go` | ✅ 已上线 |
| Prompt/Article CRUD | `controller/prompt.go`, `controller/article.go` | ✅ 已上线 |

### 3.2 需要增强的能力

#### 3.2.1 新增：AI 文章自动生成服务

基于关键词研究输出，自动生成 SEO 文章草稿。

```go
// service/content_generator.go

// GenerateArticleFromKeyword 根据关键词自动生成文章
func GenerateArticleFromKeyword(keyword string, cluster string, contentType string) (*model.Article, error) {
	cfg := operation_setting.GetTranslateSetting()
	
	systemPrompt := fmt.Sprintf(`You are an expert SEO content writer specializing in AI video generation.
Write a comprehensive, SEO-optimized article targeting the keyword: "%s".

Requirements:
1. Total length: 2000-3000 words
2. Include H1 (article title), H2 (main sections), H3 (subsections)
3. First 100 words must naturally include the target keyword
4. Include at least 3 internal link placeholders: [LINK:prompt-slug] or [LINK:article-slug]
5. Include a FAQ section with 5-7 questions
6. End with a strong CTA referencing harse.tv's prompt library
7. Tone: professional but accessible, suitable for content creators and marketers
8. Include specific examples, tool names, and actionable tips

Content type: %s
Topic cluster: %s

Output format: JSON with fields:
- "title": string (SEO-optimized H1, 50-60 chars)
- "summary": string (150-200 chars meta description)
- "content": string (full article in Markdown)
- "seo_keywords": string (comma-separated, 5-8 keywords)
- "faq": string (JSON array of {question, answer} objects)
- "suggested_slug": string (URL-friendly slug)`, keyword, contentType, cluster)

	// ... 调用 AI 生成文章
}
```

#### 3.2.2 新增：AI 提示词自动生成服务

```go
// GeneratePromptFromKeyword 根据关键词自动生成提示词
func GeneratePromptFromKeyword(keyword string, model string, categoryID int) (*model.Prompt, error) {
	cfg := operation_setting.GetTranslateSetting()
	
	systemPrompt := fmt.Sprintf(`You are an expert prompt engineer for AI video generation.
Create a high-quality, detailed prompt targeting: "%s" for the %s model.

Requirements:
1. Prompt title: Clear, descriptive, includes model name
2. Prompt content: Detailed, structured, with variable placeholders like [subject], [style], [lighting]
3. Description: 2-3 sentences explaining what this prompt creates
4. Variables: JSON array of variable definitions
5. Tags: Array of relevant tags
6. Media type: "video"
7. SEO keywords: 5-8 related keywords
8. Intro: 50-100 word introduction for SEO
9. FAQ: 3-5 common questions about using this prompt

Output format: JSON with all Prompt model fields.`, keyword, model)

	// ... 调用 AI 生成提示词
}
```

#### 3.2.3 新增：内容生成控制器

```go
// controller/content_generator.go

// GenerateArticleRequest 自动生成文章请求
type GenerateArticleRequest struct {
	Keyword     string `json:"keyword" binding:"required"`
	Cluster     string `json:"cluster"`
	ContentType string `json:"content_type"` // article/prompt
	AutoPublish bool   `json:"auto_publish"` // 是否自动发布
}

// GenerateArticleFromKeyword 根据关键词自动生成文章
// POST /api/admin/content/generate/article
func GenerateArticleFromKeyword(c *gin.Context) {
	var req GenerateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	article, err := service.GenerateArticleFromKeyword(req.Keyword, req.Cluster, req.ContentType)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	// 保存到数据库
	if err := article.Insert(); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	// 自动触发后续流程
	go func() {
		// 1. 生成 GEO 结构化内容
		service.GenerateArticleGeoBlocks(article.Id)
		
		// 2. 启动自动翻译队列
		service.StartAutoTranslate("article", article.Id)
		
		// 3. 生成 SEO 关键词和介绍
		service.GenerateSEOForArticle(article.Id)
	}()

	common.ApiSuccess(c, gin.H{
		"id":      article.Id,
		"title":   article.Title,
		"slug":    article.Slug,
		"message": "文章生成成功，已启动自动 SEO 优化和翻译",
	})
}

// BatchGenerateContentRequest 批量内容生成请求
type BatchGenerateContentRequest struct {
	Keywords    []string `json:"keywords" binding:"required"`
	ContentType string   `json:"content_type"` // article/prompt
	AutoPublish bool     `json:"auto_publish"`
}

// BatchGenerateContent 批量根据关键词生成内容
// POST /api/admin/content/generate/batch
func BatchGenerateContent(c *gin.Context) {
	var req BatchGenerateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.Keywords) == 0 {
		common.ApiErrorMsg(c, "keywords 不能为空")
		return
	}
	if len(req.Keywords) > 20 {
		common.ApiErrorMsg(c, "单次最多 20 个关键词")
		return
	}

	var results []gin.H
	for _, kw := range req.Keywords {
		if req.ContentType == "prompt" {
			// 生成提示词...
		} else {
			// 生成文章...
		}
	}

	common.ApiSuccess(c, gin.H{
		"total":    len(req.Keywords),
		"results":  results,
		"message":  fmt.Sprintf("已启动 %d 个内容生成任务", len(req.Keywords)),
	})
}
```

### 3.3 内容生成流水线（完全自动化）

```
触发条件：Phase 1 输出的内容缺口清单 或 管理员手动输入关键词

┌─────────────────────────────────────────────────────────────┐
│ Step 1: AI 生成内容草稿                                       │
│ 输入：关键词 + 内容类型（article/prompt）                       │
│ 输出：Markdown 文章 / Prompt JSON                             │
│ 自动化率：90%（AI 生成，人工审核）                              │
├─────────────────────────────────────────────────────────────┤
│ Step 2: 自动保存到数据库                                       │
│ 操作：INSERT INTO prompts/articles                            │
│ 自动化率：100%                                                │
├─────────────────────────────────────────────────────────────┤
│ Step 3: 自动生成 SEO 字段（触发器）                             │
│ 操作：AI 生成 seo_keywords, intro, faq                        │
│ 自动化率：100%（复用现有 batch translate）                     │
├─────────────────────────────────────────────────────────────┤
│ Step 4: 自动生成 GEO 结构化内容（触发器）                       │
│ 操作：AI 生成 geo_blocks (what/why/how/summary)               │
│ 自动化率：100%（复用现有 geo_blocks_generator）                │
├─────────────────────────────────────────────────────────────┤
│ Step 5: 自动启动翻译队列（触发器）                              │
│ 操作：StartAutoTranslate("article/prompt", id)                │
│ 自动化率：100%（复用现有 auto_translate_queue）                │
├─────────────────────────────────────────────────────────────┤
│ Step 6: 翻译完成后自动更新 Sitemap                              │
│ 操作：Sitemap 自动包含新 slug                                  │
│ 自动化率：100%（复用现有 Sitemap 生成）                        │
├─────────────────────────────────────────────────────────────┤
│ Step 7: 提交 Google Indexing API                              │
│ 操作：自动提交新 URL 到 Google                                │
│ 自动化率：100%（需新建）                                      │
└─────────────────────────────────────────────────────────────┘
```

---

## 四、Phase 3：SEO 优化自动化（90% 自动化）

### 4.1 已有能力（几乎全覆盖）

| SEO 元素 | 已有实现 | 自动化状态 |
|----------|----------|-----------|
| **Schema.org JSON-LD** | `controller/misc.go` + `SchemaOrg.jsx` | ✅ 自动注入（Article/FAQPage/WebPage） |
| **Meta Title/Description** | `model.article.go` seo_title/seo_description | ✅ 批量翻译自动生成 |
| **SEO Keywords** | `model.prompt.go` / `model.article.go` seo_keywords | ✅ 批量翻译自动生成 |
| **FAQ Schema** | `model.prompt.go` faq 字段 → FAQPage Schema | ✅ 有 FAQ 则自动渲染 |
| **GEO Blocks** | `service/geo_blocks_generator.go` | ✅ AI 自动生成 + 多语言翻译 |
| **Canonical URL** | `web/src/components/seo/SEO.jsx` | ✅ 自动计算 |
| **Hreflang** | `web/src/components/seo/SEO.jsx` | ✅ 12 语种自动渲染 |
| **Open Graph / Twitter Card** | `web/src/components/seo/SEO.jsx` | ✅ 自动渲染 |
| **Sitemap** | `controller/misc.go` GetSitemap | ✅ 自动包含所有 public 页面 |
| **Slug** | `model.prompt.go` / `model.article.go` | ✅ 创建时自动生成 |

### 4.2 需新增的自动化能力

#### 4.2.1 自动内链建议服务

```go
// service/auto_internal_link.go

// SuggestInternalLinks 为内容自动推荐内链
func SuggestInternalLinks(content string, contentType string, currentID int) []InternalLinkSuggestion {
	// 1. 提取内容中的关键词
	keywords := extractKeywords(content)
	
	// 2. 查询数据库中已有的相关 Prompt/Article
	var suggestions []InternalLinkSuggestion
	
	// 查询匹配的 prompts
	var prompts []model.Prompt
	model.DB.Where("status = ? AND id != ?", 1, currentID).
		Where("title LIKE ? OR seo_keywords LIKE ?", "%"+keywords[0]+"%", "%"+keywords[0]+"%").
		Limit(5).Find(&prompts)
	
	for _, p := range prompts {
		suggestions = append(suggestions, InternalLinkSuggestion{
			AnchorText: p.Title,
			URL:        fmt.Sprintf("/prompt/%s", p.Slug),
			Type:       "prompt",
			Relevance:  calculateRelevance(content, p.Title, p.SeoKeywords),
		})
	}
	
	// 查询匹配的 articles
	var articles []model.Article
	model.DB.Where("status = ? AND id != ?", 1, currentID).
		Where("title LIKE ? OR seo_keywords LIKE ?", "%"+keywords[0]+"%", "%"+keywords[0]+"%").
		Limit(5).Find(&articles)
	
	for _, a := range articles {
		suggestions = append(suggestions, InternalLinkSuggestion{
			AnchorText: a.Title,
			URL:        fmt.Sprintf("/article/%s", a.Slug),
			Type:       "article",
			Relevance:  calculateRelevance(content, a.Title, a.SeoKeywords),
		})
	}
	
	// 按相关度排序，取前 5
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Relevance > suggestions[j].Relevance
	})
	
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	
	return suggestions
}
```

#### 4.2.2 自动 SEO 审计（定期任务）

```go
// service/seo_audit.go

// RunSEOAudit 定期执行 SEO 审计
func RunSEOAudit() {
	// 1. 检查是否有页面缺少 Schema
	// 2. 检查是否有页面缺少 hreflang
	// 3. 检查是否有页面缺少 Meta Description
	// 4. 检查死链
	// 5. 检查重复内容
	// 6. 生成审计报告
}
```

### 4.3 新增前端：SEO 优化面板

在管理后台新增「SEO 优化中心」：

```jsx
// web/src/pages/SEODashboard/index.jsx

// 功能模块：
// 1. SEO 健康度评分（实时）
// 2. 缺少 SEO 字段的内容列表（一键批量生成）
// 3. 内链建议（为每篇文章推荐 3-5 个内链目标）
// 4. 关键词排名监控（集成 Google Search Console API）
// 5. 内容缺口追踪（与 Phase 1 的研究报告联动）
```

---

## 五、Phase 4：多语言发布自动化（95% 自动化）

### 5.1 已有能力（非常完善）

| 能力 | 实现 | 状态 |
|------|------|------|
| **自动翻译队列** | `service/auto_translate_queue.go` | ✅ 创建记录后自动触发 |
| **批量翻译 API** | `controller/prompt.go` / `controller/article.go` | ✅ 支持批量操作 |
| **多语言字段** | i18n / title_i18n / seo_i18n / geo_blocks_i18n | ✅ 4 层多语言覆盖 |
| **语言切换** | `ApplyLanguage(lang)` | ✅ 后端自动切换 |
| **hreflang** | `SEO.jsx` | ✅ 前端自动渲染 |
| **翻译状态追踪** | `is_translated` + `translation_error` | ✅ 实时状态 |

### 5.2 需要增强的自动化

#### 5.2.1 翻译完成后自动发布

```go
// 在 auto_translate_queue.go 的 processAutoTranslate 中，翻译完成后：

func processAutoTranslate(task *AutoTranslateTask, runningKey string) {
	// ... 现有翻译逻辑 ...
	
	// 新增：翻译完成后自动触发 SEO 字段生成
	if task.Type == "prompt" {
		// 如果 SEO 字段为空，自动补全
		var p model.Prompt
		model.DB.First(&p, task.RecordID)
		if p.SeoKeywords == "" {
			go service.GenerateSEOForPrompt(task.RecordID)
		}
		if p.GeoBlocks == "" {
			go service.GeneratePromptGeoBlocks(task.RecordID)
		}
	} else if task.Type == "article" {
		var a model.Article
		model.DB.First(&a, task.RecordID)
		if a.SeoKeywords == "" {
			go service.GenerateSEOForArticle(task.RecordID)
		}
		if a.GeoBlocks == "" {
			go service.GenerateArticleGeoBlocks(task.RecordID)
		}
	}
	
	// 新增：翻译完成后自动提交 Google Indexing API
	go submitToGoogleIndexing(task.Type, task.RecordID)
}
```

#### 5.2.2 新增：Google Indexing API 提交

```go
// service/google_indexing.go

import (
	"context"
	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/indexing/v3"
)

// SubmitURLToGoogle 提交 URL 到 Google Indexing API
func SubmitURLToGoogle(url string) error {
	// 读取服务账号 JSON
	data, err := ioutil.ReadFile("config/google-service-account.json")
	if err != nil {
		return fmt.Errorf("failed to read service account: %w", err)
	}

	config, err := google.JWTConfigFromJSON(data, indexing.IndexingScope)
	if err != nil {
		return fmt.Errorf("failed to parse service account: %w", err)
	}

	client := config.Client(context.Background())
	service, err := indexing.New(client)
	if err != nil {
		return fmt.Errorf("failed to create indexing service: %w", err)
	}

	notification := &indexing.UrlNotification{
		Url:  url,
		Type: "URL_UPDATED",
	}

	_, err = service.UrlNotifications.Publish(notification).Do()
	if err != nil {
		return fmt.Errorf("failed to submit URL: %w", err)
	}

	return nil
}

// submitToGoogleIndexing 根据记录类型和 ID 提交 URL
func submitToGoogleIndexing(recordType string, recordID int) {
	baseURL := "https://harse.tv" // 从配置读取
	
	var slug string
	if recordType == "prompt" {
		var p model.Prompt
		if err := model.DB.Select("slug").First(&p, recordID).Error; err == nil {
			slug = p.Slug
		}
	} else if recordType == "article" {
		var a model.Article
		if err := model.DB.Select("slug").First(&a, recordID).Error; err == nil {
			slug = a.Slug
		}
	}
	
	if slug == "" {
		return
	}
	
	// 提交所有语言版本
	langs := []string{"", "en", "ja", "ko", "de", "fr", "es", "pt", "ru", "ar", "hi", "tr", "id"}
	for _, lang := range langs {
		url := fmt.Sprintf("%s/%s/%s", baseURL, recordType, slug)
		if lang != "" {
			url = fmt.Sprintf("%s/%s/%s/%s", baseURL, lang, recordType, slug)
		}
		if err := SubmitURLToGoogle(url); err != nil {
			common.SysLog(fmt.Sprintf("Google Indexing failed for %s: %v", url, err))
		}
	}
}
```

### 5.3 多语言发布流水线

```
中文内容创建
     │
     ▼
┌─────────────────┐
│ 自动翻译队列启动  │ ← 创建/更新记录时自动触发
│ (12 种语言)      │
└─────────────────┘
     │
     ▼
┌─────────────────┐
│ 翻译完成         │
└─────────────────┘
     │
     ├──→ 自动补全 SEO 字段（如果缺失）
     ├──→ 自动补全 GEO 结构化内容（如果缺失）
     ├──→ 自动更新 Sitemap（包含所有语言版本）
     └──→ 自动提交 Google Indexing API（所有语言版本）
     │
     ▼
┌─────────────────┐
│ 内容上线         │ ← 无需人工干预
│ (13 个语言版本)  │
└─────────────────┘
```

---

## 六、Phase 5：索引监控自动化（85% 自动化）

### 6.1 新增：SEO 监控服务

```go
// service/seo_monitor.go

type SEOMonitorReport struct {
	Date              time.Time              `json:"date"`
	TotalPages        int                    `json:"total_pages"`
	IndexedPages      int                    `json:"indexed_pages"`      // Google 已索引
	IndexedRatio      float64                `json:"indexed_ratio"`
	TopKeywords       []KeywordRanking       `json:"top_keywords"`
	NewKeywords       []string               `json:"new_keywords"`
	LostKeywords      []string               `json:"lost_keywords"`
	AvgPosition       float64                `json:"avg_position"`
	OrganicTraffic    int                    `json:"organic_traffic"`
	Issues            []SEOIssue             `json:"issues"`
}

type KeywordRanking struct {
	Keyword   string  `json:"keyword"`
	Position  float64 `json:"position"`
	Clicks    int     `json:"clicks"`
	Impressions int   `json:"impressions"`
	CTR       float64 `json:"ctr"`
}

type SEOIssue struct {
	Severity string `json:"severity"` // critical/warning/info
	Type     string `json:"type"`     // noindex/missing_schema/slow_page/etc
	URL      string `json:"url"`
	Message  string `json:"message"`
}

// FetchGoogleSearchConsoleData 从 GSC 获取数据
func FetchGoogleSearchConsoleData(siteURL string, startDate, endDate string) (*SEOMonitorReport, error) {
	// 使用 Google Search Console API 获取数据
	// 需要配置 OAuth2 凭证
}

// RunDailySEOMonitor 每日运行 SEO 监控
func RunDailySEOMonitor() {
	report, err := FetchGoogleSearchConsoleData("https://harse.tv/", 
		time.Now().AddDate(0, 0, -7).Format("2006-01-02"),
		time.Now().Format("2006-01-02"))
	
	if err != nil {
		common.SysLog(fmt.Sprintf("SEO monitor failed: %v", err))
		return
	}
	
	// 保存报告到数据库
	saveSEOMonitorReport(report)
	
	// 如果有严重问题，发送通知
	for _, issue := range report.Issues {
		if issue.Severity == "critical" {
			sendAlert(fmt.Sprintf("SEO Critical Issue: %s on %s", issue.Message, issue.URL))
		}
	}
}
```

### 6.2 新增：监控仪表盘 API

```go
// controller/seo_monitor.go

// GetSEODashboard 获取 SEO 监控仪表盘数据
// GET /api/admin/seo/dashboard
func GetSEODashboard(c *gin.Context) {
	// 返回综合 SEO 数据
}

// GetSEOReport 获取历史 SEO 报告
// GET /api/admin/seo/reports?start=2026-01-01&end=2026-06-30
func GetSEOReport(c *gin.Context) {
	// 返回历史监控数据
}
```

### 6.3 自动化 cron 任务

```go
// 在 main.go 或专门的 cron 文件中注册

func initSEOCronJobs() {
	// 每日 2:00 AM 执行 SEO 监控
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
			time.Sleep(time.Until(next))
			
			service.RunDailySEOMonitor()
		}
	}()
	
	// 每周一 3:00 AM 执行 SEO 审计
	go func() {
		for {
			now := time.Now()
			// 计算下周一
			daysUntilMonday := (1 - int(now.Weekday()) + 7) % 7
			if daysUntilMonday == 0 {
				daysUntilMonday = 7
			}
			next := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 3, 0, 0, 0, now.Location())
			time.Sleep(time.Until(next))
			
			service.RunSEOAudit()
		}
	}()
}
```

---

## 七、完整执行计划

### 第一阶段：基础设施（Week 1-2）

| # | 任务 | 文件 | 工作量 | 优先级 |
|---|------|------|--------|--------|
| 1 | 新建 `service/seo_research.go` | 新增 | 2 天 | P0 |
| 2 | 新建 `controller/seo_research.go` + 路由 | 新增 | 1 天 | P0 |
| 3 | 新建 `service/content_generator.go` | 新增 | 2 天 | P0 |
| 4 | 新建 `controller/content_generator.go` + 路由 | 新增 | 1 天 | P0 |
| 5 | 新建 `service/google_indexing.go` | 新增 | 1 天 | P1 |
| 6 | 新增前端：SEO 研究面板 | web 新增 | 2 天 | P1 |
| 7 | 新增前端：内容生成面板 | web 新增 | 2 天 | P1 |
| 8 | 配置 Google Service Account | 配置 | 0.5 天 | P1 |

### 第二阶段：自动化增强（Week 3-4）

| # | 任务 | 文件 | 工作量 | 优先级 |
|---|------|------|--------|--------|
| 9 | 翻译完成后自动补全 SEO/GEO | 修改 `auto_translate_queue.go` | 1 天 | P0 |
| 10 | 翻译完成后自动提交 Google Indexing | 修改 `auto_translate_queue.go` | 1 天 | P0 |
| 11 | 新建 `service/auto_internal_link.go` | 新增 | 1 天 | P1 |
| 12 | 新建 `service/seo_monitor.go` | 新增 | 2 天 | P1 |
| 13 | 新建 `controller/seo_monitor.go` + 路由 | 新增 | 1 天 | P1 |
| 14 | 新增前端：SEO 监控仪表盘 | web 新增 | 2 天 | P1 |
| 15 | 注册 Cron 任务 | 修改 `main.go` | 0.5 天 | P2 |

### 第三阶段：内容填充（Week 5-8）

| # | 任务 | 策略 | 预期产出 |
|---|------|------|----------|
| 16 | 执行 Phase 1：关键词研究 | 使用自动化研究工具 | 300+ 关键词清单 + 4 个主题簇 |
| 17 | 执行 Phase 2：批量内容生成 | 批量生成文章 + 提示词 | 20 篇文章 + 100 个提示词 |
| 18 | 执行 Phase 3：自动 SEO 优化 | 批量触发 GEO + SEO 翻译 | 全部内容自动优化 |
| 19 | 执行 Phase 4：多语言发布 | 自动翻译队列运行 | 13 语言版本全部上线 |
| 20 | 执行 Phase 5：索引监控 | Google Indexing API + GSC | 所有页面被 Google 索引 |

### 第四阶段：持续优化（Month 3-6）

| # | 任务 | 频率 | 自动化率 |
|---|------|------|----------|
| 21 | 关键词研究更新 | 每月 1 次 | 80% |
| 22 | 新内容生成 | 每周 5-10 篇 | 70% |
| 23 | SEO 审计 | 每周 1 次 | 90% |
| 24 | 排名监控 | 每日 | 85% |
| 25 | 内容更新 | 每季度 | 60% |

---

## 八、预期效果

### 8.1 自动化节省的人力

| 任务 | 手动耗时 | 自动化后 | 节省 |
|------|----------|----------|------|
| 关键词研究（300 词） | 16 小时 | 3 小时（审核） | **81%** |
| 内容生成（20 篇文章） | 40 小时 | 8 小时（审核） | **80%** |
| SEO 优化（120 个页面） | 24 小时 | 2 小时（审核） | **92%** |
| 多语言翻译（12 语种） | 120 小时 | 0 小时 | **100%** |
| GEO 结构化内容 | 40 小时 | 0 小时 | **100%** |
| Google 索引提交 | 4 小时 | 0 小时 | **100%** |
| **总计** | **244 小时** | **13 小时** | **95%** |

### 8.2 流量增长预期

| 时间点 | 索引页面 | 排名关键词 | 月自然流量 | 备注 |
|--------|----------|-----------|------------|------|
| Month 0 | ~10 | 0 | ~0 | 现状 |
| Month 1 | 100+ | 20-50 | 500-2K | P0 内容上线 |
| Month 2 | 300+ | 100-200 | 3K-8K | 多语言生效 |
| Month 3 | 500+ | 300-500 | 10K-25K | 长尾词积累 |
| Month 6 | 1000+ | 800-1500 | 30K-80K | 持续内容输出 |
| Month 12 | 2000+ | 2000+ | 80K-200K | 权威度建立 |

---

## 九、文件清单（新增/修改）

### 新增文件（Backend）

```
service/seo_research.go              # Phase 1: 关键词研究自动化
service/content_generator.go          # Phase 2: 内容生成自动化
service/auto_internal_link.go         # Phase 3: 自动内链建议
service/google_indexing.go            # Phase 4: Google Indexing API
service/seo_monitor.go                # Phase 5: SEO 监控
service/seo_audit.go                  # Phase 3: 定期 SEO 审计
controller/seo_research.go            # Phase 1 API
controller/content_generator.go       # Phase 2 API
controller/seo_monitor.go             # Phase 5 API
```

### 修改文件（Backend）

```
service/auto_translate_queue.go       # 翻译完成后自动触发 SEO/GEO/Indexing
router/api-router.go                  # 注册新路由
main.go                               # 注册 Cron 任务
```

### 新增文件（Frontend）

```
web/src/pages/SEODashboard/index.jsx       # SEO 监控仪表盘
web/src/pages/SEOResearch/index.jsx        # 关键词研究面板
web/src/pages/ContentGenerator/index.jsx   # 内容生成面板
```

---

*本方案基于 SOP v2.0 海外市场版制定*  
*作者：SEO 内容营销团队*  
*版本：v1.0 | 日期：2026-06-11*
