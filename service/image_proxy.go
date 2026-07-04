package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	imageProxyDir     = "./uploads/img"
	imageProxyMapFile = "./uploads/img/_index.json"
)

var (
	imageProxyMap   = make(map[string]string) // uuid -> upstream_url
	imageProxyMutex sync.RWMutex
	imageProxyOnce  sync.Once
)

func initImageProxy() {
	_ = os.MkdirAll(imageProxyDir, 0755)
	loadImageProxyMap()
	// periodic save every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			saveImageProxyMap()
		}
	}()
}

func ensureImageProxyInit() {
	imageProxyOnce.Do(initImageProxy)
}

func loadImageProxyMap() {
	data, err := os.ReadFile(imageProxyMapFile)
	if err != nil {
		return
	}
	_ = common.Unmarshal(data, &imageProxyMap)
}

func saveImageProxyMap() {
	imageProxyMutex.RLock()
	m := make(map[string]string, len(imageProxyMap))
	for k, v := range imageProxyMap {
		m[k] = v
	}
	imageProxyMutex.RUnlock()

	data, err := common.Marshal(m)
	if err != nil {
		return
	}
	_ = os.MkdirAll(imageProxyDir, 0755)
	_ = os.WriteFile(imageProxyMapFile, data, 0644)
}

// RegisterImageProxyURL registers an upstream image URL and returns a local proxy id.
func RegisterImageProxyURL(upstreamURL string) string {
	ensureImageProxyInit()
	id := uuid.New().String()
	imageProxyMutex.Lock()
	imageProxyMap[id] = upstreamURL
	imageProxyMutex.Unlock()
	saveImageProxyMap()
	return id
}

// GetImageProxyURL returns the upstream URL for a proxy id.
func GetImageProxyURL(id string) string {
	ensureImageProxyInit()
	imageProxyMutex.RLock()
	defer imageProxyMutex.RUnlock()
	return imageProxyMap[id]
}

// FetchAndCacheProxyImage fetches the image from upstream and caches it locally.
// Returns the local file path. If already cached, returns immediately.
func FetchAndCacheProxyImage(id string) (string, error) {
	ensureImageProxyInit()
	cachePath := filepath.Join(imageProxyDir, id+".png")
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	upstreamURL := GetImageProxyURL(id)
	if upstreamURL == "" {
		return "", fmt.Errorf("proxy id not found")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(upstreamURL)
	if err != nil {
		return "", fmt.Errorf("fetch upstream failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read upstream body failed: %w", err)
	}

	// Try to use correct extension from Content-Type
	ext := ".png"
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		switch {
		case strings.Contains(ct, "image/jpeg"):
			ext = ".jpg"
		case strings.Contains(ct, "image/webp"):
			ext = ".webp"
		case strings.Contains(ct, "image/gif"):
			ext = ".gif"
		}
	}
	cachePath = filepath.Join(imageProxyDir, id+ext)

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return "", fmt.Errorf("write cache failed: %w", err)
	}
	return cachePath, nil
}

// RewriteImageResponseWithProxyURLs reads an upstream image generation response,
// replaces temporary upstream image URLs with persistent local proxy URLs, and
// returns a new http.Response with the modified body.
func RewriteImageResponseWithProxyURLs(c *gin.Context, resp *http.Response) *http.Response {
	if resp == nil || resp.Body == nil {
		return resp
	}
	// Only rewrite JSON image responses (skip b64_json, stream, etc.)
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return resp
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return resp
	}

	var imgResp dto.ImageResponse
	if err := common.Unmarshal(body, &imgResp); err != nil {
		// Not a valid image response, restore body and return as-is
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	modified := false
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	baseURL := scheme + "://" + host

	for i := range imgResp.Data {
		if imgResp.Data[i].Url != "" && imgResp.Data[i].B64Json == "" {
			proxyID := RegisterImageProxyURL(imgResp.Data[i].Url)
			imgResp.Data[i].Url = baseURL + "/image-proxy/" + proxyID + ".png"
			modified = true
		}
	}

	if !modified {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	newBody, err := common.Marshal(imgResp)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}
	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
	return resp
}

// ExtractImageURLFromResponse extracts the first image URL from an OpenAI-
// compatible image response, if present.
func ExtractImageURLFromResponse(body []byte) string {
	var imgResp dto.ImageResponse
	if err := common.Unmarshal(body, &imgResp); err != nil {
		return ""
	}
	for _, item := range imgResp.Data {
		if item.Url != "" {
			return item.Url
		}
	}
	return ""
}
