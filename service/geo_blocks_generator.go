package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// ========== 任务状态定义 ==========

type GeoBlocksTaskStatus string

const (
	GeoBlocksTaskStatusPending   GeoBlocksTaskStatus = "pending"
	GeoBlocksTaskStatusRunning   GeoBlocksTaskStatus = "running"
	GeoBlocksTaskStatusCompleted GeoBlocksTaskStatus = "completed"
	GeoBlocksTaskStatusFailed    GeoBlocksTaskStatus = "failed"
)

type GeoBlocksTaskItem struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type GeoBlocksTask struct {
	ID        string              `json:"id"`
	Type      string              `json:"type"` // "prompt" or "article"
	Status    GeoBlocksTaskStatus `json:"status"`
	Total     int                 `json:"total"`
	Completed int                 `json:"completed"`
	Failed    int                 `json:"failed"`
	Items     []GeoBlocksTaskItem `json:"items"`
	CreatedAt time.Time           `json:"created_at"`
}

// ========== 内存任务管理 ==========

var (
	geoBlocksTasks   = make(map[string]*GeoBlocksTask)
	geoBlocksTaskMu  sync.RWMutex
	geoBlocksTaskTTL = 2 * time.Hour
)

func init() {
	go cleanupGeoBlocksTasks()
}

func cleanupGeoBlocksTasks() {
	for {
		time.Sleep(10 * time.Minute)
		now := time.Now()
		geoBlocksTaskMu.Lock()
		for id, task := range geoBlocksTasks {
			if now.Sub(task.CreatedAt) > geoBlocksTaskTTL {
				delete(geoBlocksTasks, id)
			}
		}
		geoBlocksTaskMu.Unlock()
	}
}

// StartGeoBlocksGeneration 启动批量 GEO 结构化内容生成任务
func StartGeoBlocksGeneration(recordType string, ids []int) string {
	taskID := fmt.Sprintf("geo-%s-%d", recordType, time.Now().UnixNano())
	task := &GeoBlocksTask{
		ID:        taskID,
		Type:      recordType,
		Status:    GeoBlocksTaskStatusPending,
		Total:     len(ids),
		Items:     make([]GeoBlocksTaskItem, 0, len(ids)),
		CreatedAt: time.Now(),
	}
	geoBlocksTaskMu.Lock()
	geoBlocksTasks[taskID] = task
	geoBlocksTaskMu.Unlock()

	go processGeoBlocksBatch(task, ids)
	return taskID
}

// GetGeoBlocksTask 查询任务状态
func GetGeoBlocksTask(taskID string) *GeoBlocksTask {
	geoBlocksTaskMu.RLock()
	defer geoBlocksTaskMu.RUnlock()
	return geoBlocksTasks[taskID]
}

// GeneratePromptGeoBlocks 为单个 Prompt 生成 GEO 结构化内容
func GeneratePromptGeoBlocks(id int) error {
	pwc, err := model.GetPromptById(id)
	if err != nil || pwc == nil || pwc.Prompt == nil {
		return fmt.Errorf("prompt not found")
	}
	p := pwc.Prompt
	if p.Title == "" || p.Content == "" {
		return fmt.Errorf("empty title or content")
	}

	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		return fmt.Errorf("AI translation not configured")
	}

	systemPrompt := `You are a GEO (Generative Engine Optimization) content specialist. Your task is to analyze a prompt and generate structured content blocks that help AI search engines understand and cite this prompt.

Generate a JSON object with exactly these keys:
- "scenarios": string (80-150 words). First sentence directly answers what scenarios this prompt is best for. Then expand with 2-3 specific use cases. Use natural, search-friendly language.
- "steps": array of 3-5 strings. Action-oriented steps for using this prompt. Each step should be concise (under 20 words). Include any key variables in [brackets].
- "tips": string (60-120 words). Practical tips for getting the best results, including parameter tuning advice.

Rules:
1. Return ONLY valid JSON. No markdown, no explanations.
2. Use the same language as the input content (assume Chinese input → Chinese output).
3. Make content factual, specific, and citation-worthy.
4. Include the prompt title naturally in the scenarios text if relevant.`

	userPrompt := fmt.Sprintf("Prompt Title: %s\n\nDescription: %s\n\nPrompt Content:\n%s\n\nGenerate the JSON now.", p.Title, p.Description, p.Content)

	response := callGeoBlocksAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		return fmt.Errorf("AI returned empty response")
	}

	geoBlocksJSON := extractGeoBlocksJSON(response)
	if geoBlocksJSON == "" {
		return fmt.Errorf("failed to extract valid JSON from AI response")
	}

	// 简单校验
	var validation struct {
		Scenarios string   `json:"scenarios"`
		Steps     []string `json:"steps"`
		Tips      string   `json:"tips"`
	}
	if err := common.Unmarshal([]byte(geoBlocksJSON), &validation); err != nil {
		return fmt.Errorf("invalid JSON structure: %w", err)
	}
	if validation.Scenarios == "" || len(validation.Steps) == 0 || validation.Tips == "" {
		return fmt.Errorf("AI response missing required fields")
	}

	updates := map[string]interface{}{
		"geo_blocks": geoBlocksJSON,
	}
	if err := model.DB.Model(&model.Prompt{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	// 异步翻译多语言 GEO 块
	translatePromptGeoBlocksAsync(id, geoBlocksJSON)
	return nil
}

// GenerateArticleGeoBlocks 为单篇文章生成 GEO 结构化内容
func GenerateArticleGeoBlocks(id int) error {
	article, err := model.GetArticleById(id)
	if err != nil || article == nil {
		return fmt.Errorf("article not found")
	}
	if article.Title == "" || article.Content == "" {
		return fmt.Errorf("empty title or content")
	}

	cfg := operation_setting.GetTranslateSetting()
	if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
		return fmt.Errorf("AI translation not configured")
	}

	systemPrompt := `You are a GEO (Generative Engine Optimization) content specialist. Your task is to analyze an article and extract/generenerate 4 structured semantic blocks that help AI search engines understand and cite this content.

Generate a JSON object with exactly these keys:
- "what": string (30-80 words). First sentence gives a clear definition or core concept. Then 1-2 sentences expand.
- "why": string (80-150 words). First sentence gives the main conclusion/value. Then explain 2-3 specific scenarios or problems it solves.
- "how": array of 3-6 strings. Action-oriented step-by-step guide. Each step under 25 words. Use imperative verbs.
- "summary": string (under 40 words). Core conclusion that AI can directly quote.

Rules:
1. Return ONLY valid JSON. No markdown, no explanations.
2. Use the same language as the article content (assume Chinese input → Chinese output).
3. Every block must start with a direct, quotable conclusion sentence.
4. Include specific facts, numbers, or tool names when available in the article.
5. The output should feel like answers to real user search queries.`

	userPrompt := fmt.Sprintf("Article Title: %s\n\nArticle Summary: %s\n\nArticle Content:\n%s\n\nGenerate the JSON now.", article.Title, article.Summary, article.Content)

	response := callGeoBlocksAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		return fmt.Errorf("AI returned empty response")
	}

	geoBlocksJSON := extractGeoBlocksJSON(response)
	if geoBlocksJSON == "" {
		return fmt.Errorf("failed to extract valid JSON from AI response")
	}

	// 简单校验
	var validation struct {
		What    string   `json:"what"`
		Why     string   `json:"why"`
		How     []string `json:"how"`
		Summary string   `json:"summary"`
	}
	if err := common.Unmarshal([]byte(geoBlocksJSON), &validation); err != nil {
		return fmt.Errorf("invalid JSON structure: %w", err)
	}
	if validation.What == "" || validation.Why == "" || len(validation.How) == 0 || validation.Summary == "" {
		return fmt.Errorf("AI response missing required fields")
	}

	updates := map[string]interface{}{
		"geo_blocks": geoBlocksJSON,
	}
	if err := model.DB.Model(&model.Article{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	// 异步翻译多语言 GEO 块
	translateArticleGeoBlocksAsync(id, geoBlocksJSON)
	return nil
}

// ========== 批量处理逻辑 ==========

func processGeoBlocksBatch(task *GeoBlocksTask, ids []int) {
	task.Status = GeoBlocksTaskStatusRunning

	for _, id := range ids {
		var err error
		if task.Type == "prompt" {
			err = GeneratePromptGeoBlocks(id)
		} else {
			err = GenerateArticleGeoBlocks(id)
		}

		task.Completed++
		if err != nil {
			task.Failed++
			task.Items = append(task.Items, GeoBlocksTaskItem{
				ID:     id,
				Status: "failed",
				Error:  err.Error(),
			})
			common.SysLog(fmt.Sprintf("GeoBlocks generation failed for %s %d: %v", task.Type, id, err))
		} else {
			task.Items = append(task.Items, GeoBlocksTaskItem{
				ID:     id,
				Status: "success",
			})
		}

		// 每记录间隔，避免限流
		time.Sleep(1 * time.Second)
	}

	if task.Failed == task.Total {
		task.Status = GeoBlocksTaskStatusFailed
	} else {
		task.Status = GeoBlocksTaskStatusCompleted
	}
}

// ========== AI 调用（内联实现） ==========

func callGeoBlocksAI(cfg *operation_setting.TranslateSetting, systemPrompt, userPrompt string) string {
	reqBody := map[string]interface{}{
		"model": cfg.TranslateAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		common.SysLog("GeoBlocks marshal error: " + err.Error())
		return ""
	}

	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		client := &http.Client{Timeout: 120 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 125*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TranslateAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			cancel()
			return ""
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.TranslateAIApiKey)

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			if attempt < maxRetries {
				continue
			}
			return ""
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < maxRetries {
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			return ""
		}

		var apiResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := common.Unmarshal(bodyBytes, &apiResp); err != nil {
			return ""
		}
		if len(apiResp.Choices) == 0 {
			return ""
		}

		content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
		if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
			content = content[1 : len(content)-1]
		}
		return content
	}

	return ""
}

func extractGeoBlocksJSON(response string) string {
	response = strings.TrimSpace(response)

	// 去掉 markdown 代码块
	if strings.HasPrefix(response, "```") {
		start := strings.Index(response, "\n")
		if start != -1 {
			end := strings.LastIndex(response, "```")
			if end > start {
				response = strings.TrimSpace(response[start:end])
			}
		}
	}

	// 找第一个 { 和最后一个 }
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}

	return response[start : end+1]
}

// ========== GEO 块多语言翻译 ==========

var geoBlocksTargetLangs = []string{"en", "fr", "ru", "ja", "vi", "ko", "es", "de", "pt", "it", "ar"}

func translateGeoBlocksJSON(cfg *operation_setting.TranslateSetting, geoBlocksJSON, targetLang string) (string, error) {
	targetLangName := getSEOLangName(targetLang)
	systemPrompt := fmt.Sprintf(`You are a professional translator. Translate the following JSON from Chinese to %s.

Rules:
1. Keep ALL keys exactly the same (do not translate keys)
2. Translate ALL string values and ALL array string items into %s
3. Return ONLY valid JSON. No markdown, no explanations.`, targetLangName, targetLangName)

	response := callGeoBlocksAI(cfg, systemPrompt, geoBlocksJSON)
	if response == "" {
		return "", fmt.Errorf("empty translation response")
	}

	translatedJSON := extractGeoBlocksJSON(response)
	if translatedJSON == "" {
		return "", fmt.Errorf("failed to extract JSON from translation")
	}

	return translatedJSON, nil
}

func translatePromptGeoBlocksAsync(id int, geoBlocksJSON string) {
	go func() {
		cfg := operation_setting.GetTranslateSetting()
		if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
			return
		}

		geoBlocksI18n := make(map[string]string)
		for _, lang := range geoBlocksTargetLangs {
			translated, err := translateGeoBlocksJSON(cfg, geoBlocksJSON, lang)
			if err != nil || translated == "" {
				common.SysLog(fmt.Sprintf("GeoBlocks translate failed: prompt %d lang=%s err=%v", id, lang, err))
				continue
			}
			geoBlocksI18n[lang] = translated
			time.Sleep(1 * time.Second) // 避免限流
		}

		if len(geoBlocksI18n) == 0 {
			return
		}

		i18nJSON, _ := common.Marshal(geoBlocksI18n)
		model.DB.Model(&model.Prompt{}).Where("id = ?", id).Updates(map[string]interface{}{
			"geo_blocks_i18n": string(i18nJSON),
		})
		common.SysLog(fmt.Sprintf("GeoBlocks i18n saved: prompt %d, langs=%d", id, len(geoBlocksI18n)))
	}()
}

func translateArticleGeoBlocksAsync(id int, geoBlocksJSON string) {
	go func() {
		cfg := operation_setting.GetTranslateSetting()
		if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
			return
		}

		geoBlocksI18n := make(map[string]string)
		for _, lang := range geoBlocksTargetLangs {
			translated, err := translateGeoBlocksJSON(cfg, geoBlocksJSON, lang)
			if err != nil || translated == "" {
				common.SysLog(fmt.Sprintf("GeoBlocks translate failed: article %d lang=%s err=%v", id, lang, err))
				continue
			}
			geoBlocksI18n[lang] = translated
			time.Sleep(1 * time.Second) // 避免限流
		}

		if len(geoBlocksI18n) == 0 {
			return
		}

		i18nJSON, _ := common.Marshal(geoBlocksI18n)
		model.DB.Model(&model.Article{}).Where("id = ?", id).Updates(map[string]interface{}{
			"geo_blocks_i18n": string(i18nJSON),
		})
		common.SysLog(fmt.Sprintf("GeoBlocks i18n saved: article %d, langs=%d", id, len(geoBlocksI18n)))
	}()
}

// ========== GEO 块自动翻译轮询 ==========

func init() {
	// 5分钟后启动 GEO 自动翻译轮询
	time.AfterFunc(5*time.Minute, func() {
		go startGeoAutoTranslatePoller()
	})
}

func startGeoAutoTranslatePoller() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cfg := operation_setting.GetTranslateSetting()
		if !cfg.TranslateAIEnabled || cfg.TranslateAIApiKey == "" || cfg.TranslateAIBaseURL == "" {
			continue
		}
		pollAndAutoTranslateGeoBlocks()
	}
}

func getMissingGeoBlocksLangs(recordType string, id int, targetLangs []string) ([]string, string) {
	var existingI18n, geoBlocks string
	if recordType == "prompt" {
		var p model.Prompt
		if err := model.DB.Model(&model.Prompt{}).Select("geo_blocks_i18n", "geo_blocks").Where("id = ?", id).First(&p).Error; err != nil {
			return targetLangs, ""
		}
		existingI18n = p.GeoBlocksI18n
		geoBlocks = p.GeoBlocks
	} else {
		var a model.Article
		if err := model.DB.Model(&model.Article{}).Select("geo_blocks_i18n", "geo_blocks").Where("id = ?", id).First(&a).Error; err != nil {
			return targetLangs, ""
		}
		existingI18n = a.GeoBlocksI18n
		geoBlocks = a.GeoBlocks
	}

	if existingI18n == "" {
		return targetLangs, geoBlocks
	}

	var i18nMap map[string]string
	if err := common.Unmarshal([]byte(existingI18n), &i18nMap); err != nil {
		return targetLangs, geoBlocks
	}

	missing := []string{}
	for _, lang := range targetLangs {
		if v, ok := i18nMap[lang]; !ok || v == "" {
			missing = append(missing, lang)
		}
	}
	return missing, geoBlocks
}

func pollAndAutoTranslateGeoBlocks() {
	const batchSize = 20

	// 1. 扫描有 geo_blocks 但 geo_blocks_i18n 不完整的 Prompts
	var promptIDs []int
	err := model.DB.Model(&model.Prompt{}).
		Select("id").
		Where("geo_blocks != ? AND geo_blocks IS NOT NULL", "").
		Limit(batchSize).
		Order("id desc").
		Pluck("id", &promptIDs).Error
	if err != nil {
		common.SysLog("GeoAutoTranslate poller query prompts error: " + err.Error())
	}

	processed := 0
	for _, id := range promptIDs {
		missingLangs, geoBlocks := getMissingGeoBlocksLangs("prompt", id, geoBlocksTargetLangs)
		if len(missingLangs) == 0 {
			continue // 已经完整
		}
		if geoBlocks == "" {
			continue
		}

		cfg := operation_setting.GetTranslateSetting()
		geoBlocksI18n := make(map[string]string)

		// 解析已有的 i18n
		var p model.Prompt
		model.DB.Model(&model.Prompt{}).Select("geo_blocks_i18n").Where("id = ?", id).First(&p)
		if p.GeoBlocksI18n != "" {
			_ = common.Unmarshal([]byte(p.GeoBlocksI18n), &geoBlocksI18n)
		}

		for _, lang := range missingLangs {
			translated, err := translateGeoBlocksJSON(cfg, geoBlocks, lang)
			if err != nil || translated == "" {
				common.SysLog(fmt.Sprintf("GeoAutoTranslate failed: prompt %d lang=%s err=%v", id, lang, err))
				continue
			}
			geoBlocksI18n[lang] = translated
			processed++
			time.Sleep(1 * time.Second)
		}

		if len(geoBlocksI18n) > 0 {
			i18nJSON, _ := common.Marshal(geoBlocksI18n)
			model.DB.Model(&model.Prompt{}).Where("id = ?", id).Updates(map[string]interface{}{
				"geo_blocks_i18n": string(i18nJSON),
			})
			common.SysLog(fmt.Sprintf("GeoAutoTranslate saved: prompt %d, langs=%d", id, len(geoBlocksI18n)))
		}
	}
	if len(promptIDs) > 0 {
		common.SysLog(fmt.Sprintf("GeoAutoTranslate poller: processed %d prompts (%d langs)", len(promptIDs), processed))
	}

	// 2. 扫描有 geo_blocks 但 geo_blocks_i18n 不完整的 Articles
	var articleIDs []int
	err = model.DB.Model(&model.Article{}).
		Select("id").
		Where("geo_blocks != ? AND geo_blocks IS NOT NULL", "").
		Limit(batchSize).
		Order("id desc").
		Pluck("id", &articleIDs).Error
	if err != nil {
		common.SysLog("GeoAutoTranslate poller query articles error: " + err.Error())
	}

	articleProcessed := 0
	for _, id := range articleIDs {
		missingLangs, geoBlocks := getMissingGeoBlocksLangs("article", id, geoBlocksTargetLangs)
		if len(missingLangs) == 0 {
			continue
		}
		if geoBlocks == "" {
			continue
		}

		cfg := operation_setting.GetTranslateSetting()
		geoBlocksI18n := make(map[string]string)

		var a model.Article
		model.DB.Model(&model.Article{}).Select("geo_blocks_i18n").Where("id = ?", id).First(&a)
		if a.GeoBlocksI18n != "" {
			_ = common.Unmarshal([]byte(a.GeoBlocksI18n), &geoBlocksI18n)
		}

		for _, lang := range missingLangs {
			translated, err := translateGeoBlocksJSON(cfg, geoBlocks, lang)
			if err != nil || translated == "" {
				common.SysLog(fmt.Sprintf("GeoAutoTranslate failed: article %d lang=%s err=%v", id, lang, err))
				continue
			}
			geoBlocksI18n[lang] = translated
			articleProcessed++
			time.Sleep(1 * time.Second)
		}

		if len(geoBlocksI18n) > 0 {
			i18nJSON, _ := common.Marshal(geoBlocksI18n)
			model.DB.Model(&model.Article{}).Where("id = ?", id).Updates(map[string]interface{}{
				"geo_blocks_i18n": string(i18nJSON),
			})
			common.SysLog(fmt.Sprintf("GeoAutoTranslate saved: article %d, langs=%d", id, len(geoBlocksI18n)))
		}
	}
	if len(articleIDs) > 0 {
		common.SysLog(fmt.Sprintf("GeoAutoTranslate poller: processed %d articles (%d langs)", len(articleIDs), articleProcessed))
	}
}
