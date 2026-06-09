package controller

import (
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// GeneratePromptGeoBlocks 为单个 Prompt 生成 GEO 结构化内容
// POST /api/admin/prompts/:id/geo-blocks
func GeneratePromptGeoBlocks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.GeneratePromptGeoBlocks(id); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":      id,
		"message": "Prompt GEO 结构化内容生成成功",
	})
}

// GenerateArticleGeoBlocks 为单篇文章生成 GEO 结构化内容
// POST /api/admin/articles/:id/geo-blocks
func GenerateArticleGeoBlocks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.GenerateArticleGeoBlocks(id); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":      id,
		"message": "文章 GEO 结构化内容生成成功",
	})
}

// BatchGeneratePromptGeoBlocksRequest 批量生成请求
type BatchGeneratePromptGeoBlocksRequest struct {
	Ids []int `json:"ids" binding:"required"`
}

// BatchGeneratePromptGeoBlocks 批量为 Prompt 生成 GEO 结构化内容
// POST /api/admin/prompts/geo-blocks/batch
func BatchGeneratePromptGeoBlocks(c *gin.Context) {
	var req BatchGeneratePromptGeoBlocksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "ids 不能为空")
		return
	}
	if len(req.Ids) > 50 {
		common.ApiErrorMsg(c, "单次最多 50 个")
		return
	}

	taskID := service.StartGeoBlocksGeneration("prompt", req.Ids)
	common.ApiSuccess(c, gin.H{
		"task_id": taskID,
		"message": fmt.Sprintf("已启动 %d 个 Prompt 的 GEO 结构化内容生成任务", len(req.Ids)),
	})
}

// BatchGenerateArticleGeoBlocks 批量为文章生成 GEO 结构化内容
// POST /api/admin/articles/geo-blocks/batch
func BatchGenerateArticleGeoBlocks(c *gin.Context) {
	var req BatchGeneratePromptGeoBlocksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "ids 不能为空")
		return
	}
	if len(req.Ids) > 50 {
		common.ApiErrorMsg(c, "单次最多 50 个")
		return
	}

	taskID := service.StartGeoBlocksGeneration("article", req.Ids)
	common.ApiSuccess(c, gin.H{
		"task_id": taskID,
		"message": fmt.Sprintf("已启动 %d 篇文章的 GEO 结构化内容生成任务", len(req.Ids)),
	})
}

// GetGeoBlocksBatchStatus 查询批量 GEO 结构化内容生成任务状态
// GET /api/admin/geo-blocks/batch/:task_id
func GetGeoBlocksBatchStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		common.ApiErrorMsg(c, "task_id 不能为空")
		return
	}

	task := service.GetGeoBlocksTask(taskID)
	if task == nil {
		common.ApiErrorMsg(c, "任务不存在或已过期")
		return
	}

	common.ApiSuccess(c, task)
}
