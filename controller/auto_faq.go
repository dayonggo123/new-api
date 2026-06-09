package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// ========== 单条 FAQ 生成 ==========

// AutoGenerateArticleFAQ 为单篇文章自动生成 FAQ
// POST /api/admin/articles/:id/auto-faq
func AutoGenerateArticleFAQ(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	article, err := model.GetArticleById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	faqJSON, err := generateArticleFAQWithAI(article)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 保存到数据库
	article.Faq = faqJSON
	article.UpdatedTime = time.Now().Unix()
	if err := article.Update(); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":      article.Id,
		"title":   article.Title,
		"faq":     faqJSON,
		"message": "文章 FAQ 自动生成并保存成功",
	})
}

// AutoGeneratePromptFAQ 为单个 Prompt 自动生成 FAQ
// POST /api/admin/prompts/:id/auto-faq
func AutoGeneratePromptFAQ(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	prompt, err := model.GetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	faqJSON, err := generatePromptFAQWithAI(prompt)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 保存到数据库
	prompt.Faq = faqJSON
	prompt.UpdatedTime = time.Now().Unix()
	if err := prompt.Update(); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":      prompt.Id,
		"title":   prompt.Title,
		"faq":     faqJSON,
		"message": "Prompt FAQ 自动生成并保存成功",
	})
}

// ========== 批量 FAQ 生成 ==========

// BatchAutoGenerateArticleFAQ 批量为文章生成 FAQ
// POST /api/admin/articles/auto-faq/batch
func BatchAutoGenerateArticleFAQ(c *gin.Context) {
	var req struct {
		Ids []int `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "请选择要生成 FAQ 的文章")
		return
	}
	if len(req.Ids) > 20 {
		common.ApiErrorMsg(c, "单次最多处理 20 篇文章")
		return
	}

	taskID := startAutoFAQBatchTask("article", req.Ids)
	common.ApiSuccess(c, gin.H{
		"task_id": taskID,
		"status":  "running",
		"total":   len(req.Ids),
		"message": fmt.Sprintf("已启动 %d 篇文章的批量 FAQ 生成任务", len(req.Ids)),
	})
}

// BatchAutoGeneratePromptFAQ 批量为 Prompt 生成 FAQ
// POST /api/admin/prompts/auto-faq/batch
func BatchAutoGeneratePromptFAQ(c *gin.Context) {
	var req struct {
		Ids []int `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "请选择要生成 FAQ 的提示词")
		return
	}
	if len(req.Ids) > 20 {
		common.ApiErrorMsg(c, "单次最多处理 20 个提示词")
		return
	}

	taskID := startAutoFAQBatchTask("prompt", req.Ids)
	common.ApiSuccess(c, gin.H{
		"task_id": taskID,
		"status":  "running",
		"total":   len(req.Ids),
		"message": fmt.Sprintf("已启动 %d 个提示词的批量 FAQ 生成任务", len(req.Ids)),
	})
}

// GetAutoFAQBatchStatus 查询批量 FAQ 生成任务状态
// GET /api/admin/auto-faq/batch/:task_id
func GetAutoFAQBatchStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		common.ApiErrorMsg(c, "task_id 不能为空")
		return
	}

	task := getAutoFAQBatchTask(taskID)
	if task == nil {
		common.ApiErrorMsg(c, "任务不存在或已过期")
		return
	}

	common.ApiSuccess(c, task)
}

// ========== AI 生成逻辑 ==========

// generateArticleFAQWithAI 调用 AI 为文章生成 FAQ
func generateArticleFAQWithAI(article *model.Article) (string, error) {
	cfg := operation_setting.GetTranslateSetting()
	if cfg == nil || cfg.TranslateAIModel == "" {
		return "", fmt.Errorf("翻译 AI 模型未配置，请先在后台「系统设置 → 翻译设置」中配置")
	}

	systemPrompt := `You are a professional SEO content writer specializing in GEO (Generative Engine Optimization). Your task is to generate high-quality FAQ content based on the given article.

CRITICAL RULES:
1. Generate 3-5 FAQ pairs (question + answer)
2. Each question MUST use complete natural language with subject-verb-object structure (15-40 Chinese characters)
3. Each answer MUST start with a direct conclusion in the first sentence (15-30 characters)
4. Each answer should be 50-150 Chinese characters total
5. Include at least 1 specific fact/data/number in each answer
6. NEVER use vague pronouns like "it" or "this" - always use the full name "HarseTV"
7. Tone must be objective and informative, NOT marketing or promotional
8. Output MUST be valid JSON array format

Output format:
[
  {
    "question": "HarseTV 支持哪些 AI 图像生成模型？",
    "answer": "HarseTV 目前支持 Seedream、Imagen 3、Nano Banana 三种图像生成模型。你可以在画布节点的模型下拉菜单中直接切换，无需额外配置 API 密钥。"
  }
]`

	userPrompt := fmt.Sprintf(`请根据以下文章内容生成 FAQ。

文章标题：%s
文章摘要：%s
文章内容（前 2000 字）：%s

要求：
1. 生成 3-5 个常见问题及答案
2. 问题使用完整自然语言，包含主谓宾
3. 答案首句直接给结论
4. 每个答案 50-150 字
5. 包含至少 1 个具体数据/事实
6. 避免使用"它""这个"等代词，用全称
7. 语气客观，不营销
8. 只返回 JSON 数组，不要其他内容`,
		article.Title,
		truncateString(article.Summary, 500),
		truncateString(article.Content, 2000),
	)

	response := callTranslateAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		return "", fmt.Errorf("AI 生成 FAQ 失败，返回为空")
	}

	// 清理并验证 JSON
	faqJSON := cleanFAQResponse(response)
	if !isValidFAQJSON(faqJSON) {
		return "", fmt.Errorf("AI 返回的 FAQ 格式无效")
	}

	return faqJSON, nil
}

// generatePromptFAQWithAI 调用 AI 为 Prompt 生成 FAQ
func generatePromptFAQWithAI(prompt *model.PromptWithCategory) (string, error) {
	cfg := operation_setting.GetTranslateSetting()
	if cfg == nil || cfg.TranslateAIModel == "" {
		return "", fmt.Errorf("翻译 AI 模型未配置，请先在后台「系统设置 → 翻译设置」中配置")
	}

	systemPrompt := `You are a professional SEO content writer specializing in GEO (Generative Engine Optimization). Your task is to generate high-quality FAQ content for an AI prompt/tool page.

CRITICAL RULES:
1. Generate 2-3 FAQ pairs (question + answer) - ONLY for complex prompts (description > 100 chars or has multiple parameters)
2. Each question MUST use complete natural language (15-40 Chinese characters)
3. Each answer MUST start with a direct conclusion in the first sentence (15-30 characters)
4. Each answer should be 50-150 Chinese characters total
5. Include at least 1 specific fact/data/operation path in each answer
6. NEVER use vague pronouns - always use the full prompt name or "HarseTV"
7. Focus on: usage scenarios, parameter adjustment, model selection, common failures
8. Output MUST be valid JSON array format

Output format:
[
  {
    "question": "这个提示词适合生成什么类型的内容？",
    "answer": "这个提示词专为高端时尚摄影设计，适合生成杂志封面和品牌 Campaign 主视觉。最佳使用场景是需要强烈光影对比和电影级色调的人像图像。"
  }
]`

	contentPreview := prompt.Content
	if len(contentPreview) > 1500 {
		contentPreview = contentPreview[:1500]
	}

	userPrompt := fmt.Sprintf(`请为以下提示词生成 FAQ。

提示词标题：%s
提示词描述：%s
提示词内容（前 1500 字）：%s

要求：
1. 生成 2-3 个常见问题及答案
2. 问题使用完整自然语言
3. 答案首句直接给结论
4. 每个答案 50-150 字
5. 包含具体使用场景或参数说明
6. 避免代词模糊
7. 只返回 JSON 数组`,
		prompt.Title,
		prompt.Description,
		contentPreview,
	)

	response := callTranslateAI(cfg, systemPrompt, userPrompt)
	if response == "" {
		return "", fmt.Errorf("AI 生成 FAQ 失败，返回为空")
	}

	faqJSON := cleanFAQResponse(response)
	if !isValidFAQJSON(faqJSON) {
		return "", fmt.Errorf("AI 返回的 FAQ 格式无效")
	}

	return faqJSON, nil
}

// ========== 批量任务管理 ==========

// AutoFAQBatchTask 批量 FAQ 生成任务
type AutoFAQBatchTask struct {
	ID        string               `json:"id"`
	Type      string               `json:"type"` // article | prompt
	Status    string               `json:"status"`
	Total     int                  `json:"total"`
	Completed int                  `json:"completed"`
	Failed    int                  `json:"failed"`
	Results   []AutoFAQTaskResult  `json:"results"`
	CreatedAt string               `json:"created_at"`
}

// AutoFAQTaskResult 单条结果
type AutoFAQTaskResult struct {
	Id      int    `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"` // success | failed
	Message string `json:"message"`
}

var (
	autoFAQTaskStore = make(map[string]*AutoFAQBatchTask)
)

func startAutoFAQBatchTask(taskType string, ids []int) string {
	taskID := fmt.Sprintf("auto-faq-%d", time.Now().UnixNano())
	task := &AutoFAQBatchTask{
		ID:        taskID,
		Type:      taskType,
		Status:    "running",
		Total:     len(ids),
		Completed: 0,
		Failed:    0,
		Results:   make([]AutoFAQTaskResult, 0, len(ids)),
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	autoFAQTaskStore[taskID] = task

	go processAutoFAQBatchTask(task, ids)
	return taskID
}

func processAutoFAQBatchTask(task *AutoFAQBatchTask, ids []int) {
	for _, id := range ids {
		result := AutoFAQTaskResult{Id: id}

		if task.Type == "article" {
			article, err := model.GetArticleById(id)
			if err != nil {
				result.Status = "failed"
				result.Message = err.Error()
				autoFAQTaskStore[task.ID].Failed++
			} else {
				faqJSON, err := generateArticleFAQWithAI(article)
				if err != nil {
					result.Status = "failed"
					result.Message = err.Error()
					autoFAQTaskStore[task.ID].Failed++
				} else {
					article.Faq = faqJSON
					article.UpdatedTime = time.Now().Unix()
					if err := article.Update(); err != nil {
						result.Status = "failed"
						result.Message = err.Error()
						autoFAQTaskStore[task.ID].Failed++
					} else {
						result.Status = "success"
						result.Title = article.Title
						autoFAQTaskStore[task.ID].Completed++
					}
				}
			}
		} else {
			prompt, err := model.GetPromptById(id)
			if err != nil {
				result.Status = "failed"
				result.Message = err.Error()
				autoFAQTaskStore[task.ID].Failed++
			} else {
				faqJSON, err := generatePromptFAQWithAI(prompt)
				if err != nil {
					result.Status = "failed"
					result.Message = err.Error()
					autoFAQTaskStore[task.ID].Failed++
				} else {
					prompt.Faq = faqJSON
					prompt.UpdatedTime = time.Now().Unix()
					if err := prompt.Update(); err != nil {
						result.Status = "failed"
						result.Message = err.Error()
						autoFAQTaskStore[task.ID].Failed++
					} else {
						result.Status = "success"
						result.Title = prompt.Title
						autoFAQTaskStore[task.ID].Completed++
					}
				}
			}
		}

		autoFAQTaskStore[task.ID].Results = append(autoFAQTaskStore[task.ID].Results, result)

		// 避免 AI API 限流，每条之间间隔 1 秒
		time.Sleep(1 * time.Second)
	}

	autoFAQTaskStore[task.ID].Status = "completed"
	// 2 小时后清理任务
	go func(tid string) {
		time.Sleep(2 * time.Hour)
		delete(autoFAQTaskStore, tid)
	}(task.ID)
}

func getAutoFAQBatchTask(taskID string) *AutoFAQBatchTask {
	return autoFAQTaskStore[taskID]
}

// ========== 工具函数 ==========

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func cleanFAQResponse(response string) string {
	response = strings.TrimSpace(response)
	// 去掉可能的 markdown 代码块
	if strings.HasPrefix(response, "```") {
		start := strings.Index(response, "[")
		if start == -1 {
			start = strings.Index(response, "{")
		}
		end := strings.LastIndex(response, "```")
		if start != -1 && end > start {
			response = strings.TrimSpace(response[start:end])
		}
	}
	return response
}

func isValidFAQJSON(jsonStr string) bool {
	// 简单验证：包含 question 和 answer 关键字
	if !strings.Contains(jsonStr, "question") || !strings.Contains(jsonStr, "answer") {
		return false
	}
	// 尝试解析为通用结构
	var faqs []map[string]string
	if err := common.Unmarshal([]byte(jsonStr), &faqs); err != nil {
		return false
	}
	return len(faqs) > 0
}
