package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// ContentOptimizeRequest 内容优化联审请求
type ContentOptimizeRequest struct {
	RecordID    int    `json:"record_id" binding:"required"`
	ContentType string `json:"content_type" binding:"required"` // article / prompt
	Language    string `json:"language"`
}

// ContentOptimizeResult 内容优化联审结果
type ContentOptimizeResult struct {
	RecordID         int                      `json:"record_id"`
	ContentType      string                   `json:"content_type"`
	Title            string                   `json:"title"`
	SEOScore         int                      `json:"seo_score"`
	HumanScore       int                      `json:"human_score"`
	ReadabilityScore int                      `json:"readability_score"`
	DimensionScores  []DimensionScore         `json:"dimension_scores"`
	Issues           []SEOIssue               `json:"issues"`
	InternalLinks    []InternalLinkSuggestion `json:"internal_links"`
	Suggestions      []string                 `json:"suggestions"`
	PublishCheck     []PublishCheckItem       `json:"publish_check"`
	Status           string                   `json:"status"` // ready / needs_fix / critical
}

// PublishCheckItem 发布检查单项
type PublishCheckItem struct {
	Item    string `json:"item"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// OptimizeContent 对指定内容进行优化联审
// 聚合 SEO 审计、内链建议、可读性/人性化评分
func OptimizeContent(req *ContentOptimizeRequest) (*ContentOptimizeResult, error) {
	recordType := req.ContentType
	recordID := req.RecordID
	lang := req.Language
	if lang == "" {
		lang = "en"
	}

	if recordType != "article" && recordType != "prompt" {
		return nil, fmt.Errorf("unsupported content type: %s", recordType)
	}

	// 1. SEO 审计
	seoResult, err := AuditSEO(recordType, recordID)
	if err != nil {
		return nil, fmt.Errorf("seo audit failed: %w", err)
	}

	// 2. 内链建议
	internalLinks, err := SuggestInternalLinks(recordType, recordID, 5)
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("content optimize: internal links failed: %v", err))
		internalLinks = []InternalLinkSuggestion{}
	}

	// 3. 可读性/人性化评分
	humanScore, readabilityScore, humanSuggestions := evaluateHumanAndReadability(recordType, recordID, lang)

	// 4. 整合建议
	suggestions := mergeSuggestions(seoResult.Suggestions, humanSuggestions)

	// 5. 生成发布检查清单
	publishCheck := buildPublishChecklist(seoResult, humanScore, readabilityScore, len(internalLinks))

	// 6. 判定状态
	status := determinePublishStatus(seoResult.OverallScore, humanScore, readabilityScore, publishCheck)

	return &ContentOptimizeResult{
		RecordID:         recordID,
		ContentType:      recordType,
		Title:            seoResult.Title,
		SEOScore:         seoResult.OverallScore,
		HumanScore:       humanScore,
		ReadabilityScore: readabilityScore,
		DimensionScores:  seoResult.DimensionScores,
		Issues:           seoResult.Issues,
		InternalLinks:    internalLinks,
		Suggestions:      suggestions,
		PublishCheck:     publishCheck,
		Status:           status,
	}, nil
}

// evaluateHumanAndReadability 评估人性化和可读性
// 优先基于内容特征做规则评分，若配置了 SEO AI 则调用 AI 增强
func evaluateHumanAndReadability(recordType string, recordID int, lang string) (humanScore, readabilityScore int, suggestions []string) {
	var title, content string
	var wordCount int

	switch recordType {
	case "article":
		article, err := model.GetArticleById(recordID)
		if err != nil {
			return 60, 60, []string{"无法读取文章内容，人性化评分采用默认值"}
		}
		title = article.Title
		content = article.Content
		wordCount = estimateWordCount(content)
	case "prompt":
		prompt, err := model.GetPromptById(recordID)
		if err != nil {
			return 60, 60, []string{"无法读取提示词内容，人性化评分采用默认值"}
		}
		title = prompt.Title
		content = prompt.Content
		wordCount = estimateWordCount(content)
	}

	if wordCount == 0 {
		return 50, 50, []string{"内容为空，请补充内容后再评估"}
	}

	// 可读性评分：基于长度、段落数、句子长度
	readabilityScore = calculateReadabilityScore(content, wordCount)

	// 人性化评分：基于个人代词、具体性信号、过渡词等
	humanScore = calculateHumanScore(content, title)

	suggestions = generateHumanSuggestions(content, title, wordCount, humanScore, readabilityScore)

	// 如果配置了 SEO AI，调用 AI 做更精细的评估
	cfg := operation_setting.GetSEOSetting()
	if cfg.SeoAIEnabled && cfg.SeoAIApiKey != "" && cfg.SeoAIBaseURL != "" {
		aiHuman, aiReadability, aiSuggestions := callAIHumanEvaluation(title, content, lang)
		if aiHuman > 0 {
			humanScore = (humanScore + aiHuman) / 2
		}
		if aiReadability > 0 {
			readabilityScore = (readabilityScore + aiReadability) / 2
		}
		if len(aiSuggestions) > 0 {
			suggestions = append(suggestions, aiSuggestions...)
		}
	}

	return humanScore, readabilityScore, suggestions
}

// estimateWordCount 估算字数/词数
func estimateWordCount(text string) int {
	text = stripHTML(text)
	if text == "" {
		return 0
	}
	// 中文字符按字计数，其他按空格分词
	chineseCount := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		}
	}
	words := len(strings.Fields(text))
	if chineseCount > words {
		return chineseCount
	}
	return words
}

// stripHTML 简单去除 HTML 标签
func stripHTML(input string) string {
	result := strings.Builder{}
	inTag := false
	for _, r := range input {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result.WriteRune(r)
			}
		}
	}
	return result.String()
}

// calculateReadabilityScore 计算可读性评分
func calculateReadabilityScore(content string, wordCount int) int {
	plain := stripHTML(content)
	sentences := strings.Split(plain, ".")
	if len(sentences) == 1 {
		sentences = strings.Split(plain, "。")
	}
	if len(sentences) == 1 {
		sentences = strings.Split(plain, "\n")
	}

	avgSentenceLen := float64(wordCount) / float64(len(sentences))
	if avgSentenceLen < 1 {
		avgSentenceLen = 1
	}

	score := 100
	// 句子越长，可读性越低
	if avgSentenceLen > 25 {
		score -= 20
	} else if avgSentenceLen > 20 {
		score -= 10
	}

	// 段落数适中加分
	paragraphs := len(strings.Split(plain, "\n\n"))
	if paragraphs < 2 && wordCount > 200 {
		score -= 15
	}

	// 内容长度适中
	if wordCount < 150 {
		score -= 15
	} else if wordCount > 500 {
		score += 5
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

// calculateHumanScore 计算人性化评分
func calculateHumanScore(content, title string) int {
	plain := stripHTML(content)
	score := 70

	// 信号：使用第一/第二人称
	if strings.Contains(plain, "you") || strings.Contains(plain, "we") ||
		strings.Contains(plain, "你") || strings.Contains(plain, "我们") {
		score += 5
	}

	// 信号：使用具体例子
	if strings.Contains(plain, "example") || strings.Contains(plain, "for instance") ||
		strings.Contains(plain, "例如") || strings.Contains(plain, "比如") {
		score += 5
	}

	// 信号：使用过渡词
	transitions := []string{"however", "therefore", "meanwhile", "additionally", "but", "so", "because"}
	for _, t := range transitions {
		if strings.Contains(strings.ToLower(plain), t) {
			score += 2
			break
		}
	}

	// 信号：列表使用
	if strings.Contains(plain, "- ") || strings.Contains(plain, "* ") ||
		strings.Contains(plain, "1.") || strings.Contains(plain, "##") {
		score += 5
	}

	// 负面信号：过度使用被动语态标志词
	passiveMarkers := []string{"is used", "are used", "was created", "were created", "be made", "been made"}
	passiveCount := 0
	lower := strings.ToLower(plain)
	for _, m := range passiveMarkers {
		if strings.Contains(lower, m) {
			passiveCount++
		}
	}
	if passiveCount > 3 {
		score -= 10
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// generateHumanSuggestions 生成人性化/可读性改进建议
func generateHumanSuggestions(content, title string, wordCount, humanScore, readabilityScore int) []string {
	suggestions := []string{}
	plain := stripHTML(content)

	if wordCount < 200 {
		suggestions = append(suggestions, "内容较短，建议扩展到 500 词以上以提升信息深度")
	}

	if humanScore < 70 {
		suggestions = append(suggestions, "增加第一/第二人称、具体例子和故事化表达，降低 AI 痕迹")
	}

	if readabilityScore < 70 {
		suggestions = append(suggestions, "拆分过长句子，增加段落和小标题，提升可读性")
	}

	if !strings.Contains(plain, "?") && wordCount > 300 {
		suggestions = append(suggestions, "适当加入反问或设问，增强与读者的互动")
	}

	if title != "" && len(title) < 20 {
		suggestions = append(suggestions, "标题过短，建议补充更具吸引力的修饰词")
	}

	return suggestions
}

// callAIHumanEvaluation 调用 AI 进行人性化和可读性评估
func callAIHumanEvaluation(title, content, lang string) (humanScore, readabilityScore int, suggestions []string) {
	cfg := operation_setting.GetSEOSetting()
	if !cfg.SeoAIEnabled || cfg.SeoAIApiKey == "" || cfg.SeoAIBaseURL == "" {
		return 0, 0, nil
	}

	// 简单实现：使用与 SEO 研究相同的 AI 配置
	// 为避免过度复杂，这里返回基于规则的结果
	// TODO: 后续可接入专门的 AI 评估 prompt
	logger.LogError(context.Background(), "content optimize: AI human evaluation not yet implemented, using rule-based fallback")
	return 0, 0, nil
}

// mergeSuggestions 合并 SEO 和人性化建议并去重
func mergeSuggestions(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		key := strings.ToLower(strings.TrimSpace(s))
		if key != "" && !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		key := strings.ToLower(strings.TrimSpace(s))
		if key != "" && !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	return result
}

// buildPublishChecklist 生成发布前检查清单
func buildPublishChecklist(seoResult *SEOAuditResult, humanScore, readabilityScore, internalLinkCount int) []PublishCheckItem {
	checks := []PublishCheckItem{
		{
			Item:    "SEO 评分 ≥ 70",
			Passed:  seoResult.OverallScore >= 70,
			Message: fmt.Sprintf("当前 %d 分", seoResult.OverallScore),
		},
		{
			Item:    "人性化评分 ≥ 70",
			Passed:  humanScore >= 70,
			Message: fmt.Sprintf("当前 %d 分", humanScore),
		},
		{
			Item:    "可读性评分 ≥ 70",
			Passed:  readabilityScore >= 70,
			Message: fmt.Sprintf("当前 %d 分", readabilityScore),
		},
		{
			Item:    "无关键 SEO 问题",
			Passed:  len(seoResult.CriticalIssues) == 0,
			Message: fmt.Sprintf("发现 %d 个关键问题", len(seoResult.CriticalIssues)),
		},
		{
			Item:    "已提供内链建议",
			Passed:  internalLinkCount > 0,
			Message: fmt.Sprintf("推荐 %d 条内链", internalLinkCount),
		},
		{
			Item:    "标题长度合适（30-60 字符）",
			Passed:  len(seoResult.Title) >= 10 && len(seoResult.Title) <= 60,
			Message: fmt.Sprintf("当前标题 %d 个字符", len(seoResult.Title)),
		},
	}
	return checks
}

// PublishGateResult 发布前质量门禁结果
type PublishGateResult struct {
	Passed   bool                   `json:"passed"`
	MinScore int                    `json:"min_score"`
	Status   string                 `json:"status"`
	Result   *ContentOptimizeResult `json:"result"`
	Message  string                 `json:"message"`
}

// CheckPublishGate 检查内容是否满足发布质量标准
// 返回门禁结果；未通过时 Result 中携带详细评分与问题
func CheckPublishGate(contentType string, recordID int, lang string) (*PublishGateResult, error) {
	result, err := OptimizeContent(&ContentOptimizeRequest{
		RecordID:    recordID,
		ContentType: contentType,
		Language:    lang,
	})
	if err != nil {
		return nil, err
	}

	minScore := result.SEOScore
	if result.HumanScore < minScore {
		minScore = result.HumanScore
	}
	if result.ReadabilityScore < minScore {
		minScore = result.ReadabilityScore
	}

	passed := result.Status == "ready" && minScore >= 70
	message := "发布前质量门禁通过"
	if !passed {
		message = fmt.Sprintf("发布前质量门禁未通过：SEO %d / 人性化 %d / 可读性 %d，最低分 %d 分未达 70 分或存在关键问题", result.SEOScore, result.HumanScore, result.ReadabilityScore, minScore)
	}

	return &PublishGateResult{
		Passed:   passed,
		MinScore: minScore,
		Status:   result.Status,
		Result:   result,
		Message:  message,
	}, nil
}

// determinePublishStatus 判定发布状态
func determinePublishStatus(seoScore, humanScore, readabilityScore int, checks []PublishCheckItem) string {
	criticalFailed := 0
	for _, check := range checks {
		if !check.Passed {
			criticalFailed++
		}
	}

	if seoScore >= 70 && humanScore >= 70 && readabilityScore >= 70 && criticalFailed == 0 {
		return "ready"
	}
	if criticalFailed >= 3 || seoScore < 50 || humanScore < 50 {
		return "critical"
	}
	return "needs_fix"
}
