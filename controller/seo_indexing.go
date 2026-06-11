package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== Google Indexing ====================

// SubmitToGoogleIndexingRequest 提交索引请求
type SubmitToGoogleIndexingRequest struct {
	URL  string `json:"url" binding:"required"`
	Type string `json:"type"` // "URL_UPDATED" or "URL_DELETED"
}

// SubmitToGoogleIndexing 手动提交 URL 到 Google Indexing
// POST /api/admin/seo/indexing
func SubmitToGoogleIndexing(c *gin.Context) {
	var req SubmitToGoogleIndexingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.SubmitToGoogleIndexing(req.URL, req.Type)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, result)
}

// BatchSubmitToGoogleIndexingRequest 批量提交请求
type BatchSubmitToGoogleIndexingRequest struct {
	URLs []string `json:"urls" binding:"required"`
}

// BatchSubmitToGoogleIndexing 批量提交 URL
// POST /api/admin/seo/indexing/batch
func BatchSubmitToGoogleIndexing(c *gin.Context) {
	var req BatchSubmitToGoogleIndexingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.URLs) == 0 {
		common.ApiErrorMsg(c, "urls 不能为空")
		return
	}
	if len(req.URLs) > 100 {
		common.ApiErrorMsg(c, "单次最多 100 个 URL")
		return
	}

	results := service.BatchSubmitToGoogleIndexing(req.URLs)
	common.ApiSuccess(c, results)
}

// GetGoogleIndexingStatus 获取 URL 索引状态
// GET /api/admin/seo/indexing/status?url=https://harse.tv/article/xxx
func GetGoogleIndexingStatus(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		common.ApiErrorMsg(c, "url 参数必填")
		return
	}

	status, err := service.GetIndexingStatus(url)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, gin.H{
		"url":    url,
		"status": status,
	})
}
