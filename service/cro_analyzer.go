package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// CROAnalyzeRequest CRO 分析请求
type CROAnalyzeRequest struct {
	RecordID    int    `json:"record_id" binding:"required"`
	ContentType string `json:"content_type" binding:"required"` // article / prompt
	Language    string `json:"language"`
}

// CRODimension CRO 维度评分
type CRODimension struct {
	Name       string `json:"name"`
	Score      int    `json:"score"`
	Status     string `json:"status"` // pass / warning / fail
	Suggestion string `json:"suggestion"`
}

// CROCTA CTA 检测项
type CROCTA struct {
	Text     string `json:"text"`
	Context  string `json:"context"`
	Position string `json:"position"`
	Strength int    `json:"strength"` // 1-10
}

// CROAnalyzeResult CRO 分析结果
type CROAnalyzeResult struct {
	RecordID     int            `json:"record_id"`
	ContentType  string         `json:"content_type"`
	Title        string         `json:"title"`
	OverallScore int            `json:"overall_score"`
	Dimensions   []CRODimension `json:"dimensions"`
	CTAs         []CROCTA       `json:"ctas"`
	ABTests      []string       `json:"ab_tests"`
	Suggestions  []string       `json:"suggestions"`
}

// AnalyzeCRO 对指定内容进行转化率优化分析
func AnalyzeCRO(req *CROAnalyzeRequest) (*CROAnalyzeResult, error) {
	recordType := req.ContentType
	recordID := req.RecordID
	if recordType != "article" && recordType != "prompt" {
		return nil, fmt.Errorf("unsupported content type: %s", recordType)
	}

	var title, content string
	switch recordType {
	case "article":
		article, err := model.GetArticleById(recordID)
		if err != nil {
			return nil, err
		}
		title = article.Title
		content = article.Content
	case "prompt":
		prompt, err := model.GetPromptById(recordID)
		if err != nil {
			return nil, err
		}
		title = prompt.Title
		content = prompt.Content
	}

	plain := stripHTML(content)
	lower := strings.ToLower(plain)

	dimensions := []CRODimension{}

	// 1. 行动召唤（CTA）
	ctaScore, ctas, ctaSuggestions := analyzeCTA(plain, lower)
	dimensions = append(dimensions, CRODimension{
		Name:       "行动召唤 (CTA)",
		Score:      ctaScore,
		Status:     scoreStatus(ctaScore),
		Suggestion: ctaSuggestions,
	})

	// 2. 信任建设
	trustScore, trustSuggestion := analyzeTrustSignals(lower)
	dimensions = append(dimensions, CRODimension{
		Name:       "信任建设",
		Score:      trustScore,
		Status:     scoreStatus(trustScore),
		Suggestion: trustSuggestion,
	})

	// 3. 情感触发 / 说服力
	emotionScore, emotionSuggestion := analyzeEmotionTriggers(lower)
	dimensions = append(dimensions, CRODimension{
		Name:       "情感触发与说服",
		Score:      emotionScore,
		Status:     scoreStatus(emotionScore),
		Suggestion: emotionSuggestion,
	})

	// 4. 认知负荷
	cognitiveScore, cognitiveSuggestion := analyzeCognitiveLoad(plain)
	dimensions = append(dimensions, CRODimension{
		Name:       "认知负荷",
		Score:      cognitiveScore,
		Status:     scoreStatus(cognitiveScore),
		Suggestion: cognitiveSuggestion,
	})

	// 5. 异议处理
	objectionScore, objectionSuggestion := analyzeObjectionHandling(lower)
	dimensions = append(dimensions, CRODimension{
		Name:       "异议处理",
		Score:      objectionScore,
		Status:     scoreStatus(objectionScore),
		Suggestion: objectionSuggestion,
	})

	// 6. 决策旅程覆盖
	journeyScore, journeySuggestion := analyzeDecisionJourney(lower)
	dimensions = append(dimensions, CRODimension{
		Name:       "决策旅程覆盖",
		Score:      journeyScore,
		Status:     scoreStatus(journeyScore),
		Suggestion: journeySuggestion,
	})

	overallScore := calculateCROOverallScore(dimensions)

	suggestions := []string{}
	suggestions = append(suggestions, ctaSuggestions)
	suggestions = append(suggestions, trustSuggestion)
	suggestions = append(suggestions, emotionSuggestion)
	suggestions = append(suggestions, cognitiveSuggestion)
	suggestions = append(suggestions, objectionSuggestion)
	suggestions = append(suggestions, journeySuggestion)
	suggestions = filterEmptySuggestions(suggestions)

	abTests := generateABTests(dimensions, title)

	return &CROAnalyzeResult{
		RecordID:     recordID,
		ContentType:  recordType,
		Title:        title,
		OverallScore: overallScore,
		Dimensions:   dimensions,
		CTAs:         ctas,
		ABTests:      abTests,
		Suggestions:  suggestions,
	}, nil
}

func analyzeCTA(plain, lower string) (score int, ctas []CROCTA, suggestion string) {
	ctaKeywords := []string{
		"立即", "马上", "点击", "注册", "试用", "下载", "购买", "订阅", "获取", "开始", "申请", "查看", "了解更多",
		"get started", "sign up", "try now", "buy now", "download", "subscribe", "register", "learn more",
		"contact us", "get a demo", "book a call", "join now", "start free", "claim", "unlock",
	}

	found := []CROCTA{}
	for _, kw := range ctaKeywords {
		if strings.Contains(lower, kw) {
			found = append(found, CROCTA{
				Text:     kw,
				Context:  extractContext(plain, kw, 30),
				Position: estimatePosition(plain, kw),
				Strength: 7,
			})
		}
	}

	score = 40 + len(found)*15
	if score > 100 {
		score = 100
	}

	suggestion = "CTA 数量合适，建议测试不同文案"
	if len(found) == 0 {
		suggestion = "未检测到明确的行动召唤，建议在内容中加入 1-2 个清晰的 CTA（如“立即试用”“免费注册”）"
		score = 30
	} else if len(found) == 1 {
		suggestion = "仅检测到 1 个 CTA，建议在首屏和结尾各放置一个，提升转化机会"
	}

	return score, found, suggestion
}

func analyzeTrustSignals(lower string) (int, string) {
	trustMarkers := []string{
		"认证", "权威", "专家", "客户", "案例", "评价", "信任", "安全", "保障", "退款", "隐私", "官方",
		"certified", "expert", "trusted", "secure", "guarantee", "refund", "privacy", "verified",
		"case study", "testimonial", "review", "customer", "enterprise", "award", "partner",
	}
	count := 0
	for _, m := range trustMarkers {
		if strings.Contains(lower, m) {
			count++
		}
	}
	score := 40 + count*12
	if score > 100 {
		score = 100
	}
	if count == 0 {
		return 30, "缺少信任信号，建议加入客户案例、权威背书、安全认证或退款保障"
	}
	return score, "已包含信任信号，可进一步增加具体数据或权威推荐"
}

func analyzeEmotionTriggers(lower string) (int, string) {
	emotionMarkers := []string{
		"限时", "免费", "独家", "秘密", "轻松", "快速", "立即", "提升", "翻倍", "突破", "焦虑", "梦想",
		"free", "limited", "exclusive", "secret", "easy", "fast", "instant", "boost", "double", "breakthrough",
		"imagine", "discover", "proven", "guaranteed",
	}
	count := 0
	for _, m := range emotionMarkers {
		if strings.Contains(lower, m) {
			count++
		}
	}
	score := 45 + count*10
	if score > 100 {
		score = 100
	}
	if count == 0 {
		return 35, "情感触发较弱，建议加入“限时”“免费”“轻松达成”等利益点或稀缺性描述"
	}
	return score, "情感触发较好，可尝试加入更多故事化场景增强共鸣"
}

func analyzeCognitiveLoad(plain string) (int, string) {
	words := estimateWordCount(plain)
	if words == 0 {
		return 30, "内容为空，无法评估认知负荷"
	}

	score := 80
	if words > 2500 {
		score -= 20
	}
	if words > 4000 {
		score -= 15
	}

	sentences := strings.Split(plain, "。")
	if len(sentences) <= 1 {
		sentences = strings.Split(plain, ".")
	}
	avgLen := float64(words) / float64(len(sentences))
	if avgLen > 30 {
		score -= 15
	} else if avgLen > 25 {
		score -= 10
	}

	if !strings.Contains(plain, "##") && !strings.Contains(plain, "<h") {
		score -= 10
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	if score < 60 {
		return score, "认知负荷偏高，建议拆分长句、增加小标题和列表，降低阅读门槛"
	}
	return score, "认知负荷控制良好"
}

func analyzeObjectionHandling(lower string) (int, string) {
	objectionMarkers := []string{
		"faq", "常见问题", "疑问", "担心", "风险", "怎么办", "如何", "价格", "费用", "成本",
		"worry", "concern", "risk", "how to", "pricing", "cost", "what if", "guarantee", "refund",
	}
	count := 0
	for _, m := range objectionMarkers {
		if strings.Contains(lower, m) {
			count++
		}
	}
	score := 40 + count*12
	if score > 100 {
		score = 100
	}
	if count == 0 {
		return 30, "未明显处理用户异议，建议补充 FAQ、价格说明或风险保障"
	}
	return score, "已覆盖部分异议，可针对常见反对意见补充更直接的回答"
}

func analyzeDecisionJourney(lower string) (int, string) {
	// 简化：检测意识、考虑、决策三阶段信号
	hasAwareness := strings.Contains(lower, "问题") || strings.Contains(lower, "挑战") || strings.Contains(lower, "痛点") || strings.Contains(lower, "struggle") || strings.Contains(lower, "problem")
	hasConsideration := strings.Contains(lower, "对比") || strings.Contains(lower, "方案") || strings.Contains(lower, "选择") || strings.Contains(lower, "compare") || strings.Contains(lower, "solution") || strings.Contains(lower, "vs")
	hasDecision := strings.Contains(lower, "立即") || strings.Contains(lower, "开始") || strings.Contains(lower, "购买") || strings.Contains(lower, "注册") || strings.Contains(lower, "get started") || strings.Contains(lower, "buy now")

	score := 30
	if hasAwareness {
		score += 20
	}
	if hasConsideration {
		score += 25
	}
	if hasDecision {
		score += 25
	}

	if score < 60 {
		return score, "决策旅程覆盖不足，建议按“问题→方案→行动”三段式组织内容"
	}
	return score, "决策旅程覆盖较好"
}

func calculateCROOverallScore(dimensions []CRODimension) int {
	if len(dimensions) == 0 {
		return 0
	}
	total := 0
	for _, d := range dimensions {
		total += d.Score
	}
	return total / len(dimensions)
}

func filterEmptySuggestions(suggestions []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, s := range suggestions {
		key := strings.TrimSpace(s)
		if key != "" && !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	return result
}

func generateABTests(dimensions []CRODimension, title string) []string {
	tests := []string{}
	tests = append(tests, "测试标题变体：在标题中加入数字或利益点，如“"+title+"（2026 完全指南）”")
	tests = append(tests, "测试 CTA 文案：将“了解更多”改为“立即免费试用”，观察点击率变化")
	tests = append(tests, "测试首屏内容：在首屏 100 词内增加社会认同或具体数据")
	tests = append(tests, "测试 FAQ 模块：在页面底部增加 3-5 个常见异议 FAQ")
	return tests
}

func extractContext(plain, keyword string, radius int) string {
	idx := strings.Index(strings.ToLower(plain), strings.ToLower(keyword))
	if idx < 0 {
		return ""
	}
	start := idx - radius
	if start < 0 {
		start = 0
	}
	end := idx + len(keyword) + radius
	if end > len(plain) {
		end = len(plain)
	}
	return strings.TrimSpace(plain[start:end])
}

func estimatePosition(plain, keyword string) string {
	idx := strings.Index(strings.ToLower(plain), strings.ToLower(keyword))
	if idx < 0 {
		return "unknown"
	}
	if idx < len(plain)/3 {
		return "top"
	}
	if idx < 2*len(plain)/3 {
		return "middle"
	}
	return "bottom"
}
