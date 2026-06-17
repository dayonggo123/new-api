package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// ========== 任务状态定义 ==========

type SEOTaskStatus string

const (
	SEOTaskStatusPending   SEOTaskStatus = "pending"
	SEOTaskStatusRunning   SEOTaskStatus = "running"
	SEOTaskStatusCompleted SEOTaskStatus = "completed"
	SEOTaskStatusFailed    SEOTaskStatus = "failed"
)

type SEOTaskItemResult struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type SEOTask struct {
	ID        string              `json:"id"`
	Status    SEOTaskStatus       `json:"status"`
	Total     int                 `json:"total"`
	Completed int                 `json:"completed"`
	Failed    int                 `json:"failed"`
	Results   []SEOTaskItemResult `json:"results"`
	CreatedAt time.Time           `json:"created_at"`
}

// ========== 内存任务管理 ==========

var (
	seoTasks   = make(map[string]*SEOTask)
	seoTaskMu  sync.RWMutex
	seoTaskTTL = 2 * time.Hour
)

func init() {
	go cleanupSEOTasks()
	// 3分钟后启动 SEO 自动翻译/重试轮询
	time.AfterFunc(3*time.Minute, func() {
		if os.Getenv("DISABLE_SEO_AUTO_TRANSLATE") == "true" {
			common.SysLog("SEOAutoTranslate poller: disabled by env DISABLE_SEO_AUTO_TRANSLATE=true")
			return
		}
		go startSEOAutoRetryPoller()
	})
}

func cleanupSEOTasks() {
	for {
		time.Sleep(10 * time.Minute)
		now := time.Now()
		seoTaskMu.Lock()
		for id, task := range seoTasks {
			if now.Sub(task.CreatedAt) > seoTaskTTL {
				delete(seoTasks, id)
			}
		}
		seoTaskMu.Unlock()
	}
}

func StartSEOBatchTranslate(ids []int, targetLangs []string) string {
	taskID := fmt.Sprintf("seo-batch-%d", time.Now().UnixNano())
	task := &SEOTask{
		ID:        taskID,
		Status:    SEOTaskStatusPending,
		Total:     len(ids),
		Results:   make([]SEOTaskItemResult, 0, len(ids)),
		CreatedAt: time.Now(),
	}
	seoTaskMu.Lock()
	seoTasks[taskID] = task
	seoTaskMu.Unlock()

	go processSEOBatchTranslate(task, ids, targetLangs)
	return taskID
}

func GetSEOBatchTask(taskID string) *SEOTask {
	seoTaskMu.RLock()
	defer seoTaskMu.RUnlock()
	return seoTasks[taskID]
}

// ========== 批量处理逻辑 ==========

func processSEOBatchTranslate(task *SEOTask, ids []int, targetLangs []string) {
	task.Status = SEOTaskStatusRunning

	for _, id := range ids {
		err := processSinglePromptSEO(id, targetLangs)
		task.Completed++
		if err != nil {
			task.Failed++
			task.Results = append(task.Results, SEOTaskItemResult{
				ID:     id,
				Status: "failed",
				Error:  err.Error(),
			})
		} else {
			task.Results = append(task.Results, SEOTaskItemResult{
				ID:     id,
				Status: "success",
			})
		}
	}

	if task.Failed == task.Total {
		task.Status = SEOTaskStatusFailed
	} else {
		task.Status = SEOTaskStatusCompleted
	}
}

func processSinglePromptSEO(id int, targetLangs []string) error {
	var finalErr error
	defer func() {
		if finalErr != nil {
			model.DB.Model(&model.Prompt{}).Where("id = ?", id).Select("seo_translation_error", "updated_time").Updates(map[string]interface{}{
				"seo_translation_error": finalErr.Error(),
				"updated_time":          common.GetTimestamp(),
			})
		}
	}()

	promptWC, err := model.GetPromptById(id)
	if err != nil {
		finalErr = fmt.Errorf("get prompt failed: %w", err)
		return finalErr
	}
	if promptWC == nil || promptWC.Prompt == nil {
		finalErr = fmt.Errorf("prompt not found")
		return finalErr
	}

	p := promptWC.Prompt

	// 构建要翻译的完整 SEO JSON（与 GEO 翻译模式一致）
	sourceSEO := map[string]interface{}{}
	if p.SeoKeywords != "" {
		sourceSEO["seo_keywords"] = p.SeoKeywords
	}
	if p.Intro != "" {
		sourceSEO["intro"] = p.Intro
	}
	var sourceFaq []map[string]string
	if p.Faq != "" {
		if err := common.Unmarshal([]byte(p.Faq), &sourceFaq); err == nil && len(sourceFaq) > 0 {
			sourceSEO["faq"] = sourceFaq
		}
	}
	if len(sourceSEO) == 0 {
		return nil // 无内容可翻译
	}

	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		finalErr = fmt.Errorf("AI translation not configured")
		return finalErr
	}

	// 解析已有的 seo_i18n，避免覆盖已有翻译
	seoI18n := make(map[string]interface{})
	if p.SeoI18n != "" {
		_ = common.Unmarshal([]byte(p.SeoI18n), &seoI18n)
	}

	// 按语言逐个翻译
	for _, lang := range targetLangs {
		translated, err := translateSEOJSONWithAI(cfg, sourceSEO, "zh", lang)
		if err != nil || translated == nil || len(translated) == 0 {
			common.SysLog(fmt.Sprintf("SEO JSON translate failed: prompt %d lang=%s err=%v", id, lang, err))
			continue
		}

		// 校验翻译结果是否仍包含中文
		if !isValidSEOJSONTranslation(translated, "zh", lang) {
			common.SysLog(fmt.Sprintf("SEO JSON translate contains Chinese: prompt %d lang=%s", id, lang))
			continue
		}

		langData := map[string]interface{}{
			"seo_keywords": getStringFromMap(translated, "seo_keywords"),
			"intro":        getStringFromMap(translated, "intro"),
		}
		if faq, ok := translated["faq"]; ok {
			langData["faq"] = faq
		}

		seoI18n[lang] = langData
	}

	if len(seoI18n) == 0 {
		return nil
	}

	seoI18nJSON, err := common.Marshal(seoI18n)
	if err != nil {
		return fmt.Errorf("marshal seo_i18n failed: %w", err)
	}

	updates := map[string]interface{}{
		"seo_i18n":              string(seoI18nJSON),
		"seo_translation_error": "", // 成功时清空错误
		"updated_time":          common.GetTimestamp(),
	}
	if err := model.DB.Model(&model.Prompt{}).Where("id = ?", id).Select("seo_i18n", "seo_translation_error", "updated_time").Updates(updates).Error; err != nil {
		finalErr = err
		return err
	}

	// 翻译完成后自动触发 SEO 审计
	p.SeoI18n = string(seoI18nJSON)
	go AuditPromptSEO(p)

	return nil
}

// ========== AI 翻译调用（简化内联实现，避免循环依赖） ==========

var seoLangCodeToName = map[string]string{
	"zh": "Chinese", "en": "English", "fr": "French", "ru": "Russian",
	"ja": "Japanese", "vi": "Vietnamese", "ko": "Korean", "es": "Spanish",
	"de": "German", "pt": "Portuguese", "it": "Italian", "ar": "Arabic",
}

func getSEOLangName(code string) string {
	if name, ok := seoLangCodeToName[code]; ok {
		return name
	}
	if name, ok := seoLangCodeToName[strings.ToLower(code)]; ok {
		return name
	}
	return code
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	val, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func containsChinese(text string) bool {
	hasHan := false
	hasJapaneseKana := false
	hasKoreanHangul := false
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
		if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			hasJapaneseKana = true
		}
		if unicode.Is(unicode.Hangul, r) {
			hasKoreanHangul = true
		}
	}
	// 日文/韩文也会使用汉字（kanji/hanja），只要包含假名/谚文就不是中文
	if hasJapaneseKana || hasKoreanHangul {
		return false
	}
	return hasHan
}

// translateSEOJSONWithAI 采用与 GEO 翻译相同的整 JSON 翻译模式：
// 将中文 SEO 内容（seo_keywords/intro/faq）作为完整 JSON 一次性翻译，
// AI 只需保持 key 不变、翻译 value，prompt 更简洁，稳定性和速度更好。
func translateSEOJSONWithAI(cfg *operation_setting.TranslateSetting, sourceSEO map[string]interface{}, sourceLang, targetLang string) (map[string]interface{}, error) {
	sourceLangName := getSEOLangName(sourceLang)
	targetLangName := getSEOLangName(targetLang)

	sourceJSON, err := common.Marshal(sourceSEO)
	if err != nil {
		return nil, fmt.Errorf("marshal source seo failed: %w", err)
	}

	systemPrompt := fmt.Sprintf(`You are a professional translator. Translate the following JSON from %s to %s.

Rules:
1. Keep ALL keys exactly the same (do not translate keys)
2. Translate ALL string values and ALL array string items into %s
3. Return ONLY valid JSON. No markdown, no explanations.
4. Do not include any %s text in the output.`, sourceLangName, targetLangName, targetLangName, sourceLangName)

	response := callSEOTranslateAI(cfg, systemPrompt, string(sourceJSON))
	if response == "" {
		return nil, fmt.Errorf("empty translation response")
	}

	translatedJSON := extractSEOJSON(response)
	if translatedJSON == "" {
		return nil, fmt.Errorf("failed to extract JSON from translation")
	}

	var result map[string]interface{}
	if err := common.Unmarshal([]byte(translatedJSON), &result); err != nil {
		return nil, fmt.Errorf("invalid translated json: %w", err)
	}

	return result, nil
}

// extractSEOJSON 从 AI 响应中提取 JSON 对象（复用 GEO 的提取逻辑）
func extractSEOJSON(response string) string {
	response = strings.TrimSpace(response)

	// 去掉 markdown 代码块
	if strings.HasPrefix(response, "```") {
		start := strings.Index(response, "\n")
		if start != -1 {
			end := strings.LastIndex(response, "```")
			if end > start {
				response = strings.TrimSpace(response[start:end])
			}
		}
	}

	// 找第一个 { 和最后一个 }
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}

	return response[start : end+1]
}

// isValidSEOJSONTranslation 递归检查翻译后的 JSON 是否仍包含中文（目标语言非中文时）
func isValidSEOJSONTranslation(value interface{}, sourceLang, targetLang string) bool {
	if targetLang == "zh" {
		return true
	}
	switch v := value.(type) {
	case string:
		return !containsChinese(v)
	case []interface{}:
		for _, item := range v {
			if !isValidSEOJSONTranslation(item, sourceLang, targetLang) {
				return false
			}
		}
	case map[string]interface{}:
		for _, item := range v {
			if !isValidSEOJSONTranslation(item, sourceLang, targetLang) {
				return false
			}
		}
	}
	return true
}

func callSEOTranslateAI(cfg *operation_setting.TranslateSetting, systemPrompt, userPrompt string) string {
	reqBody := map[string]interface{}{
		"model": cfg.TranslateAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		common.SysLog("SEO batch translate marshal error: " + err.Error())
		return ""
	}

	common.SysLog(fmt.Sprintf("SEO batch translate request: model=%s", cfg.TranslateAIModel))

	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			common.SysLog(fmt.Sprintf("SEO batch translate retry %d/%d", attempt, maxRetries))
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		client := &http.Client{Timeout: 120 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 125*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TranslateAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			cancel()
			common.SysLog("SEO batch translate request error: " + err.Error())
			return ""
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.TranslateAIApiKey)

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			common.SysLog("SEO batch translate do error (attempt " + strconv.Itoa(attempt+1) + "): " + err.Error())
			if attempt < maxRetries {
				continue
			}
			return ""
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable {
			common.SysLog(fmt.Sprintf("SEO batch translate server error %d, retrying", resp.StatusCode))
			if attempt < maxRetries {
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			common.SysLog(fmt.Sprintf("SEO batch translate non-200: %d", resp.StatusCode))
			return ""
		}

		var apiResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := common.Unmarshal(bodyBytes, &apiResp); err != nil {
			common.SysLog("SEO batch translate decode error: " + err.Error())
			return ""
		}
		if len(apiResp.Choices) == 0 {
			common.SysLog("SEO batch translate empty choices")
			return ""
		}

		content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
		// 去除首尾引号
		if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
			content = content[1 : len(content)-1]
		}
		common.SysLog(fmt.Sprintf("SEO batch translate success (attempt %d)", attempt+1))
		return content
	}

	return ""
}

// ========== SEO 自动翻译/重试轮询 ==========

var seoAutoTranslateTargetLangs = []string{"en", "fr", "ru", "ja", "vi", "ko", "es", "de", "pt", "it", "ar"}

func startSEOAutoRetryPoller() {
	common.SysLog("SEOAutoTranslate poller: started (interval=1min)")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// 检查运行时开关（环境变量优先，其次检查 Option 表）
		if os.Getenv("DISABLE_SEO_AUTO_TRANSLATE") == "true" {
			continue // 环境变量强制禁用
		}
		common.OptionMapRWMutex.Lock()
		optionEnabled := common.OptionMap["AutoTranslateEnabled"] != "false"
		common.OptionMapRWMutex.Unlock()
		if !optionEnabled {
			continue // 运行时开关已关闭
		}

		cfg := operation_setting.GetTranslateSetting()
		if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
			common.SysLog("SEOAutoTranslate poller: skipped — AI translation not configured")
			continue
		}
		pollAndAutoTranslateSEO()
	}
}

func getMissingSEOLangs(recordType string, id int, targetLangs []string) []string {
	var existingI18n string
	if recordType == "prompt" {
		var p model.Prompt
		if err := model.DB.Model(&model.Prompt{}).Select("seo_i18n").Where("id = ?", id).First(&p).Error; err != nil {
			return targetLangs
		}
		existingI18n = p.SeoI18n
	} else {
		var a model.Article
		if err := model.DB.Model(&model.Article{}).Select("seo_i18n").Where("id = ?", id).First(&a).Error; err != nil {
			return targetLangs
		}
		existingI18n = a.SeoI18n
	}

	if existingI18n == "" {
		return targetLangs
	}

	var seoMap map[string]interface{}
	if err := common.Unmarshal([]byte(existingI18n), &seoMap); err != nil {
		return targetLangs
	}

	missing := []string{}
	for _, lang := range targetLangs {
		if _, ok := seoMap[lang]; !ok {
			missing = append(missing, lang)
		}
	}
	return missing
}

func pollAndAutoTranslateSEO() {
	common.SysLog("SEOAutoTranslate poller: starting scan")
	const batchSize = 20
	cooldown := time.Now().Add(-3 * time.Minute).Unix()

	// Prompts: 分两部分查询
	// A. 从未翻译过（seo_i18n 为空）— 不限冷却时间，这是最优先的
	// B. 已部分翻译（seo_i18n 不为空）— 需要冷却时间，避免频繁扫描
	var promptIDs []int
	var retryPromptIDs []int

	// A. 从未翻译的 prompts
	err := model.DB.Model(&model.Prompt{}).
		Select("id").
		Where("(seo_i18n = ? OR seo_i18n IS NULL) AND ((seo_keywords IS NOT NULL AND seo_keywords != ?) OR (intro IS NOT NULL AND intro != ?) OR (faq IS NOT NULL AND faq != ?))", "", "", "", "").
		Limit(batchSize).
		Order("id desc").
		Pluck("id", &promptIDs).Error
	if err != nil {
		common.SysLog("SEOAutoTranslate poller query new prompts error: " + err.Error())
	}
	common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: found %d new prompts (seo_i18n empty)", len(promptIDs)))

	// B. 已部分翻译的 prompts（冷却时间已过）
	var partialPromptIDs []int
	if len(promptIDs) < batchSize {
		err = model.DB.Model(&model.Prompt{}).
			Select("id").
			Where("(seo_i18n IS NOT NULL AND seo_i18n != ?) AND ((seo_keywords IS NOT NULL AND seo_keywords != ?) OR (intro IS NOT NULL AND intro != ?) OR (faq IS NOT NULL AND faq != ?)) AND updated_time < ?", "", "", "", "", cooldown).
			Limit(batchSize - len(promptIDs)).
			Order("updated_time asc").
			Pluck("id", &partialPromptIDs).Error
		if err != nil {
			common.SysLog("SEOAutoTranslate poller query partial prompts error: " + err.Error())
		}
		promptIDs = append(promptIDs, partialPromptIDs...)
		common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: found %d partial prompts", len(partialPromptIDs)))
	}

	// C. 失败重试（冷却时间已过）
	if len(promptIDs) < batchSize {
		err = model.DB.Model(&model.Prompt{}).
			Select("id").
			Where("seo_translation_error != ? AND updated_time < ?", "", cooldown).
			Limit(batchSize - len(promptIDs)).
			Order("updated_time asc").
			Pluck("id", &retryPromptIDs).Error
		if err != nil {
			common.SysLog("SEOAutoTranslate poller query retry prompts error: " + err.Error())
		}
		promptIDs = append(promptIDs, retryPromptIDs...)
		common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: found %d retry prompts", len(retryPromptIDs)))
	}

	for _, id := range promptIDs {
		missingLangs := getMissingSEOLangs("prompt", id, seoAutoTranslateTargetLangs)
		if len(missingLangs) == 0 {
			// 已经完整，清空错误并更新时间避免下次重复扫描
			model.DB.Model(&model.Prompt{}).Where("id = ?", id).Select("seo_translation_error", "updated_time").Updates(map[string]interface{}{
				"seo_translation_error": "",
				"updated_time":          common.GetTimestamp(),
			})
			continue
		}
		err := processSinglePromptSEO(id, missingLangs)
		if err != nil {
			common.SysLog(fmt.Sprintf("SEOAutoTranslate failed: prompt %d, error=%s", id, err.Error()))
		}
		time.Sleep(2 * time.Second)
	}
	if len(promptIDs) > 0 {
		common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: triggered %d prompts (%d new, %d partial, %d retry)", len(promptIDs), len(promptIDs)-len(partialPromptIDs)-len(retryPromptIDs), len(partialPromptIDs), len(retryPromptIDs)))
	} else {
		common.SysLog("SEOAutoTranslate poller: no prompts to translate")
	}

	// Articles: 同样分两部分查询
	var articleIDs []int
	var retryArticleIDs []int

	// A. 从未翻译的 articles
	err = model.DB.Model(&model.Article{}).
		Select("id").
		Where("(seo_i18n = ? OR seo_i18n IS NULL) AND ((seo_keywords IS NOT NULL AND seo_keywords != ?) OR (intro IS NOT NULL AND intro != ?) OR (faq IS NOT NULL AND faq != ?))", "", "", "", "").
		Limit(batchSize).
		Order("id desc").
		Pluck("id", &articleIDs).Error
	if err != nil {
		common.SysLog("SEOAutoTranslate poller query new articles error: " + err.Error())
	}
	common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: found %d new articles (seo_i18n empty)", len(articleIDs)))

	// B. 已部分翻译的 articles（冷却时间已过）
	var partialArticleIDs []int
	if len(articleIDs) < batchSize {
		err = model.DB.Model(&model.Article{}).
			Select("id").
			Where("(seo_i18n IS NOT NULL AND seo_i18n != ?) AND ((seo_keywords IS NOT NULL AND seo_keywords != ?) OR (intro IS NOT NULL AND intro != ?) OR (faq IS NOT NULL AND faq != ?)) AND updated_time < ?", "", "", "", "", cooldown).
			Limit(batchSize - len(articleIDs)).
			Order("updated_time asc").
			Pluck("id", &partialArticleIDs).Error
		if err != nil {
			common.SysLog("SEOAutoTranslate poller query partial articles error: " + err.Error())
		}
		articleIDs = append(articleIDs, partialArticleIDs...)
		common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: found %d partial articles", len(partialArticleIDs)))
	}

	// C. 失败重试
	if len(articleIDs) < batchSize {
		err = model.DB.Model(&model.Article{}).
			Select("id").
			Where("seo_translation_error != ? AND updated_time < ?", "", cooldown).
			Limit(batchSize - len(articleIDs)).
			Order("updated_time asc").
			Pluck("id", &retryArticleIDs).Error
		if err != nil {
			common.SysLog("SEOAutoTranslate poller query retry articles error: " + err.Error())
		}
		articleIDs = append(articleIDs, retryArticleIDs...)
		common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: found %d retry articles", len(retryArticleIDs)))
	}

	for _, id := range articleIDs {
		missingLangs := getMissingSEOLangs("article", id, seoAutoTranslateTargetLangs)
		if len(missingLangs) == 0 {
			model.DB.Model(&model.Article{}).Where("id = ?", id).Select("seo_translation_error", "updated_time").Updates(map[string]interface{}{
				"seo_translation_error": "",
				"updated_time":          common.GetTimestamp(),
			})
			continue
		}
		err := processSingleArticleSEO(id, missingLangs)
		if err != nil {
			common.SysLog(fmt.Sprintf("SEOAutoTranslate failed: article %d, error=%s", id, err.Error()))
		}
		time.Sleep(2 * time.Second)
	}
	if len(articleIDs) > 0 {
		common.SysLog(fmt.Sprintf("SEOAutoTranslate poller: triggered %d articles (%d retry)", len(articleIDs), len(retryArticleIDs)))
	}
}

// processSingleArticleSEO 处理单条 Article 的 SEO 多语言翻译
func processSingleArticleSEO(id int, targetLangs []string) error {
	var finalErr error
	defer func() {
		if finalErr != nil {
			model.DB.Model(&model.Article{}).Where("id = ?", id).Select("seo_translation_error", "updated_time").Updates(map[string]interface{}{
				"seo_translation_error": finalErr.Error(),
				"updated_time":          common.GetTimestamp(),
			})
		}
	}()

	article, err := model.GetArticleById(id)
	if err != nil {
		finalErr = fmt.Errorf("get article failed: %w", err)
		return finalErr
	}
	if article == nil {
		finalErr = fmt.Errorf("article not found")
		return finalErr
	}

	// 构建要翻译的完整 SEO JSON（与 GEO 翻译模式一致）
	sourceSEO := map[string]interface{}{}
	if article.SeoKeywords != "" {
		sourceSEO["seo_keywords"] = article.SeoKeywords
	}
	if article.Intro != "" {
		sourceSEO["intro"] = article.Intro
	}
	var sourceFaq []map[string]string
	if article.Faq != "" {
		if err := common.Unmarshal([]byte(article.Faq), &sourceFaq); err == nil && len(sourceFaq) > 0 {
			sourceSEO["faq"] = sourceFaq
		}
	}
	if len(sourceSEO) == 0 {
		return nil // 无内容可翻译
	}

	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		finalErr = fmt.Errorf("AI translation not configured")
		return finalErr
	}

	// 解析已有的 seo_i18n，避免覆盖已有翻译
	seoI18n := make(map[string]interface{})
	if article.SeoI18n != "" {
		_ = common.Unmarshal([]byte(article.SeoI18n), &seoI18n)
	}

	// 按语言逐个翻译
	for _, lang := range targetLangs {
		translated, err := translateSEOJSONWithAI(cfg, sourceSEO, "zh", lang)
		if err != nil || translated == nil || len(translated) == 0 {
			common.SysLog(fmt.Sprintf("SEO JSON translate failed: article %d lang=%s err=%v", id, lang, err))
			continue
		}

		// 校验翻译结果是否仍包含中文
		if !isValidSEOJSONTranslation(translated, "zh", lang) {
			common.SysLog(fmt.Sprintf("SEO JSON translate contains Chinese: article %d lang=%s", id, lang))
			continue
		}

		langData := map[string]interface{}{
			"seo_keywords": getStringFromMap(translated, "seo_keywords"),
			"intro":        getStringFromMap(translated, "intro"),
		}
		if faq, ok := translated["faq"]; ok {
			langData["faq"] = faq
		}

		seoI18n[lang] = langData
	}

	if len(seoI18n) == 0 {
		return nil
	}

	seoI18nJSON, err := common.Marshal(seoI18n)
	if err != nil {
		return fmt.Errorf("marshal seo_i18n failed: %w", err)
	}

	updates := map[string]interface{}{
		"seo_i18n":              string(seoI18nJSON),
		"seo_translation_error": "",
		"updated_time":          common.GetTimestamp(),
	}
	if err := model.DB.Model(&model.Article{}).Where("id = ?", id).Select("seo_i18n", "seo_translation_error", "updated_time").Updates(updates).Error; err != nil {
		finalErr = err
		return err
	}

	// 翻译完成后自动触发 SEO 审计
	article.SeoI18n = string(seoI18nJSON)
	go AuditArticleSEO(article)

	return nil
}
