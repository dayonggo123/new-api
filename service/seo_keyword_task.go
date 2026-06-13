package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	seoKeywordUpdateTickInterval = 24 * time.Hour // 每天更新一次
	seoKeywordUpdateBatchSize    = 50             // 每批处理数量
	seoKeywordUpdateTimeout      = 30 * time.Second
	seoKeywordMaxPerPrompt       = 10             // 每个提示词最多保存的关键词数
)

var (
	seoKeywordUpdateOnce    sync.Once
	seoKeywordUpdateRunning atomic.Bool
)

// GoogleSuggestResponse Google Suggest API 返回的 XML 结构
type GoogleSuggestResponse struct {
	XMLName             xml.Name              `xml:"toplevel"`
	CompleteSuggestions []CompleteSuggestion  `xml:"CompleteSuggestion"`
}

type CompleteSuggestion struct {
	Suggestion Suggestion `xml:"suggestion"`
}

type Suggestion struct {
	Data string `xml:"data,attr"`
}

// StartSEOKeywordUpdateTask 启动 SEO 关键词定时更新任务
func StartSEOKeywordUpdateTask() {
	seoKeywordUpdateOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("seo keyword update task started: tick=%s", seoKeywordUpdateTickInterval))

			ticker := time.NewTicker(seoKeywordUpdateTickInterval)
			defer ticker.Stop()

			// 启动时立即执行一次
			runSEOKeywordUpdateOnce()
			for range ticker.C {
				runSEOKeywordUpdateOnce()
			}
		})
	})
}

func runSEOKeywordUpdateOnce() {
	if !seoKeywordUpdateRunning.CompareAndSwap(false, true) {
		return
	}
	defer seoKeywordUpdateRunning.Store(false)

	ctx := context.Background()

	// 获取所有启用的提示词
	prompts, _, err := model.GetPublicPrompts(0, "", 0, 10000, "id", "asc")
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("seo keyword update: fetch prompts failed: %v", err))
		return
	}

	if len(prompts) == 0 {
		return
	}

	updated := 0
	failed := 0

	for _, p := range prompts {
		if p == nil {
			continue
		}

		keywords, err := fetchGoogleSuggestions(ctx, buildSearchQuery(p.Prompt))
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("seo keyword update: prompt_id=%d fetch failed: %v", p.Id, err))
			failed++
			continue
		}

		if len(keywords) == 0 {
			continue
		}

		// 限制数量并去重
		var filtered []string
		seen := make(map[string]struct{})
		for _, kw := range keywords {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			if _, ok := seen[kw]; ok {
				continue
			}
			seen[kw] = struct{}{}
			filtered = append(filtered, kw)
			if len(filtered) >= seoKeywordMaxPerPrompt {
				break
			}
		}

		if len(filtered) == 0 {
			continue
		}

		seoKeywords := strings.Join(filtered, ", ")
		if err := model.DB.Model(&model.Prompt{}).Where("id = ?", p.Id).Update("seo_keywords", seoKeywords).Error; err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("seo keyword update: prompt_id=%d save failed: %v", p.Id, err))
			failed++
			continue
		}

		updated++
		time.Sleep(500 * time.Millisecond) // 避免请求过快被限制
	}

	logger.LogInfo(ctx, fmt.Sprintf("seo keyword update completed: total=%d updated=%d failed=%d", len(prompts), updated, failed))
}

// buildSearchQuery 根据提示词构建用于 Google Suggest 的搜索查询
func buildSearchQuery(prompt *model.Prompt) string {
	var parts []string

	// 优先使用标题
	if prompt.Title != "" {
		parts = append(parts, prompt.Title)
	}

	// 添加模型名称
	if prompt.Model != "" {
		parts = append(parts, prompt.Model)
	}

	// 从标签中提取
	if prompt.Tags != "" {
		var tags []string
		_ = common.Unmarshal([]byte(prompt.Tags), &tags)
		if len(tags) > 0 {
			parts = append(parts, tags[0])
		}
	}

	query := strings.Join(parts, " ")
	if query == "" {
		query = "AI prompt"
	}

	return query
}

// fetchGoogleSuggestions 调用 Google Suggest API 获取相关搜索建议
func fetchGoogleSuggestions(ctx context.Context, query string) ([]string, error) {
	u := "https://suggestqueries.google.com/complete/search?output=toolbar&hl=zh-CN&q=" + url.QueryEscape(query)

	client := &http.Client{Timeout: seoKeywordUpdateTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头模拟浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google suggest api returned status %d", resp.StatusCode)
	}

	var gs GoogleSuggestResponse
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&gs); err != nil {
		return nil, fmt.Errorf("parse google suggest response failed: %w", err)
	}

	var suggestions []string
	for _, cs := range gs.CompleteSuggestions {
		if cs.Suggestion.Data != "" {
			suggestions = append(suggestions, cs.Suggestion.Data)
		}
	}

	return suggestions, nil
}
