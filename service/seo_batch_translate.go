package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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

type seoItem struct{ Key, Text string }

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
	items := []seoItem{
		{Key: "seo_keywords", Text: p.SeoKeywords},
		{Key: "intro", Text: p.Intro},
	}

	// 解析 FAQ
	faqMap := make(map[int]map[string]string)
	if p.Faq != "" {
		var faqArr []map[string]string
		if err := common.Unmarshal([]byte(p.Faq), &faqArr); err == nil {
			for idx, f := range faqArr {
				if q, ok := f["question"]; ok && q != "" {
					items = append(items, seoItem{Key: fmt.Sprintf("faq_%d_question", idx), Text: q})
					if faqMap[idx] == nil {
						faqMap[idx] = make(map[string]string)
					}
					faqMap[idx]["question"] = q
				}
				if a, ok := f["answer"]; ok && a != "" {
					items = append(items, seoItem{Key: fmt.Sprintf("faq_%d_answer", idx), Text: a})
					if faqMap[idx] == nil {
						faqMap[idx] = make(map[string]string)
					}
					faqMap[idx]["answer"] = a
				}
			}
		}
	}

	validItems := make([]seoItem, 0, len(items))
	for _, it := range items {
		if it.Text != "" {
			validItems = append(validItems, it)
		}
	}
	if len(validItems) == 0 {
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
		translations := translateSEOItemsWithAI(cfg, validItems, "zh", lang)
		if translations == nil || len(translations) == 0 {
			continue
		}

		langData := map[string]interface{}{
			"seo_keywords": translations["seo_keywords"],
			"intro":        translations["intro"],
		}

		// 重组 FAQ
		newFaqMap := make(map[int]map[string]string)
		faqRe := regexp.MustCompile(`^faq_(\d+)_(question|answer)$`)
		for key, val := range translations {
			matches := faqRe.FindStringSubmatch(key)
			if len(matches) == 3 {
				idx, _ := strconv.Atoi(matches[1])
				field := matches[2]
				if newFaqMap[idx] == nil {
					newFaqMap[idx] = make(map[string]string)
				}
				newFaqMap[idx][field] = val
			}
		}

		if len(newFaqMap) > 0 {
			indices := make([]int, 0, len(newFaqMap))
			for idx := range newFaqMap {
				indices = append(indices, idx)
			}
			sort.Ints(indices)
			var faqList []map[string]string
			for _, idx := range indices {
				faqList = append(faqList, newFaqMap[idx])
			}
			langData["faq"] = faqList
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

func translateSEOItemsWithAI(cfg *operation_setting.TranslateSetting, items []seoItem, sourceLang, targetLang string) map[string]string {
	result := make(map[string]string)

	sourceLangName := getSEOLangName(sourceLang)
	targetLangName := getSEOLangName(targetLang)

	// 构建 fields JSON
	fieldsMap := make(map[string]string)
	for _, it := range items {
		if it.Text != "" {
			fieldsMap[it.Key] = it.Text
		}
	}
	fieldsJSON, _ := common.Marshal(fieldsMap)

	// 读取 Skill 模板
	systemPrompt := ""
	userPromptTemplate := ""
	if skill, err := model.GetSkillBySkillId("batch-translate"); err == nil && skill.SystemPromptTemplate != "" {
		systemPrompt = skill.SystemPromptTemplate
		userPromptTemplate = skill.UserPromptTemplate
	}
	if systemPrompt == "" {
		systemPrompt = "You are a professional translator. Your ONLY task is to translate the given fields from {{sourceLang}} to {{targetLang}}. You MUST respond entirely in {{targetLang}}. Do NOT respond in {{sourceLang}} or any other language. If you accidentally start writing in {{sourceLang}}, STOP and rewrite everything in {{targetLang}}. Preserve all variable placeholders like {{variableName}} exactly as-is. Return results in valid JSON format with the exact same keys as the input. No explanations, no markdown code blocks around the JSON."
	}
	if userPromptTemplate == "" {
		userPromptTemplate = "Translate the following fields from {{sourceLang}} to {{targetLang}}.\n\nInput (JSON):\n{{fields}}\n\nRules:\n1. Return ONLY a JSON object with the same keys\n2. All values must be pure {{targetLang}} text — absolutely NO {{sourceLang}} allowed\n3. Preserve {{variables}} exactly as-is\n4. Do not add explanations or wrap in markdown\n5. If any value is still in {{sourceLang}}, you have failed — translate again into {{targetLang}}\n\nOutput format example:\n{\"title\":\"Translated Title\",\"summary\":\"Translated summary...\",\"content\":\"Translated content...\"}"
	}

	systemPrompt = strings.ReplaceAll(systemPrompt, "{{sourceLang}}", sourceLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{targetLang}}", targetLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{language}}", targetLangName)
	userPrompt := strings.ReplaceAll(userPromptTemplate, "{{sourceLang}}", sourceLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{targetLang}}", targetLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{language}}", targetLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", string(fieldsJSON))

	response := callSEOTranslateAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		for _, it := range items {
			result[it.Key] = it.Text
		}
		return result
	}

	// 去掉 markdown 代码块
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		start := strings.Index(response, "\n")
		if start != -1 {
			end := strings.LastIndex(response, "```")
			if end > start {
				response = strings.TrimSpace(response[start:end])
			}
		}
	}

	// JSON 解析
	var jsonResult map[string]string
	if err := common.Unmarshal([]byte(response), &jsonResult); err == nil && len(jsonResult) > 0 {
		for k, v := range jsonResult {
			result[k] = v
		}
	} else {
		// 回退到 key: value 格式
		lines := strings.Split(response, "\n")
		var currentKey string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, ":") {
				idx := strings.Index(line, ": ")
				if idx == -1 {
					idx = strings.Index(line, ":")
				}
				if idx > 0 {
					currentKey = strings.TrimSpace(line[:idx])
					val := strings.TrimSpace(line[idx+1:])
					if currentKey != "" {
						result[currentKey] = val
					}
				}
			} else if currentKey != "" {
				result[currentKey] += "\n" + line
			}
		}
	}

	// 补翻缺失的字段
	for _, it := range items {
		if result[it.Key] == "" || result[it.Key] == it.Text {
			translated := callSEOTranslateAI(cfg,
				"You are a translator. CRITICAL: Your response MUST be in "+targetLangName+". ZERO "+sourceLangName+" words allowed.",
				"Translate to "+targetLangName+" ONLY:\n\n"+it.Text)
			if translated != "" && translated != it.Text {
				result[it.Key] = translated
			} else {
				result[it.Key] = it.Text
			}
		}
	}

	return result
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
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cfg := operation_setting.GetTranslateSetting()
		if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
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
	cooldown := time.Now().Add(-30 * time.Minute).Unix()

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

	items := []seoItem{
		{Key: "seo_keywords", Text: article.SeoKeywords},
		{Key: "intro", Text: article.Intro},
	}

	// 解析 FAQ
	if article.Faq != "" {
		var faqArr []map[string]string
		if err := common.Unmarshal([]byte(article.Faq), &faqArr); err == nil {
			for idx, f := range faqArr {
				if q, ok := f["question"]; ok && q != "" {
					items = append(items, seoItem{Key: fmt.Sprintf("faq_%d_question", idx), Text: q})
				}
				if a, ok := f["answer"]; ok && a != "" {
					items = append(items, seoItem{Key: fmt.Sprintf("faq_%d_answer", idx), Text: a})
				}
			}
		}
	}

	validItems := make([]seoItem, 0, len(items))
	for _, it := range items {
		if it.Text != "" {
			validItems = append(validItems, it)
		}
	}
	if len(validItems) == 0 {
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
		translations := translateSEOItemsWithAI(cfg, validItems, "zh", lang)
		if translations == nil || len(translations) == 0 {
			continue
		}

		langData := map[string]interface{}{
			"seo_keywords": translations["seo_keywords"],
			"intro":        translations["intro"],
		}

		// 重组 FAQ
		newFaqMap := make(map[int]map[string]string)
		faqRe := regexp.MustCompile(`^faq_(\d+)_(question|answer)$`)
		for key, val := range translations {
			matches := faqRe.FindStringSubmatch(key)
			if len(matches) == 3 {
				idx, _ := strconv.Atoi(matches[1])
				field := matches[2]
				if newFaqMap[idx] == nil {
					newFaqMap[idx] = make(map[string]string)
				}
				newFaqMap[idx][field] = val
			}
		}

		if len(newFaqMap) > 0 {
			indices := make([]int, 0, len(newFaqMap))
			for idx := range newFaqMap {
				indices = append(indices, idx)
			}
			sort.Ints(indices)
			var faqList []map[string]string
			for _, idx := range indices {
				faqList = append(faqList, newFaqMap[idx])
			}
			langData["faq"] = faqList
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
