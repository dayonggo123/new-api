package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
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

	deeplxURL := os.Getenv("DEEPLX_URL")
	if deeplxURL == "" {
		deeplxURL = "http://deeplx:1188/translate"
	}
	deeplxToken := os.Getenv("DEEPLX_TOKEN")

	// 结果: { "EN": { "key1": "translated", "key2": "translated" }, ... }
	result := make(map[string]map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 控制并发数，避免 DeepLX 限流
	semaphore := make(chan struct{}, 5)

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

				translated := translateSingle(deeplxURL, deeplxToken, text, req.SourceLang, targetLang)

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

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result DeepLXResponse
	if err := common.DecodeJson(resp.Body, &result); err != nil {
		return ""
	}
	if result.Code != 200 {
		return ""
	}

	// 处理两种返回格式:
	// 1. {"code":200,"data":"翻译文本"}
	// 2. {"code":200,"data":{"text":"翻译文本","alternatives":[]}}
	var strResult string
	if err := common.Unmarshal(result.Data, &strResult); err == nil && strResult != "" {
		return strResult
	}
	var objResult struct {
		Text string `json:"text"`
	}
	if err := common.Unmarshal(result.Data, &objResult); err == nil {
		return objResult.Text
	}
	return ""
}
