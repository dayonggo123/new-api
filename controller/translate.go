package controller

import (
	"bytes"
	"context"
	"fmt"
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

	// 并发翻译，提高速度避免网关超时
	semaphore := make(chan struct{}, 3)

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
	return code
}

// translateBatchWithAI 批量翻译：一次请求翻译某语言的多个字段
func translateBatchWithAI(cfg *operation_setting.TranslateSetting, items []TranslateItem, sourceLang, targetLang string) map[string]string {
	result := make(map[string]string)

	sourceLangName := getLangName(sourceLang)
	targetLangName := getLangName(targetLang)

	// 读取 batch-translate Skill 模板，不存在则使用硬编码默认值
	systemPrompt := ""
	userPromptTemplate := ""
	if skill, err := model.GetSkillBySkillId("batch-translate"); err == nil && skill.SystemPromptTemplate != "" {
		systemPrompt = skill.SystemPromptTemplate
		userPromptTemplate = skill.UserPromptTemplate
	}
	if systemPrompt == "" {
		systemPrompt = "You are a professional translator. Your ONLY task is to translate the given fields from {{sourceLang}} to {{targetLang}}. You MUST respond entirely in {{targetLang}}. Do NOT respond in {{sourceLang}} or any other language. Preserve all variable placeholders like {{variableName}} exactly as-is. Return results in valid JSON format with the exact same keys as the input. No explanations, no markdown code blocks around the JSON."
	}
	if userPromptTemplate == "" {
		userPromptTemplate = "Translate the following fields from {{sourceLang}} to {{targetLang}}.\n\nInput (JSON):\n{{fields}}\n\nRules:\n1. Return ONLY a JSON object with the same keys\n2. All values must be pure {{targetLang}} text\n3. Preserve {{variables}} exactly as-is\n4. Do not add explanations or wrap in markdown\n\nOutput format example:\n{\"title\":\"Translated Title\",\"content\":\"Translated content...\"}"
	}

	// 构建 {{items}} 变量内容（旧文本格式，向后兼容）
	var itemsBuilder strings.Builder
	for _, item := range items {
		if item.Text != "" {
			itemsBuilder.WriteString(fmt.Sprintf("%s: %s\n", item.Key, item.Text))
		}
	}

	// 构建 {{fields}} 变量内容（新 JSON 格式）
	fieldsMap := make(map[string]string)
	for _, item := range items {
		if item.Text != "" {
			fieldsMap[item.Key] = item.Text
		}
	}
	fieldsJSON, _ := common.Marshal(fieldsMap)

	systemPrompt = strings.ReplaceAll(systemPrompt, "{{sourceLang}}", sourceLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{targetLang}}", targetLangName)
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
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !strings.Contains(line, ":") {
				continue
			}
			idx := strings.Index(line, ": ")
			if idx == -1 {
				idx = strings.Index(line, ":")
			}
			if idx <= 0 {
				continue
			}
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			if key != "" {
				result[key] = val
			}
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
				result[item.Key] = item.Text
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
		systemPrompt = "You are a professional translator. Your ONLY task is to translate text. You MUST respond entirely in {{targetLang}}. Do NOT respond in {{sourceLang}} or any other language. Do not add explanations, notes, or the original text — output ONLY the translated text in {{targetLang}}."
	}
	if userPromptTemplate == "" {
		userPromptTemplate = "Translate the following text from {{sourceLang}} to {{targetLang}}. Your response must be ONLY the translated text in {{targetLang}}, nothing else:\n\n\"\"\"\n{{prompt}}\n\"\"\""
	}

	systemPrompt = strings.ReplaceAll(systemPrompt, "{{sourceLang}}", sourceLangName)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{targetLang}}", targetLangName)
	userPrompt := strings.ReplaceAll(userPromptTemplate, "{{sourceLang}}", sourceLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{targetLang}}", targetLangName)
	userPrompt = strings.ReplaceAll(userPrompt, "{{prompt}}", text)

	return callTranslateAI(cfg, systemPrompt, userPrompt)
}

// callTranslateAI 调用 AI 模型获取文本响应
func callTranslateAI(cfg *operation_setting.TranslateSetting, systemPrompt, userPrompt string) string {
	reqBody := map[string]interface{}{
		"model": cfg.TranslateAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  4096,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		common.SysLog("AI translate marshal error: " + err.Error())
		return ""
	}

	common.SysLog(fmt.Sprintf("AI translate request: model=%s baseURL=%s", cfg.TranslateAIModel, cfg.TranslateAIBaseURL))
	common.SysLog(fmt.Sprintf("AI translate system prompt: %s", systemPrompt))
	common.SysLog(fmt.Sprintf("AI translate user prompt: %s", userPrompt))

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.TranslateAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		common.SysLog("AI translate request error: " + err.Error())
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.TranslateAIApiKey)

	resp, err := client.Do(req)
	if err != nil {
		common.SysLog("AI translate do error: " + err.Error())
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		common.SysLog("AI translate non-200: " + strconv.Itoa(resp.StatusCode))
		return ""
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := common.DecodeJson(resp.Body, &apiResp); err != nil {
		common.SysLog("AI translate decode error: " + err.Error())
		return ""
	}

	if len(apiResp.Choices) == 0 {
		common.SysLog("AI translate empty choices")
		return ""
	}

	rawContent := apiResp.Choices[0].Message.Content
	common.SysLog(fmt.Sprintf("AI translate raw response: %s", rawContent))
	return extractPlainText(rawContent)
}

// extractPlainText 去除 AI 返回内容中可能的引号、markdown 等格式
func extractPlainText(content string) string {
	content = strings.TrimSpace(content)
	// 去除首尾的双引号
	if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
		content = content[1 : len(content)-1]
	}
	// 去除首尾的代码块标记
	if len(content) >= 3 && content[0] == '`' && content[len(content)-1] == '`' {
		content = content[1 : len(content)-1]
	}
	return strings.TrimSpace(content)
}

