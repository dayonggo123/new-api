package service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
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
