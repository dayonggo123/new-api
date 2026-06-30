package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// EchotikVideoRanklist 代理 EchoTik 视频榜单接口（缓存优先 + 上游回源）
// GET /api/public/echotik/video/ranklist
func EchotikVideoRanklist(c *gin.Context) {
	setting := operation_setting.GetEchotikSetting()
	if !setting.EchotikEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "EchoTik 接口未启用",
		})
		return
	}

	if setting.EchotikUsername == "" || setting.EchotikPassword == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "EchoTik 认证信息未配置",
		})
		return
	}

	params, err := parseEchotikRanklistParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	result, err := service.GetRanklist(c.Request.Context(), params, false)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Echotik ranklist error: %s", err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "no data available",
		})
		return
	}

	// 直接透传上游原始 JSON，避免二次序列化导致字段差异。
	c.Data(http.StatusOK, "application/json", []byte(result.RawResponse))
}

// parseEchotikRanklistParams 从 gin 查询参数解析并校验 EchoTik 榜单参数。
func parseEchotikRanklistParams(c *gin.Context) (*dto.EchotikRanklistParams, error) {
	date := c.Query("date")
	if date == "" {
		return nil, fmt.Errorf("date is required")
	}

	region := c.Query("region")
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	videoRankField, err := parseIntQuery(c, "video_rank_field")
	if err != nil {
		return nil, fmt.Errorf("video_rank_field is required and must be an integer")
	}

	rankType, err := parseIntQuery(c, "rank_type")
	if err != nil {
		return nil, fmt.Errorf("rank_type is required and must be an integer")
	}

	pageNum, err := parseIntQuery(c, "page_num")
	if err != nil || pageNum < 1 {
		return nil, fmt.Errorf("page_num is required and must be a positive integer")
	}

	pageSize, err := parseIntQuery(c, "page_size")
	if err != nil || pageSize < 1 {
		return nil, fmt.Errorf("page_size is required and must be a positive integer")
	}

	params := &dto.EchotikRanklistParams{
		Date:              date,
		Region:            region,
		VideoRankField:    videoRankField,
		RankType:          rankType,
		PageNum:           pageNum,
		PageSize:          pageSize,
		ProductCategoryID: c.Query("product_category_id"),
		CreatedByAI:       c.Query("created_by_ai"),
	}

	return params, nil
}

// parseIntQuery 解析查询参数为整数。
func parseIntQuery(c *gin.Context, key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, fmt.Errorf("%s is empty", key)
	}
	return strconv.Atoi(value)
}

// EchotikSettingStatus 返回 EchoTik 配置状态（仅管理员）
func EchotikSettingStatus(c *gin.Context) {
	setting := operation_setting.GetEchotikSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":       setting.EchotikEnabled,
			"base_url":      setting.EchotikBaseURL,
			"username_set":  setting.EchotikUsername != "",
			"password_set":  setting.EchotikPassword != "",
			"cache_enabled": setting.EchotikCacheEnabled,
		},
	})
}
