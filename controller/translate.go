package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// BatchTranslateRequest 批量翻译请求
type BatchTranslateRequest struct {
	Items       []TranslateItem `json:"items" binding:"required"`
	SourceLang  string          `json:"source_lang" binding:"required"`
	TargetLangs []string        `json:"target_langs" binding:"required"`
}

// TranslateItem 单条翻译项
type TranslateItem struct {
	Key  string `json:"key" binding:"required"`
	Text string `json:"text" binding:"required"`
}

// BatchTranslate 批量翻译接口（Admin 权限）
func BatchTranslate(c *gin.Context) {
	common.SysLog("BatchTranslate: received request")
	var req BatchTranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.SysLog("BatchTranslate: bind error: " + err.Error())
		common.ApiError(c, err)
		return
	}
	common.SysLog(fmt.Sprintf("BatchTranslate: items=%d langs=%v", len(req.Items), req.TargetLangs))

	// 结果: { "EN": { "key1": "translated", "key2": "translated" }, ... }
	result := make(map[string]map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 串行翻译，避免 DeepSeek 等慢模型超时
	semaphore := make(chan struct{}, 1)

	// 检查是否启用 AI 翻译
	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		common.ApiErrorMsg(c, "AI translation not configured")
		return
	}

	// 按语言批量翻译：每种语言只发 1 次请求，把多个字段一起翻译
	for _, lang := range req.TargetLangs {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(targetLang string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// 短暂间隔，避免瞬间打满
			time.Sleep(100 * time.Millisecond)

			translations := translateBatchWithAI(cfg, req.Items, req.SourceLang, targetLang)

			mu.Lock()
			result[targetLang] = translations
			mu.Unlock()
		}(lang)
	}

	wg.Wait()
	common.ApiSuccess(c, result)
}

// langCodeToName 将语言代码映射为 AI 易于理解的完整语言名称
var langCodeToName = map[string]string{
	"zh": "Chinese", "en": "English", "fr": "French", "ru": "Russian",
	"ja": "Japanese", "vi": "Vietnamese", "ko": "Korean", "es": "Spanish",
	"de": "German", "pt": "Portuguese", "it": "Italian", "ar": "Arabic",
}

func getLangName(code string) string {
	if name, ok := langCodeToName[code]; ok {
		return name
	}
	// 兼容大小写（如 ZH、EN）
	if name, ok := langCodeToName[strings.ToLower(code)]; ok {
		return name
	}
	return code
}

// translateBatchWithAI 批量翻译：一次请求翻译某语言的多个字段
// 若 content 字段过长，会拆分出来单独翻译，避免 AI 因 token 限制截断长文本
func translateBatchWithAI(cfg *operation_setting.TranslateSetting, items []TranslateItem, sourceLang, targetLang string) map[string]string {
	result := make(map[string]string)

	sourceLangName := getLangName(sourceLang)
	targetLangName := getLangName(targetLang)

	// 若 content 过长，先拆出来单独翻译
	var contentItem *TranslateItem
	var shortItems []TranslateItem
	for i := range items {
		if items[i].Key == "content" && len(items[i].Text) > 300 {
			contentItem = &items[i]
		} else {
			shortItems = append(shortItems, items[i])
		}
	}

	// 读取 batch-translate Skill 模板，不存在则使用硬编码默认值
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

	// 构建 {{items}} 变量内容（旧文本格式，向后兼容）
	var itemsBuilder strings.Builder
	for _, item := range shortItems {
		if item.Text != "" {
			itemsBuilder.WriteString(fmt.Sprintf("%s: %s\n", item.Key, item.Text))
		}
	}

	// 构建 {{fields}} 变量内容（新 JSON 格式）
	fieldsMap := make(map[string]string)
	for _, item := range shortItems {
		if item.Text != "" {
			fieldsMap[item.Key] = item.Text
		}
	}
	fieldsJSON, _ := common.Marshal(fieldsMap)

	systemPrompt = strings.ReplaceAll(systemPrompt, "{{sourceLang}}", sourceLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{targetLang}}", targetLangName)
	// 兼容旧模板中使用 {{language}} 的写法
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{language}}", targetLangName)
	userPrompt := strings.ReplaceAll(userPromptTemplate, "{{sourceLang}}", sourceLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{targetLang}}", targetLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{items}}", itemsBuilder.String())
	userPrompt = strings.ReplaceAll(userPrompt, "{{fields}}", string(fieldsJSON))

	response := callTranslateAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		for _, item := range items {
			result[item.Key] = item.Text
		}
		return result
	}

	// 去掉可能的 markdown 代码块
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

	// 先尝试 JSON 解析（新格式）
	var jsonResult map[string]string
	if err := common.Unmarshal([]byte(response), &jsonResult); err == nil && len(jsonResult) > 0 {
		for k, v := range jsonResult {
			result[k] = v
		}
	} else {
		// 回退到 "key: translated text" 格式（旧格式，向后兼容）
		// 支持值跨多行：如果一行没有 ":"，则追加到上一个 key
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
				// 追加到当前 key（多行值）
				result[currentKey] += "\n" + line
			}
		}
	}

	// 若 content 被拆分出来，单独翻译（避免长文本被截断）
	if contentItem != nil {
		common.SysLog(fmt.Sprintf("AI translate content separately: [%s->%s] len=%d", sourceLang, targetLang, len(contentItem.Text)))
		translatedContent := translateSingleWithAI(cfg, contentItem.Text, sourceLang, targetLang)
		if translatedContent != "" && translatedContent != contentItem.Text {
			result[contentItem.Key] = translatedContent
		} else {
			result[contentItem.Key] = contentItem.Text
		}
	}

	missingKeys := []string{}
	for _, item := range items {
		if result[item.Key] == "" || result[item.Key] == item.Text {
			// 字段缺失或仍是原文，尝试单条补翻
			translated := translateSingleWithAI(cfg, item.Text, sourceLang, targetLang)
			if translated != "" && translated != item.Text {
				result[item.Key] = translated
				missingKeys = append(missingKeys, item.Key)
			} else {
				// 单条也失败了，用强制 prompt 再试一次
				forced := translateSingleWithAIForced(cfg, item.Text, sourceLang, targetLang)
				if forced != "" && forced != item.Text {
					result[item.Key] = forced
					missingKeys = append(missingKeys, item.Key+"(forced)")
				} else {
					result[item.Key] = item.Text
				}
			}
		}
	}

	if len(missingKeys) > 0 {
		common.SysLog(fmt.Sprintf("AI batch fallback translated: [%s->%s] keys=%v", sourceLang, targetLang, missingKeys))
	}
	common.SysLog(fmt.Sprintf("AI batch translated: [%s->%s] keys=%v", sourceLang, targetLang, len(result)))
	return result
}

// translateSingleWithAI 使用 AI 模型翻译单条文本
func translateSingleWithAI(cfg *operation_setting.TranslateSetting, text, sourceLang, targetLang string) string {
	sourceLangName := getLangName(sourceLang)
	targetLangName := getLangName(targetLang)

	// 读取 prompt-translate Skill 模板
	systemPrompt := ""
	userPromptTemplate := ""
	if skill, err := model.GetSkillBySkillId("prompt-translate"); err == nil && skill.SystemPromptTemplate != "" {
		systemPrompt = skill.SystemPromptTemplate
		userPromptTemplate = skill.UserPromptTemplate
	}
	if systemPrompt == "" {
		systemPrompt = "You are a professional translator. Your ONLY task is to translate text. You MUST respond entirely in {{targetLang}}. Do NOT respond in {{sourceLang}} or any other language. Do not add explanations, notes, or the original text — output ONLY the translated text in {{targetLang}}. If you output anything in {{sourceLang}}, you have failed."
	}
	if userPromptTemplate == "" {
		userPromptTemplate = "Translate the following text from {{sourceLang}} to {{targetLang}}. Your response must be ONLY the translated text in {{targetLang}}, nothing else. Do NOT include the original {{sourceLang}} text. Do NOT explain. Output pure {{targetLang}} text only:\n\n\"\"\"\n{{prompt}}\n\"\"\""
	}

	systemPrompt = strings.ReplaceAll(systemPrompt, "{{sourceLang}}", sourceLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{targetLang}}", targetLangName)
	// 兼容旧模板中使用 {{language}} 的写法
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{language}}", targetLangName)
	userPrompt := strings.ReplaceAll(userPromptTemplate, "{{sourceLang}}", sourceLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{targetLang}}", targetLangName)
	// 兼容旧模板中使用 {{language}} 的写法
	userPrompt = strings.ReplaceAll(userPrompt, "{{language}}", targetLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{prompt}}", text)

	return callTranslateAI(cfg, systemPrompt, userPrompt)
}

// translateSingleWithAIForced 强制翻译：使用更强烈的 prompt 确保模型用目标语言回复
func translateSingleWithAIForced(cfg *operation_setting.TranslateSetting, text, sourceLang, targetLang string) string {
	targetLangName := getLangName(targetLang)
	sourceLangName := getLangName(sourceLang)

	systemPrompt := "You are a translator. CRITICAL RULE: Your entire response MUST be written in " + targetLangName + ". ZERO words in " + sourceLangName + " allowed. If you write even one word in " + sourceLangName + ", you failed. Output ONLY the translation."
	userPrompt := "Translate this to " + targetLangName + ". Remember: ONLY " + targetLangName + " output. No " + sourceLangName + " at all:\n\n" + text

	return callTranslateAI(cfg, systemPrompt, userPrompt)
}

// callTranslateAI 调用 AI 模型获取文本响应（带重试）
func callTranslateAI(cfg *operation_setting.TranslateSetting, systemPrompt, userPrompt string) string {
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
		common.SysLog("AI translate marshal error: " + err.Error())
		return ""
	}

	common.SysLog(fmt.Sprintf("AI translate request: model=%s baseURL=%s", cfg.TranslateAIModel, cfg.TranslateAIBaseURL))

	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			common.SysLog(fmt.Sprintf("AI translate retry attempt %d/%d", attempt, maxRetries))
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // 指数退避
		}

		client := &http.Client{Timeout: 120 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 125*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TranslateAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			cancel()
			common.SysLog("AI translate request error: " + err.Error())
			return ""
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.TranslateAIApiKey)

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			common.SysLog("AI translate do error (attempt " + strconv.Itoa(attempt+1) + "): " + err.Error())
			// 超时或网络错误时重试
			if attempt < maxRetries {
				continue
			}
			return ""
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable {
			common.SysLog(fmt.Sprintf("AI translate server error %d (attempt %d), will retry", resp.StatusCode, attempt+1))
			if attempt < maxRetries {
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			common.SysLog(fmt.Sprintf("AI translate non-200: %d, body: %s", resp.StatusCode, string(bodyBytes)))
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
			common.SysLog("AI translate decode error: " + err.Error())
			return ""
		}

		if len(apiResp.Choices) == 0 {
			common.SysLog("AI translate empty choices")
			return ""
		}

		rawContent := apiResp.Choices[0].Message.Content
		common.SysLog(fmt.Sprintf("AI translate success (attempt %d)", attempt+1))
		return extractPlainText(rawContent)
	}

	return ""
}

// extractPlainText 去除 AI 返回内容中可能的引号、markdown 等格式
func extractPlainText(content string) string {
	content = strings.TrimSpace(content)
	// 去除首尾的双引号
	if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
		content = content[1 : len(content)-1]
	}
	// 去除首尾的单反引号，但保留 markdown 代码块 (```)
	if len(content) >= 3 && content[0] == '`' && content[len(content)-1] == '`' && !strings.HasPrefix(content, "```") {
		content = content[1 : len(content)-1]
	}
	return strings.TrimSpace(content)
}

// ============== Translate Queue ==============

type TranslateQueue struct {
	ID        string                       `json:"id"`
	Status    string                       `json:"status"` // running, done, failed
	Progress  QueueProgress                `json:"progress"`
	Results   map[string]map[string]string `json:"results"`
	Error     string                       `json:"error"`
	CreatedAt time.Time                    `json:"created_at"`
}

type QueueProgress struct {
	Current     int    `json:"current"`
	Total       int    `json:"total"`
	CurrentLang string `json:"current_lang"`
}

var (
	translateQueueMap = make(map[string]*TranslateQueue)
	queueMutex        sync.RWMutex
)

// StartTranslateQueue 启动异步翻译队列
func StartTranslateQueue(c *gin.Context) {
	var req BatchTranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	queueID := fmt.Sprintf("tq_%d", time.Now().UnixNano())
	queue := &TranslateQueue{
		ID:        queueID,
		Status:    "running",
		Progress:  QueueProgress{Current: 0, Total: len(req.TargetLangs)},
		Results:   make(map[string]map[string]string),
		CreatedAt: time.Now(),
	}

	queueMutex.Lock()
	translateQueueMap[queueID] = queue
	queueMutex.Unlock()

	go executeTranslateQueue(queue, req.Items, req.SourceLang, req.TargetLangs)

	common.SysLog(fmt.Sprintf("[TranslateQueue] started: %s langs=%v", queueID, req.TargetLangs))
	common.ApiSuccess(c, gin.H{"queue_id": queueID})
}

// GetTranslateQueue 查询翻译队列状态
func GetTranslateQueue(c *gin.Context) {
	queueID := c.Param("id")
	queueMutex.RLock()
	queue, ok := translateQueueMap[queueID]
	queueMutex.RUnlock()

	if !ok {
		common.ApiErrorMsg(c, "queue not found or expired")
		return
	}

	common.ApiSuccess(c, queue)
}

func executeTranslateQueue(queue *TranslateQueue, items []TranslateItem, sourceLang string, targetLangs []string) {
	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		queueMutex.Lock()
		queue.Status = "failed"
		queue.Error = "AI translation not configured"
		queueMutex.Unlock()
		return
	}

	for i, lang := range targetLangs {
		queueMutex.Lock()
		queue.Progress.Current = i + 1
		queue.Progress.CurrentLang = getLangName(lang)
		queueMutex.Unlock()

		common.SysLog(fmt.Sprintf("[TranslateQueue] %s translating %s (%d/%d)", queue.ID, lang, i+1, len(targetLangs)))
		translations := translateBatchWithAI(cfg, items, sourceLang, lang)
		if len(translations) > 0 {
			queueMutex.Lock()
			queue.Results[lang] = translations
			queueMutex.Unlock()
			common.SysLog(fmt.Sprintf("[TranslateQueue] %s %s done keys=%v", queue.ID, lang, len(translations)))
		} else {
			queueMutex.Lock()
			queue.Status = "failed"
			queue.Error = fmt.Sprintf("translation failed for %s", getLangName(lang))
			queueMutex.Unlock()
			common.SysLog(fmt.Sprintf("[TranslateQueue] %s %s failed", queue.ID, lang))
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	queueMutex.Lock()
	queue.Status = "done"
	queueMutex.Unlock()
	common.SysLog(fmt.Sprintf("[TranslateQueue] %s completed", queue.ID))
}

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			queueMutex.Lock()
			for id, q := range translateQueueMap {
				if time.Since(q.CreatedAt) > 10*time.Minute {
					delete(translateQueueMap, id)
				}
			}
			queueMutex.Unlock()
		}
	}()
}
