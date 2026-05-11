package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
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

	// 串行翻译，避免 DeepL 免费接口封 IP
	// semaphore=1 保证只有一个请求在进行，每次请求后间隔 500ms
	semaphore := make(chan struct{}, 1)

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

				// DeepLX 不支持 ZH-TW，映射为 ZH
				dlxTargetLang := targetLang
				if dlxTargetLang == "ZH-TW" {
					dlxTargetLang = "ZH"
				}

				var translated string
				for attempt := 0; attempt < 2; attempt++ {
					if attempt > 0 {
						time.Sleep(1 * time.Second)
					}
					translated = translateSingle(deeplxURL, deeplxToken, text, req.SourceLang, dlxTargetLang)
					if translated != "" && translated != text {
						break
					}
					// 如果返回空或原文，可能是限流，等待后重试
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
