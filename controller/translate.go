package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// DeepLXResponse DeepLX 返回结构
type DeepLXResponse struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

// BatchTranslate 批量翻译接口（Admin 权限）
func BatchTranslate(c *gin.Context) {
	var req BatchTranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	// 结果: { "EN": { "key1": "translated", "key2": "translated" }, ... }
	result := make(map[string]map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 串行翻译，避免频率过高被封
	semaphore := make(chan struct{}, 1)

	// 检查是否启用 AI 翻译
	cfg := operation_setting.GetTranslateSetting()
	useAI := cfg.TranslateAIEnabled && cfg.TranslateAIApiKey != "" && cfg.TranslateAIBaseURL != ""

	for _, lang := range req.TargetLangs {
		for _, item := range req.Items {
			if item.Text == "" {
				continue
			}
			wg.Add(1)
			semaphore <- struct{}{}
			go func(targetLang, key, text string) {
				defer wg.Done()
				defer func() { <-semaphore }()

				// 请求间隔，降低频率
				time.Sleep(500 * time.Millisecond)

				var translated string

				if useAI {
					translated = translateSingleWithAI(cfg, text, req.SourceLang, targetLang)
				}

				// AI 翻译失败或未启用时，回退到 DeepLX
				if translated == "" {
					translated = translateSingleWithDeepLX(text, req.SourceLang, targetLang)
				}

				mu.Lock()
				if result[targetLang] == nil {
					result[targetLang] = make(map[string]string)
				}
				result[targetLang][key] = translated
				mu.Unlock()
			}(lang, item.Key, item.Text)
		}
	}

	wg.Wait()
	common.ApiSuccess(c, result)
}

// translateSingleWithAI 使用 AI 模型翻译单条文本
func translateSingleWithAI(cfg *operation_setting.TranslateSetting, text, sourceLang, targetLang string) string {
	// 构建翻译 prompt
	prompt := fmt.Sprintf(
		"You are a professional translator. Translate the following text from %s to %s. "+
			"Return ONLY the translated text, without any explanation, quotes, or markdown formatting.\n\n"+
			"Text to translate:\n%s",
		sourceLang, targetLang, text,
	)

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

	client := &http.Client{Timeout: 30 * time.Second}
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

	translated := extractPlainText(apiResp.Choices[0].Message.Content)
	common.SysLog("AI translated: [" + sourceLang + "->" + targetLang + "] " + text + " -> " + translated)
	return translated
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

// translateSingleWithDeepLX 使用 DeepLX 翻译单条文本（回退方案）
func translateSingleWithDeepLX(text, sourceLang, targetLang string) string {
	deeplxURL := os.Getenv("DEEPLX_URL")
	if deeplxURL == "" {
		deeplxURL = "http://deeplx:1188/translate"
	}
	deeplxToken := os.Getenv("DEEPLX_TOKEN")

	// DeepLX 不支持 ZH-TW，映射为 ZH
	if targetLang == "ZH-TW" {
		targetLang = "ZH"
	}

	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(1 * time.Second)
		}
		translated := translateSingle(deeplxURL, deeplxToken, text, sourceLang, targetLang)
		if translated != "" && translated != text {
			return translated
		}
	}
	return ""
}

func translateSingle(url, token, text, sourceLang, targetLang string) string {
	payload := map[string]string{
		"text":        text,
		"source_lang": sourceLang,
		"target_lang": targetLang,
	}
	jsonBody, err := common.Marshal(payload)
	if err != nil {
		return ""
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result DeepLXResponse
	if err := common.DecodeJson(resp.Body, &result); err != nil {
		common.SysLog("DeepLX decode error: " + err.Error())
		return ""
	}
	if result.Code != 200 {
		common.SysLog("DeepLX non-200 response: code=" + strconv.Itoa(result.Code) + " msg=" + result.Message)
		return ""
	}

	// 处理两种返回格式:
	// 1. {"code":200,"data":"翻译文本"}
	// 2. {"code":200,"data":{"text":"翻译文本","alternatives":[]}}
	var strResult string
	if err := common.Unmarshal(result.Data, &strResult); err == nil && strResult != "" {
		common.SysLog("DeepLX translated: [" + sourceLang + "->" + targetLang + "] " + text + " -> " + strResult)
		return strResult
	}
	var objResult struct {
		Text string `json:"text"`
	}
	if err := common.Unmarshal(result.Data, &objResult); err == nil {
		common.SysLog("DeepLX translated: [" + sourceLang + "->" + targetLang + "] " + text + " -> " + objResult.Text)
		return objResult.Text
	}
	common.SysLog("DeepLX unknown data format: [" + sourceLang + "->" + targetLang + "] " + string(result.Data))
	return ""
}
