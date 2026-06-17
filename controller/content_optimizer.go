package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// OptimizeContent 对指定内容进行优化联审
// POST /api/admin/content/optimize
func OptimizeContent(c *gin.Context) {
	var req service.ContentOptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.OptimizeContent(&req)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, result)
}
