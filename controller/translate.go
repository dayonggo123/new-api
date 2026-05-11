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
	semaphore := make(chan struct{}, 10)

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

// translateBatchWithAI 批量翻译：一次请求翻译某语言的多个字段
func translateBatchWithAI(cfg *operation_setting.TranslateSetting, items []TranslateItem, sourceLang, targetLang string) map[string]string {
	result := make(map[string]string)

	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf(
		"You are a professional translator. Translate the following items from %s to %s.\n",
		sourceLang, targetLang,
	))
	promptBuilder.WriteString("Return the translations in this exact format, one per line, with the key followed by a colon and a space, then the translated text.\n")
	promptBuilder.WriteString("Do not add any extra text, explanations, or blank lines.\n\n")
	for _, item := range items {
		if item.Text != "" {
			promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", item.Key, item.Text))
		}
	}

	response := callTranslateAI(cfg, promptBuilder.String())
	if response == "" {
		// AI 调用失败，所有字段 fallback 到原文
		for _, item := range items {
			result[item.Key] = item.Text
		}
		return result
	}

	// 解析 "key: translated text" 格式
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
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

	// fallback：未翻译成功的字段使用原文
	for _, item := range items {
		if result[item.Key] == "" {
			result[item.Key] = item.Text
		}
	}

	common.SysLog(fmt.Sprintf("AI batch translated: [%s->%s] keys=%v", sourceLang, targetLang, len(result)))
	return result
}

// translateSingleWithAI 使用 AI 模型翻译单条文本
func translateSingleWithAI(cfg *operation_setting.TranslateSetting, text, sourceLang, targetLang string) string {
	prompt := fmt.Sprintf(
		"You are a professional translator. Translate the following text from %s to %s. "+
			"Return ONLY the translated text, without any explanation, quotes, or markdown formatting.\n\n"+
			"Text to translate:\n%s",
		sourceLang, targetLang, text,
	)
	return callTranslateAI(cfg, prompt)
}

// callTranslateAI 调用 AI 模型获取文本响应
func callTranslateAI(cfg *operation_setting.TranslateSetting, prompt string) string {
	reqBody := map[string]interface{}{
		"model": cfg.TranslateAIModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  4096,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		common.SysLog("AI translate marshal error: " + err.Error())
		return ""
	}

	client := &http.Client{Timeout: 15 * time.Second}
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

	return extractPlainText(apiResp.Choices[0].Message.Content)
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

