package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const articleImageAITimeout = 120 * time.Second

// ImageGenerationResult AI 生成的图片结果
type ImageGenerationResult struct {
	URLs []string `json:"urls"`
}

// GenerateImagesForArticle 调用 AI 为文章生成图片，下载后存入数据库，返回本地持久化 URL
func GenerateImagesForArticle(prompt string, n int, size string, baseURL string) (*ImageGenerationResult, error) {
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

	modelName := cfg.ImageAIModel
	if modelName == "" {
		modelName = "dall-e-3"
	}

	reqBody := map[string]interface{}{
		"model":  modelName,
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
		var imageData []byte
		var mimeType string

		// Prefer b64_json if available (already in memory, no download needed)
		if item.B64Json != "" {
			imageData, err = base64.StdEncoding.DecodeString(item.B64Json)
			if err == nil {
				mimeType = http.DetectContentType(imageData)
			}
		}

		// Fallback: download from URL
		if imageData == nil && item.Url != "" {
			imageData, mimeType, err = downloadImage(item.Url)
			if err != nil {
				common.SysLog(fmt.Sprintf("failed to download ai image: %v", err))
				continue
			}
		}

		if imageData == nil {
			continue
		}

		// Save to database
		am := &model.ArticleMedia{
			MediaType:   "content_image",
			MimeType:    mimeType,
			Data:        base64.StdEncoding.EncodeToString(imageData),
			CreatedTime: common.GetTimestamp(),
		}
		if err := am.Insert(); err != nil {
			common.SysLog(fmt.Sprintf("failed to save article media: %v", err))
			continue
		}

		localURL := fmt.Sprintf("%s/api/public/article-media/%d", baseURL, am.Id)
		urls = append(urls, localURL)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("ai returned no images or all downloads failed")
	}

	return &ImageGenerationResult{URLs: urls}, nil
}

func downloadImage(url string) ([]byte, string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, nil
}
