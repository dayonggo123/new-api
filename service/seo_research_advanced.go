package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// AdvancedResearchRequest 高级研究请求（站内机会 / 竞品反查 / 社群挖掘）
type AdvancedResearchRequest struct {
	SeedKeyword string `json:"seed_keyword" binding:"required"`
	Language    string `json:"language"`
}

// ResearchSiteOpportunity 站内机会词研究
// 基于站点已有内容，找出种子词相关但覆盖不足的机会关键词
func ResearchSiteOpportunity(req *AdvancedResearchRequest) (*model.SEOKeywordResearchResult, error) {
	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	seed := strings.TrimSpace(req.SeedKeyword)
	if seed == "" {
		return nil, fmt.Errorf("seed keyword is required")
	}

	seedLower := strings.ToLower(seed)
	likePattern := "%" + seedLower + "%"

	// 查询站内已覆盖的文章（标题、SEO 关键词、内容包含种子词）
	var articles []*model.Article
	model.DB.Select("id, title, slug, seo_keywords, tags, status").
		Where("status = ? AND (LOWER(title) LIKE ? OR LOWER(seo_keywords) LIKE ? OR LOWER(content) LIKE ?)", 1, likePattern, likePattern, likePattern).
		Limit(50).Find(&articles)

	// 查询站内已覆盖的 Prompt
	var prompts []*model.Prompt
	model.DB.Select("id, title, slug, seo_keywords, tags, description, status").
		Where("status = ? AND (LOWER(title) LIKE ? OR LOWER(seo_keywords) LIKE ? OR LOWER(description) LIKE ? OR LOWER(content) LIKE ?)", 1, likePattern, likePattern, likePattern, likePattern).
		Limit(50).Find(&prompts)

	// 提取已覆盖关键词
	coveredSet := make(map[string]bool)
	var coveredKeywords []string
	for _, a := range articles {
		coveredSet[strings.ToLower(a.Title)] = true
		for _, kw := range strings.Split(a.SeoKeywords, ",") {
			k := strings.TrimSpace(strings.ToLower(kw))
			if k != "" && !coveredSet[k] {
				coveredSet[k] = true
				coveredKeywords = append(coveredKeywords, k)
			}
		}
	}
	for _, p := range prompts {
		coveredSet[strings.ToLower(p.Title)] = true
		for _, kw := range strings.Split(p.SeoKeywords, ",") {
			k := strings.TrimSpace(strings.ToLower(kw))
			if k != "" && !coveredSet[k] {
				coveredSet[k] = true
				coveredKeywords = append(coveredKeywords, k)
			}
		}
	}

	// 构建机会词：基于站内已有关键词做长尾扩展，再排除已覆盖
	opportunityKeywords := generateSiteOpportunityKeywords(seed, lang, coveredKeywords)
	var longTail []model.KeywordItem
	for _, kw := range opportunityKeywords {
		lower := strings.ToLower(kw)
		if coveredSet[lower] {
			continue
		}
		longTail = append(longTail, model.KeywordItem{
			Keyword:       kw,
			SearchVolume:  estimateSERPVolume(kw),
			Intent:        inferIntent(kw),
			Difficulty:    "low",
			BusinessValue: 8,
			ROIScore:      78,
			SuggestedURL:  suggestURL(kw),
		})
	}

	// 高 ROI 词：取前 10
	highROI := selectHighROI(longTail)

	// 内容缺口：站内未覆盖的 FAQ / 对比 / 教程型关键词
	contentGaps := buildContentGapsFromKeywords(longTail, "site_opportunity", "补充站内覆盖，抢占长尾流量")

	// 主题簇
	topicClusters := []model.TopicCluster{
		{
			Name:            seed,
			PillarKeyword:   seed,
			PillarVolume:    estimateSERPVolume(seed),
			ClusterKeywords: takeFirstN(opportunityKeywords, 8),
			ContentType:     "article",
			Priority:        "P0",
		},
	}

	result := &model.SEOKeywordResearchResult{
		SeedKeyword:      seed,
		Language:         lang,
		SeedKeywords:     []model.KeywordItem{makeKeywordItem(seed, lang)},
		ExtendedKeywords: makeSiteCoveredItems(coveredKeywords, lang),
		LongTailKeywords: longTail,
		HighROIKeywords:  highROI,
		TopicClusters:    topicClusters,
		ContentGaps:      contentGaps,
	}

	result.TotalCount = len(result.SeedKeywords) + len(result.ExtendedKeywords) + len(result.LongTailKeywords)
	result.HighROICount = len(result.HighROIKeywords)
	result.ClusterCount = len(result.TopicClusters)
	return result, nil
}

// ResearchCompetitor 竞品反查研究
// 模拟/推导竞品在目标关键词上的覆盖情况，输出可抢流量的关键词缺口
func ResearchCompetitor(req *AdvancedResearchRequest) (*model.SEOKeywordResearchResult, error) {
	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	seed := strings.TrimSpace(req.SeedKeyword)
	if seed == "" {
		return nil, fmt.Errorf("seed keyword is required")
	}

	// 生成竞品相关关键词（best / vs / alternative / review 等）
	competitorKeywords := generateCompetitorKeywords(seed, lang)

	var longTail []model.KeywordItem
	for _, kw := range competitorKeywords {
		longTail = append(longTail, model.KeywordItem{
			Keyword:       kw,
			SearchVolume:  estimateSERPVolume(kw),
			Intent:        inferIntent(kw),
			Difficulty:    "medium",
			BusinessValue: 9,
			ROIScore:      82,
			SuggestedURL:  suggestURL(kw),
		})
	}

	// 内容缺口：竞品教育/对比类关键词
	contentGaps := buildContentGapsFromKeywords(longTail, "competitor_gap", "制作对比/评测内容，截流竞品品牌词")

	highROI := selectHighROI(longTail)

	topicClusters := []model.TopicCluster{
		{
			Name:            seed + " 竞品分析",
			PillarKeyword:   seed,
			PillarVolume:    estimateSERPVolume(seed),
			ClusterKeywords: takeFirstN(competitorKeywords, 8),
			ContentType:     "article",
			Priority:        "P0",
		},
	}

	result := &model.SEOKeywordResearchResult{
		SeedKeyword:      seed,
		Language:         lang,
		SeedKeywords:     []model.KeywordItem{makeKeywordItem(seed, lang)},
		ExtendedKeywords: longTail[:minInt(len(longTail), 10)],
		LongTailKeywords: longTail,
		HighROIKeywords:  highROI,
		TopicClusters:    topicClusters,
		ContentGaps:      contentGaps,
	}

	result.TotalCount = len(result.SeedKeywords) + len(result.ExtendedKeywords) + len(result.LongTailKeywords)
	result.HighROICount = len(result.HighROIKeywords)
	result.ClusterCount = len(result.TopicClusters)
	return result, nil
}

// ResearchCommunity 社群挖掘研究
// 基于 Reddit / 论坛 / QA 等社群讨论风格，挖掘真实用户问题型关键词
func ResearchCommunity(req *AdvancedResearchRequest) (*model.SEOKeywordResearchResult, error) {
	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	seed := strings.TrimSpace(req.SeedKeyword)
	if seed == "" {
		return nil, fmt.Errorf("seed keyword is required")
	}

	communityKeywords := generateCommunityKeywords(seed, lang)

	var longTail []model.KeywordItem
	for _, kw := range communityKeywords {
		longTail = append(longTail, model.KeywordItem{
			Keyword:       kw,
			SearchVolume:  estimateSERPVolume(kw),
			Intent:        "informational",
			Difficulty:    "low",
			BusinessValue: 6,
			ROIScore:      76,
			SuggestedURL:  suggestURL(kw),
		})
	}

	// FAQ 风格的内容缺口
	contentGaps := buildContentGapsFromKeywords(longTail, "community_question", "围绕真实用户问题创建问答/教程内容")

	highROI := selectHighROI(longTail)

	topicClusters := []model.TopicCluster{
		{
			Name:            seed + " 社群问题",
			PillarKeyword:   seed,
			PillarVolume:    estimateSERPVolume(seed),
			ClusterKeywords: takeFirstN(communityKeywords, 8),
			ContentType:     "article",
			Priority:        "P1",
		},
	}

	result := &model.SEOKeywordResearchResult{
		SeedKeyword:      seed,
		Language:         lang,
		SeedKeywords:     []model.KeywordItem{makeKeywordItem(seed, lang)},
		ExtendedKeywords: longTail[:minInt(len(longTail), 10)],
		LongTailKeywords: longTail,
		HighROIKeywords:  highROI,
		TopicClusters:    topicClusters,
		ContentGaps:      contentGaps,
	}

	result.TotalCount = len(result.SeedKeywords) + len(result.ExtendedKeywords) + len(result.LongTailKeywords)
	result.HighROICount = len(result.HighROIKeywords)
	result.ClusterCount = len(result.TopicClusters)
	return result, nil
}

// generateSiteOpportunityKeywords 基于站内已覆盖词生成机会长尾词
func generateSiteOpportunityKeywords(seed, lang string, coveredKeywords []string) []string {
	seen := make(map[string]bool)
	var result []string

	// 通用站内机会模板
	templates := map[string][]string{
		"en": {
			"{{seed}} examples",
			"{{seed}} tutorial for beginners",
			"{{seed}} tips and tricks",
			"{{seed}} best practices",
			"{{seed}} vs other tools",
			"{{seed}} common mistakes",
			"{{seed}} FAQ",
			"{{seed}} use cases",
			"how to get started with {{seed}}",
			"{{seed}} guide 2026",
		},
		"zh": {
			"{{seed}}入门教程",
			"{{seed}}使用技巧",
			"{{seed}}最佳实践",
			"{{seed}}常见问题",
			"{{seed}}案例分享",
			"{{seed}}对比评测",
			"{{seed}}新手误区",
			"{{seed}}实战指南",
			"如何用{{seed}}",
			"{{seed}}2026最新",
		},
		"ja": {
			"{{seed}} 初心者",
			"{{seed}} 使い方",
			"{{seed}} おすすめ",
			"{{seed}} よくある質問",
			"{{seed}} 活用例",
		},
		"ko": {
			"{{seed}} 입문",
			"{{seed}} 사용법",
			"{{seed}} 팁",
			"{{seed}} 자주 묻는 질문",
		},
		"es": {
			"{{seed}} ejemplos",
			"{{seed}} tutorial principiantes",
			"{{seed}} mejores prácticas",
			"{{seed}} preguntas frecuentes",
		},
		"de": {
			"{{seed}} beispiele",
			"{{seed}} anleitung anfänger",
			"{{seed}} best practices",
			"{{seed}} FAQ",
		},
		"fr": {
			"{{seed}} exemples",
			"{{seed}} tutoriel débutants",
			"{{seed}} meilleures pratiques",
			"{{seed}} FAQ",
		},
	}

	chosen, ok := templates[lang]
	if !ok {
		chosen = templates["en"]
	}

	for _, tmpl := range chosen {
		kw := strings.ReplaceAll(tmpl, "{{seed}}", seed)
		lower := strings.ToLower(kw)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, kw)
		}
	}

	// 基于已覆盖关键词二次扩展
	for _, covered := range coveredKeywords {
		if len(covered) < 3 {
			continue
		}
		variations := []string{
			covered + " " + seed,
			seed + " " + covered,
		}
		for _, kw := range variations {
			lower := strings.ToLower(kw)
			if !seen[lower] && len(kw) > len(seed)+3 {
				seen[lower] = true
				result = append(result, kw)
			}
		}
	}

	return result
}

// generateCompetitorKeywords 生成竞品反查关键词
func generateCompetitorKeywords(seed, lang string) []string {
	seen := make(map[string]bool)
	var result []string

	templates := map[string][]string{
		"en": {
			"best {{seed}}",
			"{{seed}} review",
			"{{seed}} vs",
			"{{seed}} alternatives",
			"{{seed}} comparison",
			"top {{seed}} tools",
			"{{seed}} pricing",
			"{{seed}} free alternative",
			"{{seed}} for beginners",
			"{{seed}} tutorial",
			"{{seed}} pros and cons",
			"why use {{seed}}",
		},
		"zh": {
			"最好的{{seed}}",
			"{{seed}}评测",
			"{{seed}}对比",
			"{{seed}}替代方案",
			"{{seed}}哪个好",
			"{{seed}}价格",
			"免费{{seed}}工具",
			"{{seed}}新手推荐",
			"{{seed}}优缺点",
			"为什么要用{{seed}}",
		},
		"ja": {
			"{{seed}} おすすめ",
			"{{seed}} 比較",
			"{{seed}} 無料",
			"{{seed}} レビュー",
			"{{seed}} 料金",
		},
		"ko": {
			"{{seed}} 추천",
			"{{seed}} 비교",
			"{{seed}} 묣",
			"{{seed}} 리뷰",
			"{{seed}} 가격",
		},
		"es": {
			"mejor {{seed}}",
			"{{seed}} reseña",
			"{{seed}} vs",
			"{{seed}} alternativas",
			"{{seed}} precio",
		},
		"de": {
			"beste {{seed}}",
			"{{seed}} bewertung",
			"{{seed}} vs",
			"{{seed}} alternative",
			"{{seed}} preis",
		},
		"fr": {
			"meilleur {{seed}}",
			"{{seed}} avis",
			"{{seed}} vs",
			"{{seed}} alternative",
			"{{seed}} prix",
		},
	}

	chosen, ok := templates[lang]
	if !ok {
		chosen = templates["en"]
	}

	for _, tmpl := range chosen {
		kw := strings.ReplaceAll(tmpl, "{{seed}}", seed)
		lower := strings.ToLower(kw)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, kw)
		}
	}

	return result
}

// generateCommunityKeywords 生成社群/论坛/QA 风格关键词
func generateCommunityKeywords(seed, lang string) []string {
	seen := make(map[string]bool)
	var result []string

	templates := map[string][]string{
		"en": {
			"how to use {{seed}}",
			"what is {{seed}}",
			"{{seed}} not working",
			"{{seed}} worth it",
			"reddit {{seed}}",
			"{{seed}} community",
			"{{seed}} help",
			"beginner question {{seed}}",
			"{{seed}} tips from experts",
			"real world {{seed}} examples",
			"{{seed}} use case",
			"how do you {{seed}}",
		},
		"zh": {
			"如何使用{{seed}}",
			"{{seed}}是什么",
			"{{seed}}用不了",
			"{{seed}}值得吗",
			"知乎 {{seed}}",
			"{{seed}}讨论",
			"{{seed}}求助",
			"{{seed}}新手问题",
			"{{seed}}专家建议",
			"{{seed}}真实案例",
		},
		"ja": {
			"{{seed}} 使い方",
			"{{seed}} とは",
			"{{seed}} エラー",
			"{{seed}} おすすめ",
			"{{seed}} 口コミ",
		},
		"ko": {
			"{{seed}} 사용법",
			"{{seed}}란",
			"{{seed}} 오류",
			"{{seed}} 추천",
			"{{seed}} 후기",
		},
		"es": {
			"cómo usar {{seed}}",
			"qué es {{seed}}",
			"{{seed}} no funciona",
			"{{seed}} opiniones",
		},
		"de": {
			"wie verwende ich {{seed}}",
			"was ist {{seed}}",
			"{{seed}} funktioniert nicht",
			"{{seed}} erfahrungen",
		},
		"fr": {
			"comment utiliser {{seed}}",
			"qu'est-ce que {{seed}}",
			"{{seed}} ne fonctionne pas",
			"{{seed}} avis",
		},
	}

	chosen, ok := templates[lang]
	if !ok {
		chosen = templates["en"]
	}

	for _, tmpl := range chosen {
		kw := strings.ReplaceAll(tmpl, "{{seed}}", seed)
		lower := strings.ToLower(kw)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, kw)
		}
	}

	return result
}

// buildContentGapsFromKeywords 从关键词列表生成内容缺口
func buildContentGapsFromKeywords(keywords []model.KeywordItem, gapType, action string) []model.ContentGap {
	var gaps []model.ContentGap
	for _, kw := range keywords {
		gaps = append(gaps, model.ContentGap{
			Keyword:         kw.Keyword,
			Volume:          kw.SearchVolume,
			Competitors:     "-",
			GapType:         gapType,
			Priority:        "P1",
			SuggestedAction: action,
		})
	}
	if len(gaps) > 12 {
		gaps = gaps[:12]
	}
	return gaps
}

// makeSiteCoveredItems 把已覆盖关键词包装为 KeywordItem（用于展示站内已有资产）
func makeSiteCoveredItems(coveredKeywords []string, lang string) []model.KeywordItem {
	var items []model.KeywordItem
	limit := 10
	if len(coveredKeywords) < limit {
		limit = len(coveredKeywords)
	}
	for i := 0; i < limit; i++ {
		kw := coveredKeywords[i]
		items = append(items, model.KeywordItem{
			Keyword:       kw,
			SearchVolume:  estimateSERPVolume(kw),
			Intent:        inferIntent(kw),
			Difficulty:    "low",
			BusinessValue: 7,
			ROIScore:      70,
			SuggestedURL:  suggestURL(kw),
		})
	}
	return items
}

// minInt 取较小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
