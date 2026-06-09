package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// ========== 自动翻译任务定义 ==========

type AutoTranslateStatus string

const (
	AutoTranslateStatusPending   AutoTranslateStatus = "pending"
	AutoTranslateStatusRunning   AutoTranslateStatus = "running"
	AutoTranslateStatusCompleted AutoTranslateStatus = "completed"
	AutoTranslateStatusFailed    AutoTranslateStatus = "failed"
)

type AutoTranslateTask struct {
	ID        string              `json:"id"`
	Type      string              `json:"type"`      // "prompt" or "article"
	RecordID  int                 `json:"record_id"`
	Status    AutoTranslateStatus `json:"status"`
	Progress  int                 `json:"progress"`
	Total     int                 `json:"total"`
	Error     string              `json:"error,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
}

// 目标语言列表（12 种，排除中文）
var autoTranslateTargetLangs = []string{
	"en", "fr", "ru", "ja", "vi", "ko", "es", "de", "pt", "it", "ar",
}

// ========== 内存任务管理 ==========

var (
	autoTranslateTasks   = make(map[string]*AutoTranslateTask)
	autoTranslateMu      sync.RWMutex
	autoTranslateTTL     = 2 * time.Hour
	autoTranslateRunning = make(map[string]bool) // 防止同一记录重复翻译
	autoTranslateRunMu   sync.Mutex
)

func init() {
	go cleanupAutoTranslateTasks()
}

func cleanupAutoTranslateTasks() {
	for {
		time.Sleep(10 * time.Minute)
		now := time.Now()
		autoTranslateMu.Lock()
		for id, task := range autoTranslateTasks {
			if now.Sub(task.CreatedAt) > autoTranslateTTL {
				delete(autoTranslateTasks, id)
			}
		}
		autoTranslateMu.Unlock()
	}
}

// ========== 公共接口 ==========

// StartAutoTranslate 启动自动翻译任务（创建/更新记录后调用）
// 同一记录短时间内不会重复触发
func StartAutoTranslate(recordType string, recordID int) string {
	key := fmt.Sprintf("%s-%d", recordType, recordID)

	autoTranslateRunMu.Lock()
	if autoTranslateRunning[key] {
		autoTranslateRunMu.Unlock()
		common.SysLog(fmt.Sprintf("AutoTranslate skipped: %s is already running", key))
		return ""
	}
	autoTranslateRunning[key] = true
	autoTranslateRunMu.Unlock()

	taskID := fmt.Sprintf("at-%s-%d-%d", recordType, recordID, time.Now().UnixNano())
	task := &AutoTranslateTask{
		ID:        taskID,
		Type:      recordType,
		RecordID:  recordID,
		Status:    AutoTranslateStatusPending,
		Total:     len(autoTranslateTargetLangs),
		CreatedAt: time.Now(),
	}
	autoTranslateMu.Lock()
	autoTranslateTasks[taskID] = task
	autoTranslateMu.Unlock()

	go processAutoTranslate(task, key)
	return taskID
}

// GetAutoTranslateTask 查询任务状态
func GetAutoTranslateTask(taskID string) *AutoTranslateTask {
	autoTranslateMu.RLock()
	defer autoTranslateMu.RUnlock()
	return autoTranslateTasks[taskID]
}

// ========== 处理逻辑 ==========

func processAutoTranslate(task *AutoTranslateTask, runningKey string) {
	defer func() {
		autoTranslateRunMu.Lock()
		delete(autoTranslateRunning, runningKey)
		autoTranslateRunMu.Unlock()
	}()

	// defer 中统一处理失败状态写入数据库
	defer func() {
		if task.Status == AutoTranslateStatusFailed && task.Error != "" {
			failUpdates := map[string]interface{}{
				"is_translated":     false,
				"translation_error": task.Error,
			}
			if task.Type == "prompt" {
				model.DB.Model(&model.Prompt{}).Where("id = ?", task.RecordID).Select("is_translated", "translation_error").Updates(failUpdates)
			} else {
				model.DB.Model(&model.Article{}).Where("id = ?", task.RecordID).Select("is_translated", "translation_error").Updates(failUpdates)
			}
		}
	}()

	task.Status = AutoTranslateStatusRunning

	var title, content string
	var recordExists bool

	if task.Type == "prompt" {
		pwc, err := model.GetPromptById(task.RecordID)
		if err != nil || pwc == nil || pwc.Prompt == nil {
			task.Status = AutoTranslateStatusFailed
			task.Error = "prompt not found"
			return
		}
		title = pwc.Prompt.Title
		content = pwc.Prompt.Content
		recordExists = true
	} else if task.Type == "article" {
		article, err := model.GetArticleById(task.RecordID)
		if err != nil || article == nil {
			task.Status = AutoTranslateStatusFailed
			task.Error = "article not found"
			return
		}
		title = article.Title
		content = article.Content
		recordExists = true
	} else {
		task.Status = AutoTranslateStatusFailed
		task.Error = "unknown record type: " + task.Type
		return
	}

	if !recordExists || title == "" || content == "" {
		task.Status = AutoTranslateStatusFailed
		task.Error = "empty title or content"
		return
	}

	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		task.Status = AutoTranslateStatusFailed
		task.Error = "AI translation not configured"
		return
	}

	// 构建翻译项
	items := []translateItem{
		{Key: "title", Text: title},
		{Key: "content", Text: content},
	}

	titleI18n := make(map[string]string)
	contentI18n := make(map[string]string)
	failedLangs := []string{}

	for _, lang := range autoTranslateTargetLangs {
		result := translateItemsWithAI(cfg, items, "zh", lang)
		task.Progress++

		if result == nil || len(result) == 0 {
			failedLangs = append(failedLangs, lang)
			continue
		}

		if t, ok := result["title"]; ok && t != "" && t != title {
			titleI18n[lang] = t
		}
		if c, ok := result["content"]; ok && c != "" && c != content {
			contentI18n[lang] = c
		}

		// 每语言间隔，避免限流
		time.Sleep(500 * time.Millisecond)
	}

	if len(titleI18n) == 0 && len(contentI18n) == 0 {
		task.Status = AutoTranslateStatusFailed
		task.Error = "all translations failed"
		return
	}

	// 保存到数据库
	updates := map[string]interface{}{
		"is_translated":      true,
		"translation_error":  "", // 成功时清空错误
	}
	if len(titleI18n) > 0 {
		titleI18nJSON, _ := common.Marshal(titleI18n)
		updates["title_i18n"] = string(titleI18nJSON)
	}
	if len(contentI18n) > 0 {
		contentI18nJSON, _ := common.Marshal(contentI18n)
		updates["i18n"] = string(contentI18nJSON)
	}
	// en 单独存到 content_en
	if en, ok := contentI18n["en"]; ok && en != "" {
		updates["content_en"] = en
	}

	var saveErr error
	if task.Type == "prompt" {
		saveErr = model.DB.Model(&model.Prompt{}).Where("id = ?", task.RecordID).Updates(updates).Error
	} else {
		saveErr = model.DB.Model(&model.Article{}).Where("id = ?", task.RecordID).Updates(updates).Error
	}

	if saveErr != nil {
		task.Status = AutoTranslateStatusFailed
		task.Error = "save failed: " + saveErr.Error()
		return
	}

	if len(failedLangs) > 0 {
		task.Error = "partial failure: " + strings.Join(failedLangs, ", ")
	}
	task.Status = AutoTranslateStatusCompleted
	common.SysLog(fmt.Sprintf("AutoTranslate completed: %s %d, title_langs=%d, content_langs=%d, failed=%v",
		task.Type, task.RecordID, len(titleI18n), len(contentI18n), failedLangs))
}

// ========== AI 翻译调用（内联实现，避免循环依赖） ==========

type translateItem struct{ Key, Text string }

var autoLangCodeToName = map[string]string{
	"zh": "Chinese", "en": "English", "fr": "French", "ru": "Russian",
	"ja": "Japanese", "vi": "Vietnamese", "ko": "Korean", "es": "Spanish",
	"de": "German", "pt": "Portuguese", "it": "Italian", "ar": "Arabic",
}

func getAutoLangName(code string) string {
	if name, ok := autoLangCodeToName[code]; ok {
		return name
	}
	if name, ok := autoLangCodeToName[strings.ToLower(code)]; ok {
		return name
	}
	return code
}

func translateItemsWithAI(cfg *operation_setting.TranslateSetting, items []translateItem, sourceLang, targetLang string) map[string]string {
	result := make(map[string]string)

	sourceLangName := getAutoLangName(sourceLang)
	targetLangName := getAutoLangName(targetLang)

	// 若 content 过长，先拆出来单独翻译
	var contentItem *translateItem
	var shortItems []translateItem
	for i := range items {
		if items[i].Key == "content" && len(items[i].Text) > 300 {
			contentItem = &items[i]
		} else {
			shortItems = append(shortItems, items[i])
		}
	}

	// 读取 batch-translate Skill 模板
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

	fieldsMap := make(map[string]string)
	for _, item := range shortItems {
		if item.Text != "" {
			fieldsMap[item.Key] = item.Text
		}
	}
	fieldsJSON, _ := common.Marshal(fieldsMap)

	systemPrompt = strings.ReplaceAll(systemPrompt, "{{sourceLang}}", sourceLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{targetLang}}", targetLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{language}}", targetLangName)
	userPrompt := strings.ReplaceAll(userPromptTemplate, "{{sourceLang}}", sourceLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{targetLang}}", targetLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{language}}", targetLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", string(fieldsJSON))

	response := callAutoTranslateAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		for _, item := range items {
			result[item.Key] = item.Text
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

	// 若 content 被拆分出来，单独翻译
	if contentItem != nil {
		translatedContent := translateSingleAutoWithAI(cfg, contentItem.Text, sourceLang, targetLang)
		if translatedContent != "" && translatedContent != contentItem.Text {
			result[contentItem.Key] = translatedContent
		} else {
			result[contentItem.Key] = contentItem.Text
		}
	}

	// 补翻缺失的字段
	for _, item := range items {
		if result[item.Key] == "" || result[item.Key] == item.Text {
			translated := translateSingleAutoWithAI(cfg, item.Text, sourceLang, targetLang)
			if translated != "" && translated != item.Text {
				result[item.Key] = translated
			} else {
				result[item.Key] = item.Text
			}
		}
	}

	return result
}

func translateSingleAutoWithAI(cfg *operation_setting.TranslateSetting, text, sourceLang, targetLang string) string {
	sourceLangName := getAutoLangName(sourceLang)
	targetLangName := getAutoLangName(targetLang)

	systemPrompt := "You are a professional translator. Your ONLY task is to translate text. You MUST respond entirely in " + targetLangName + ". Do NOT respond in " + sourceLangName + " or any other language. Do not add explanations, notes, or the original text — output ONLY the translated text in " + targetLangName + "."
	userPrompt := "Translate the following text from " + sourceLangName + " to " + targetLangName + ". Your response must be ONLY the translated text in " + targetLangName + ", nothing else:\n\n\"\"\"\n" + text + "\n\"\"\""

	return callAutoTranslateAI(cfg, systemPrompt, userPrompt)
}

func callAutoTranslateAI(cfg *operation_setting.TranslateSetting, systemPrompt, userPrompt string) string {
	reqBody := map[string]interface{}{
		"model": cfg.TranslateAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  8192,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		common.SysLog("AutoTranslate marshal error: " + err.Error())
		return ""
	}

	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		client := &http.Client{Timeout: 120 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 125*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TranslateAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			cancel()
			return ""
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.TranslateAIApiKey)

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			if attempt < maxRetries {
				continue
			}
			return ""
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < maxRetries {
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
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
			return ""
		}
		if len(apiResp.Choices) == 0 {
			return ""
		}

		content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
		if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
			content = content[1 : len(content)-1]
		}
		return content
	}

	return ""
}

// ========== 后台自动轮询：扫描未翻译记录并自动触发翻译 ==========

func init() {
	go cleanupAutoTranslateTasks()
	go startAutoTranslatePoller()
}

// startAutoTranslatePoller 每 5 分钟扫描一次未翻译记录，自动触发翻译
func startAutoTranslatePoller() {
	// 首次启动延迟 1 分钟，给服务启动留出时间
	time.Sleep(1 * time.Minute)
	for {
		pollAndAutoTranslate()
		time.Sleep(5 * time.Minute)
	}
}

func pollAndAutoTranslate() {
	defer func() {
		if r := recover(); r != nil {
			common.SysLog(fmt.Sprintf("AutoTranslate poller panic: %v", r))
		}
	}()

	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		return // 翻译 AI 未配置，跳过
	}

	const batchSize = 20

	// 扫描未翻译的 Prompts
	var promptIDs []int
	err := model.DB.Model(&model.Prompt{}).
		Select("id").
		Where("is_translated = ? AND (translation_error = ? OR translation_error IS NULL)", false, "").
		Limit(batchSize).
		Order("id desc").
		Pluck("id", &promptIDs).Error
	if err != nil {
		common.SysLog("AutoTranslate poller query prompts error: " + err.Error())
	}
	for _, id := range promptIDs {
		StartAutoTranslate("prompt", id)
		time.Sleep(2 * time.Second) // 间隔避免并发过高
	}
	if len(promptIDs) > 0 {
		common.SysLog(fmt.Sprintf("AutoTranslate poller: triggered %d prompts", len(promptIDs)))
	}

	// 扫描未翻译的 Articles
	var articleIDs []int
	err = model.DB.Model(&model.Article{}).
		Select("id").
		Where("is_translated = ? AND (translation_error = ? OR translation_error IS NULL)", false, "").
		Limit(batchSize).
		Order("id desc").
		Pluck("id", &articleIDs).Error
	if err != nil {
		common.SysLog("AutoTranslate poller query articles error: " + err.Error())
	}
	for _, id := range articleIDs {
		StartAutoTranslate("article", id)
		time.Sleep(2 * time.Second)
	}
	if len(articleIDs) > 0 {
		common.SysLog(fmt.Sprintf("AutoTranslate poller: triggered %d articles", len(articleIDs)))
	}
}
