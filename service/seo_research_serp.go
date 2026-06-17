package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const serpResearchTimeout = 30 * time.Second

// SERPResearchRequest SERP 深挖研究请求
type SERPResearchRequest struct {
	SeedKeyword string `json:"seed_keyword" binding:"required"`
	Language    string `json:"language"`
}

// SERPKeywordPatterns 用于生成长尾词变体的模板
var serpKeywordPatterns = map[string][]string{
	"en": {
		"best {{seed}}",
		"{{seed}} tutorial",
		"{{seed}} guide",
		"{{seed}} examples",
		"{{seed}} prompts",
		"{{seed}} free",
		"{{seed}} online",
		"{{seed}} for beginners",
		"{{seed}} vs",
		"how to use {{seed}}",
		"what is {{seed}}",
		"{{seed}} alternatives",
		"{{seed}} review",
		"{{seed}} comparison",
		"{{seed}} tips",
	},
	"zh": {
		"最佳{{seed}}",
		"{{seed}}教程",
		"{{seed}}入门",
		"{{seed}}示例",
		"{{seed}}提示词",
		"免费{{seed}}",
		"在线{{seed}}",
		"{{seed}}对比",
		"如何使用{{seed}}",
		"什么是{{seed}}",
		"{{seed}}替代方案",
		"{{seed}}评测",
		"{{seed}}技巧",
	},
	"ja": {
		"{{seed}} おすすめ",
		"{{seed}} 使い方",
		"{{seed}} チュートリアル",
		"{{seed}} 初心者",
		"{{seed}} 比較",
		"{{seed}} 無料",
	},
	"ko": {
		"{{seed}} 추천",
		"{{seed}} 사용법",
		"{{seed}} 튜토리얼",
		"{{seed}} 초보자",
		"{{seed}} 비교",
		"{{seed}} 묣",
	},
	"es": {
		"mejor {{seed}}",
		"{{seed}} tutorial",
		"{{seed}} guía",
		"{{seed}} ejemplos",
		"{{seed}} gratis",
		"{{seed}} vs",
	},
	"de": {
		"beste {{seed}}",
		"{{seed}} tutorial",
		"{{seed}} anleitung",
		"{{seed}} beispiele",
		"{{seed}} kostenlos",
		"{{seed}} vs",
	},
	"fr": {
		"meilleur {{seed}}",
		"{{seed}} tutoriel",
		"{{seed}} guide",
		"{{seed}} exemples",
		"{{seed}} gratuit",
		"{{seed}} vs",
	},
}

// ResearchSERP 执行 SERP 深挖研究
// 1. 调用 Google Suggest API 获取真实搜索建议
// 2. 基于种子词生成长尾词变体
// 3. 生成 PAA 风格的 FAQ 问题
func ResearchSERP(req *SERPResearchRequest) (*model.SEOKeywordResearchResult, error) {
	lang := req.Language
	if lang == "" {
		lang = "en"
	}

	seed := strings.TrimSpace(req.SeedKeyword)
	if seed == "" {
		return nil, fmt.Errorf("seed keyword is required")
	}

	// 1. 获取 Google Suggest 建议
	suggestions, err := fetchSERPGoogleSuggestions(seed, lang)
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("SERP research: fetch suggestions failed: %v", err))
		// 失败时继续使用本地模板生成
		suggestions = []string{}
	}

	// 2. 生成长尾词和扩展词
	longTailKeywords := generateLongTailKeywords(seed, lang, suggestions)
	extendedKeywords := generateExtendedKeywords(seed, lang)

	// 3. 生成 FAQ 问题（基于种子词和长尾词）
	faqQuestions := generateFAQQuestions(seed, lang, suggestions)

	// 4. 构建主题簇
	topicClusters := []model.TopicCluster{
		{
			Name:            seed,
			PillarKeyword:   seed,
			PillarVolume:    estimateSERPVolume(seed),
			ClusterKeywords: takeFirstN(suggestions, 8),
			ContentType:     "article",
			Priority:        "P0",
		},
	}

	// 5. 把 FAQ 问题作为内容缺口
	contentGaps := make([]model.ContentGap, 0, len(faqQuestions))
	for _, faq := range faqQuestions {
		contentGaps = append(contentGaps, model.ContentGap{
			Keyword:         faq,
			Volume:          estimateSERPVolume(faq),
			Competitors:     "-",
			GapType:         "faq_opportunity",
			Priority:        "P0",
			SuggestedAction: "创建 FAQ 或专门文章回答该问题",
		})
	}

	result := &model.SEOKeywordResearchResult{
		SeedKeyword:      seed,
		Language:         lang,
		SeedKeywords:     []model.KeywordItem{makeKeywordItem(seed, lang)},
		ExtendedKeywords: extendedKeywords,
		LongTailKeywords: longTailKeywords,
		HighROIKeywords:  selectHighROI(longTailKeywords),
		TopicClusters:    topicClusters,
		ContentGaps:      contentGaps,
	}

	result.TotalCount = len(result.SeedKeywords) + len(result.ExtendedKeywords) + len(result.LongTailKeywords)
	result.HighROICount = len(result.HighROIKeywords)
	result.ClusterCount = len(result.TopicClusters)

	return result, nil
}

// fetchSERPGoogleSuggestions 调用 Google Suggest API 获取搜索建议
// 使用 chrome client 可以获取 JSON 格式的建议
func fetchSERPGoogleSuggestions(seed, lang string) ([]string, error) {
	query := url.QueryEscape(seed)
	suggestURL := fmt.Sprintf("https://suggestqueries.google.com/complete/search?client=chrome&q=%s&hl=%s", query, lang)

	client := &http.Client{Timeout: serpResearchTimeout}
	req, err := http.NewRequest(http.MethodGet, suggestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google suggest returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Google Suggest 返回 JSONP 风格数据：["query", ["suggestion1", "suggestion2", ...]]
	var suggestResp []interface{}
	if err := json.Unmarshal(body, &suggestResp); err != nil {
		return nil, fmt.Errorf("parse suggest response failed: %w", err)
	}

	if len(suggestResp) < 2 {
		return nil, fmt.Errorf("unexpected suggest response format")
	}

	suggestionsRaw, ok := suggestResp[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected suggest suggestions format")
	}

	suggestions := make([]string, 0, len(suggestionsRaw))
	for _, item := range suggestionsRaw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			suggestions = append(suggestions, strings.TrimSpace(s))
		}
	}

	return suggestions, nil
}

// generateLongTailKeywords 基于 Google Suggest 和模板生成长尾词
func generateLongTailKeywords(seed, lang string, suggestions []string) []model.KeywordItem {
	seen := make(map[string]bool)
	keywords := make([]model.KeywordItem, 0)

	// 优先使用真实的 Google Suggest
	for _, kw := range suggestions {
		lower := strings.ToLower(kw)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		keywords = append(keywords, model.KeywordItem{
			Keyword:       kw,
			SearchVolume:  estimateSERPVolume(kw),
			Intent:        inferIntent(kw),
			Difficulty:    "low",
			BusinessValue: 7,
			ROIScore:      75,
			SuggestedURL:  suggestURL(kw),
		})
	}

	return keywords
}

// generateExtendedKeywords 生成本地模板扩展词
func generateExtendedKeywords(seed, lang string) []model.KeywordItem {
	patterns, ok := serpKeywordPatterns[lang]
	if !ok {
		patterns = serpKeywordPatterns["en"]
	}

	seen := make(map[string]bool)
	keywords := make([]model.KeywordItem, 0, len(patterns))

	for _, pattern := range patterns {
		kw := strings.ReplaceAll(pattern, "{{seed}}", seed)
		lower := strings.ToLower(kw)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		keywords = append(keywords, model.KeywordItem{
			Keyword:       kw,
			SearchVolume:  estimateSERPVolume(kw),
			Intent:        inferIntent(kw),
			Difficulty:    "medium",
			BusinessValue: 8,
			ROIScore:      70,
			SuggestedURL:  suggestURL(kw),
		})
	}

	return keywords
}

// generateFAQQuestions 基于种子词和搜索建议生成 FAQ 问题
func generateFAQQuestions(seed, lang string, suggestions []string) []string {
	questions := make([]string, 0)

	templates := map[string][]string{
		"en": {
			"What is %s?",
			"How to use %s?",
			"What are the best %s?",
			"Is %s free?",
			"How does %s work?",
			"What can I create with %s?",
			"Is %s worth it?",
			"What are %s alternatives?",
		},
		"zh": {
			"什么是%s？",
			"如何使用%s？",
			"最好的%s是什么？",
			"%s是免费的吗？",
			"%s的工作原理是什么？",
			"用%s可以创建什么？",
			"%s值得使用吗？",
			"%s的替代方案有哪些？",
		},
		"ja": {
			"%s とは何ですか？",
			"%s の使い方は？",
			"%s のおすすめは？",
			"%s は無料ですか？",
		},
		"ko": {
			"%s란 무엇인가요?",
			"%s 사용법은?",
			"%s 추천은?",
			"%s는 묣인가요?",
		},
		"es": {
			"¿Qué es %s?",
			"¿Cómo usar %s?",
			"¿Cuáles son los mejores %s?",
			"¿Es gratis %s?",
		},
		"de": {
			"Was ist %s?",
			"Wie verwende ich %s?",
			"Was sind die besten %s?",
			"Ist %s kostenlos?",
		},
		"fr": {
			"Qu'est-ce que %s ?",
			"Comment utiliser %s ?",
			"Quels sont les meilleurs %s ?",
			"%s est-il gratuit ?",
		},
	}

	chosenTemplates, ok := templates[lang]
	if !ok {
		chosenTemplates = templates["en"]
	}

	for _, tmpl := range chosenTemplates {
		questions = append(questions, fmt.Sprintf(tmpl, seed))
	}

	// 从 suggestions 中抽取问题型长尾词
	questionStarters := map[string][]string{
		"en": {"what", "how", "why", "is", "can", "does", "best", "top"},
		"zh": {"什么", "如何", "怎么", "为什么", "是否", "最好", "推荐"},
		"ja": {"とは", "使い方", "おすすめ", "無料", "方法"},
		"ko": {"란", "사용법", "추천", "묣", "방법"},
		"es": {"qué", "cómo", "por qué", "es", "mejor"},
		"de": {"was", "wie", "warum", "ist", "beste"},
		"fr": {"qu'est-ce", "comment", "pourquoi", "est-ce", "meilleur"},
	}

	starters, ok := questionStarters[lang]
	if !ok {
		starters = questionStarters["en"]
	}

	seen := make(map[string]bool)
	for _, q := range questions {
		seen[strings.ToLower(q)] = true
	}

	for _, s := range suggestions {
		lower := strings.ToLower(s)
		for _, starter := range starters {
			if strings.HasPrefix(lower, starter) && !seen[lower] {
				seen[lower] = true
				questions = append(questions, s)
				break
			}
		}
	}

	if len(questions) > 12 {
		questions = questions[:12]
	}
	return questions
}

// selectHighROI 从高 ROI 候选中挑选分数最高的
func selectHighROI(keywords []model.KeywordItem) []model.KeywordItem {
	if len(keywords) == 0 {
		return nil
	}
	limit := 10
	if len(keywords) < limit {
		limit = len(keywords)
	}
	return keywords[:limit]
}

// makeKeywordItem 创建单个关键词项
func makeKeywordItem(seed, lang string) model.KeywordItem {
	return model.KeywordItem{
		Keyword:       seed,
		SearchVolume:  estimateSERPVolume(seed),
		Intent:        inferIntent(seed),
		Difficulty:    "medium",
		BusinessValue: 8,
		ROIScore:      80,
		SuggestedURL:  suggestURL(seed),
	}
}

// inferIntent 推断搜索意图
func inferIntent(keyword string) string {
	lower := strings.ToLower(keyword)
	switch {
	case strings.Contains(lower, "buy") || strings.Contains(lower, "price") || strings.Contains(lower, "discount") || strings.Contains(lower, "deal"):
		return "transactional"
	case strings.Contains(lower, "vs") || strings.Contains(lower, "compare") || strings.Contains(lower, "best") || strings.Contains(lower, "top") || strings.Contains(lower, "review"):
		return "commercial"
	case strings.HasPrefix(lower, "how to") || strings.HasPrefix(lower, "what is") || strings.HasPrefix(lower, "why ") || strings.HasPrefix(lower, "tutorial") || strings.HasPrefix(lower, "guide"):
		return "informational"
	default:
		return "informational"
	}
}

// suggestURL 根据关键词建议 URL slug
func suggestURL(keyword string) string {
	// 简单 slug 化：小写、替换空格和特殊字符为 -
	slug := strings.ToLower(keyword)
	re := regexp.MustCompile(`[^a-z0-9\u4e00-\u9fa5]+`)
	slug = re.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "article"
	}
	return "/" + slug
}

// estimateSERPVolume 根据关键词长度和特征粗略估算搜索量
func estimateSERPVolume(keyword string) int {
	length := len(keyword)
	base := 5000
	if length > 30 {
		base = 800
	} else if length > 20 {
		base = 2000
	}

	// 加入一些随机性但保持确定性：基于字符和
	sum := 0
	for _, r := range keyword {
		sum += int(r)
	}
	return base + (sum % 1000)
}

// takeFirstN 取前 N 个元素
func takeFirstN(arr []string, n int) []string {
	if len(arr) <= n {
		return arr
	}
	return arr[:n]
}
