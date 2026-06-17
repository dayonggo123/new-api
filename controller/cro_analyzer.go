package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// AnalyzeCRO 对指定内容进行转化率优化（CRO）分析
// POST /api/admin/cro/analyze
func AnalyzeCRO(c *gin.Context) {
	var req service.CROAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.AnalyzeCRO(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, result)
}
