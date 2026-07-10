package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// RepairPromptVideoUrls 批量修复提示词视频 URL（管理员接口）
// GET /api/admin/prompt/repair-video-urls?dry_run=true
func RepairPromptVideoUrls(c *gin.Context) {
	dryRun := c.DefaultQuery("dry_run", "true") == "true"

	result, err := service.RepairPromptVideoUrls(c.Request.Context(), dryRun)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// GetBrokenPromptVideoUrls 获取所有视频 URL 异常的提示词列表（只读预览）
// GET /api/admin/prompt/broken-video-urls?page=1&size=50
func GetBrokenPromptVideoUrls(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 50
	}
	offset := (page - 1) * size

	var total int64
	model.DB.Model(&model.Prompt{}).
		Where("media_type = ?", "video").
		Where("video_url = '' OR video_url IS NULL OR video_url = cover_image_url").
		Where("status = ?", 1).
		Count(&total)

	var prompts []model.Prompt
	model.DB.Where("media_type = ?", "video").
		Where("video_url = '' OR video_url IS NULL OR video_url = cover_image_url").
		Where("status = ?", 1).
		Select("id", "title", "cover_image_url", "video_url", "source_url", "author", "source", "model", "created_time").
		Order("id desc").
		Limit(size).Offset(offset).
		Find(&prompts)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"total": total,
			"page":  page,
			"size":  size,
			"items": prompts,
		},
	})
}
