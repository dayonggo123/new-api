package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const seoResearchTimeout = 120 * time.Second

// SEOResearchRequest 关键词研究请求
type SEOResearchRequest struct {
	SeedKeyword string `json:"seed_keyword" binding:"required"`
	Language    string `json:"language"` // 默认 "en"
}

// ResearchKeywords 执行 AI 关键词研究
func ResearchKeywords(req *SEOResearchRequest) (*model.SEOKeywordResearchResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return nil, fmt.Errorf("seo ai not configured: please configure SEO AI settings first")
	}

	lang := req.Language
	if lang == "" {
		lang = "en"
	}

	systemPrompt := buildSEOResearchSystemPrompt(lang)
	userPrompt := buildSEOResearchUserPrompt(req.SeedKeyword, lang)

	reqBody := map[string]interface{}{
		"model": cfg.SeoAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.5,
		"max_tokens":  4000,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: seoResearchTimeout}
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

	var result model.SEOKeywordResearchResult
	if err := common.Unmarshal([]byte(content), &result); err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("parse seo research json failed: %v, content=%s", err, content))
		// 尝试二次解析：AI 可能返回了不完全符合结构的数据
		return fallbackParseResearchResult(content, req.SeedKeyword, lang)
	}

	// 补全元数据
	result.SeedKeyword = req.SeedKeyword
	result.Language = lang

	// 自动补全高 ROI 关键词（若 AI 未生成）
	if len(result.HighROIKeywords) == 0 {
		allKeywords := append(result.SeedKeywords, result.ExtendedKeywords...)
		allKeywords = append(allKeywords, result.LongTailKeywords...)
		// 按 ROI 分数降序选取前 10
		sort.Slice(allKeywords, func(i, j int) bool {
			return allKeywords[i].ROIScore > allKeywords[j].ROIScore
		})
		limit := 10
		if len(allKeywords) < limit {
			limit = len(allKeywords)
		}
		for i := 0; i < limit; i++ {
			allKeywords[i].BusinessValue = 8
			if allKeywords[i].ROIScore < 60 {
				allKeywords[i].ROIScore = 70
			}
			result.HighROIKeywords = append(result.HighROIKeywords, allKeywords[i])
		}
	}

	// 自动补全主题簇（若 AI 未生成）
	if len(result.TopicClusters) == 0 {
		result.TopicClusters = generateDefaultTopicClusters(result.SeedKeywords, result.ExtendedKeywords)
	}

	result.TotalCount = len(result.SeedKeywords) + len(result.ExtendedKeywords) + len(result.LongTailKeywords)
	result.HighROICount = len(result.HighROIKeywords)
	result.ClusterCount = len(result.TopicClusters)

	return &result, nil
}

// buildSEOResearchSystemPrompt 构建关键词研究系统提示
func buildSEOResearchSystemPrompt(lang string) string {
	return fmt.Sprintf(`You are an expert SEO keyword researcher specializing in the AI video creation and prompt library space. Your task is to perform comprehensive keyword research for the given topic.

Target market: Global (Google Search), primary language: %s

Analyze the following dimensions and return ONLY valid JSON (no markdown, no explanation):

1. seed_keywords: 5-8 core/seed keywords directly related to the topic (include brand names, product types, model names)
2. extended_keywords: 8-12 extended keywords (variations, related concepts, use cases)
3. long_tail_keywords: 10-15 long-tail keywords (specific questions, detailed queries, problem-solving phrases)
4. high_roi_keywords: Top 8-10 keywords with highest ROI potential (high business value + manageable competition)
5. topic_clusters: 3-5 topic clusters, each with a pillar keyword and supporting cluster keywords
6. content_gaps: 3-5 content gaps that competitors likely haven't covered well

For each keyword item, provide:
- keyword: the exact search term
- search_volume: estimated monthly search volume (realistic numbers, not inflated)
- intent: one of [informational, navigational, transactional, commercial]
- difficulty: one of [low, medium, high]
- business_value: 1-10 score
- roi_score: 0-100 score
- suggested_url: suggested URL slug for content targeting this keyword

For topic_clusters:
- name: cluster name
- pillar_keyword: main pillar page keyword
- pillar_volume: search volume for pillar keyword
- cluster_keywords: array of supporting keywords
- content_type: one of [article, prompt, tool, landing]
- priority: one of [P0, P1, P2]

For content_gaps:
- keyword: the gap keyword
- volume: estimated search volume
- competitors: brief description of who covers this (or "none" if untouched)
- gap_type: one of [missing_topic, undercovered, poor_quality]
- priority: one of [P0, P1, P2]
- suggested_action: brief description of what content to create

JSON format:
{
  "seed_keywords": [{"keyword":"...","search_volume":1200,"intent":"informational","difficulty":"low","business_value":8,"roi_score":85,"suggested_url":"..."},...],
  "extended_keywords": [...],
  "long_tail_keywords": [...],
  "high_roi_keywords": [...],
  "topic_clusters": [{"name":"...","pillar_keyword":"...","pillar_volume":5000,"cluster_keywords":["...","..."],"content_type":"article","priority":"P0"},...],
  "content_gaps": [{"keyword":"...","volume":800,"competitors":"none","gap_type":"missing_topic","priority":"P0","suggested_action":"..."},...]
}`, lang)
}

// buildSEOResearchUserPrompt 构建用户提示
func buildSEOResearchUserPrompt(seedKeyword, lang string) string {
	return fmt.Sprintf(`Perform comprehensive SEO keyword research for the topic: "%s"

Context:
- This is for an AI creative workspace platform (harse.tv) that offers:
  * Node-based canvas video creation (like ComfyUI but simpler)
  * AI video prompt library (Sora, Kling, Veo, Runway, etc.)
  * Multi-language support (12 languages)
  * Schema markup and GEO optimization

- Target audience: video creators, marketers, AI enthusiasts, non-professional video editors
- Primary search engine: Google (global market)
- Language: %s

Please analyze:
1. What people actually search for when looking for this topic
2. What questions they have (informational intent)
3. What comparisons they make (commercial intent)
4. What specific tools/models they search for (transactional intent)
5. What content gaps exist that harse.tv can fill

Be specific and realistic with search volumes. Focus on keywords that a new/young domain can realistically rank for within 3-6 months.`, seedKeyword, lang)
}

// fallbackParseResearchResult 当 JSON 解析失败时的降级处理
func fallbackParseResearchResult(content, seedKeyword, lang string) (*model.SEOKeywordResearchResult, error) {
	// 尝试提取关键词列表
	result := &model.SEOKeywordResearchResult{
		SeedKeyword: seedKeyword,
		Language:    lang,
	}

	// 简单提取：按行分割，找包含 "keyword" 的行
	lines := strings.Split(content, "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检测当前段落
		lower := strings.ToLower(line)
		if strings.Contains(lower, "seed") || strings.Contains(lower, "核心") {
			currentSection = "seed"
			continue
		}
		if strings.Contains(lower, "extended") || strings.Contains(lower, "扩展") {
			currentSection = "extended"
			continue
		}
		if strings.Contains(lower, "long") || strings.Contains(lower, "长尾") {
			currentSection = "long"
			continue
		}
		if strings.Contains(lower, "high roi") || strings.Contains(lower, "高 roi") {
			currentSection = "high_roi"
			continue
		}

		// 提取关键词（简单模式：去除标点和数字前缀）
		keyword := extractKeywordFromLine(line)
		if keyword == "" {
			continue
		}

		item := model.KeywordItem{
			Keyword:      keyword,
			SearchVolume: estimateVolume(keyword),
			Intent:       "informational",
			Difficulty:   "medium",
			BusinessValue: 5,
			ROIScore:     50,
		}

		switch currentSection {
		case "seed":
			result.SeedKeywords = append(result.SeedKeywords, item)
		case "extended":
			result.ExtendedKeywords = append(result.ExtendedKeywords, item)
		case "long":
			result.LongTailKeywords = append(result.LongTailKeywords, item)
		case "high_roi":
			item.ROIScore = 75
			item.BusinessValue = 8
			result.HighROIKeywords = append(result.HighROIKeywords, item)
		}
	}

	// 确保至少有一些数据
	if len(result.SeedKeywords) == 0 && len(result.ExtendedKeywords) == 0 {
		// 完全无法解析，使用种子词生成基础数据
		result.SeedKeywords = []model.KeywordItem{
			{Keyword: seedKeyword, SearchVolume: 1000, Intent: "informational", Difficulty: "medium", BusinessValue: 8, ROIScore: 70},
		}
	}

	result.TotalCount = len(result.SeedKeywords) + len(result.ExtendedKeywords) + len(result.LongTailKeywords)
	result.HighROICount = len(result.HighROIKeywords)

	return result, nil
}

// extractKeywordFromLine 从行中提取关键词
func extractKeywordFromLine(line string) string {
	// 移除常见前缀符号
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimPrefix(line, "•")
	line = strings.TrimPrefix(line, "1.")
	line = strings.TrimPrefix(line, "2.")
	line = strings.TrimPrefix(line, "3.")
	line = strings.TrimSpace(line)

	// 移除 JSON 语法
	if idx := strings.Index(line, `"`); idx != -1 {
		// 尝试提取引号内的内容
		parts := strings.Split(line, `"`)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) > 3 && !strings.Contains(part, ":") && !strings.Contains(part, "{") {
				return part
			}
		}
	}

	// 移除冒号后的内容
	if idx := strings.Index(line, ":"); idx != -1 {
		line = line[:idx]
	}

	line = strings.TrimSpace(line)
	if len(line) < 3 || len(line) > 100 {
		return ""
	}

	return line
}

// estimateVolume 根据关键词估算搜索量
func estimateVolume(keyword string) int {
	lower := strings.ToLower(keyword)
	// 大词
	if strings.Contains(lower, "best") || strings.Contains(lower, "top") || strings.Contains(lower, "guide") {
		return 5000 + len(keyword)*100
	}
	// 品牌/模型词
	if strings.Contains(lower, "sora") || strings.Contains(lower, "kling") || strings.Contains(lower, "veo") || strings.Contains(lower, "runway") {
		return 3000 + len(keyword)*50
	}
	// 长尾词
	if strings.Contains(lower, "how to") || strings.Contains(lower, "tutorial") || strings.Contains(lower, "example") {
		return 800 + len(keyword)*20
	}
	// 默认
	return 1200 + len(keyword)*30
}

// generateDefaultTopicClusters 当 AI 未返回主题簇时，基于关键词自动生成
func generateDefaultTopicClusters(seed, extended []model.KeywordItem) []model.TopicCluster {
	all := append(seed, extended...)
	if len(all) == 0 {
		return nil
	}

	// 简单分组策略：按关键词中的常见主题词分组
	groups := map[string][]model.KeywordItem{}
	for _, kw := range all {
		k := strings.ToLower(kw.Keyword)
		matched := false
		for _, tag := range []string{"prompt", "workflow", "canvas", "sora", "kling", "template", "tool", "guide", "tutorial"} {
			if strings.Contains(k, tag) {
				groups[tag] = append(groups[tag], kw)
				matched = true
				break
			}
		}
		if !matched {
			groups["general"] = append(groups["general"], kw)
		}
	}

	var clusters []model.TopicCluster
	idx := 0
	for tag, items := range groups {
		if len(items) == 0 {
			continue
		}
		pillar := items[0]
		var clusterKws []string
		for i := 1; i < len(items) && i < 6; i++ {
			clusterKws = append(clusterKws, items[i].Keyword)
		}
		priority := "P1"
		if idx == 0 {
			priority = "P0"
		}
		clusters = append(clusters, model.TopicCluster{
			Name:           strings.Title(tag) + " Cluster",
			PillarKeyword:  pillar.Keyword,
			PillarVolume:   pillar.SearchVolume,
			ClusterKeywords: clusterKws,
			ContentType:    "article",
			Priority:       priority,
		})
		idx++
		if idx >= 5 {
			break
		}
	}

	if len(clusters) == 0 {
		clusters = append(clusters, model.TopicCluster{
			Name:           "General",
			PillarKeyword:  all[0].Keyword,
			PillarVolume:   all[0].SearchVolume,
			ClusterKeywords: []string{},
			ContentType:    "article",
			Priority:       "P0",
		})
	}

	return clusters
}
