package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const articleImageAITimeout = 120 * time.Second

// ImageGenerationResult AI 生成的图片结果
type ImageGenerationResult struct {
	URLs []string `json:"urls"`
}

// GenerateImagesForArticle 调用 AI 为文章生成图片
func GenerateImagesForArticle(prompt string, n int, size string) (*ImageGenerationResult, error) {
	cfg := operation_setting.GetImageAISetting()
	if !cfg.ImageAIEnabled || cfg.ImageAIApiKey == "" || cfg.ImageAIBaseURL == "" {
		return nil, fmt.Errorf("image ai not configured")
	}

	if n <= 0 {
		n = cfg.ImageAIN
	}
	if n <= 0 {
		n = 1
	}
	if size == "" {
		size = cfg.ImageAISize
	}
	if size == "" {
		size = "1024x1024"
	}

	model := cfg.ImageAIModel
	if model == "" {
		model = "dall-e-3"
	}

	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"n":      n,
		"size":   size,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: articleImageAITimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.ImageAIBaseURL+"/v1/images/generations", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ImageAIApiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai api returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Data []struct {
			Url           string `json:"url"`
			B64Json       string `json:"b64_json"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := common.DecodeJson(resp.Body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse ai response failed: %w", err)
	}

	var urls []string
	for _, item := range apiResp.Data {
		if item.Url != "" {
			urls = append(urls, item.Url)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("ai returned no images")
	}

	return &ImageGenerationResult{URLs: urls}, nil
}
