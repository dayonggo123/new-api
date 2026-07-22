package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetTikHubPriceConfigs 获取 TikHub 收费配置列表（管理员）
// GET /api/admin/tikhub/prices
func GetTikHubPriceConfigs(c *gin.Context) {
	configs, err := model.GetAllTikHubPriceConfigs()
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configs,
	})
}

// GetTikHubPrices 公开接口：获取 TikHub 收费配置列表（仅返回启用的配置）
// GET /api/public/tikhub/prices
func GetTikHubPrices(c *gin.Context) {
	configs, err := model.GetTikHubPriceConfigsForPublic()
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configs,
	})
}

// UpdateTikHubPriceConfig 更新 TikHub 收费配置
// PUT /api/admin/tikhub/prices/:id
func UpdateTikHubPriceConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的ID",
		})
		return
	}

	var req struct {
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		Price         float64 `json:"price"`
		VipPrice      float64 `json:"vip_price"`
		SvipPrice     float64 `json:"svip_price"`
		FreeQuota     int     `json:"free_quota"`
		VipFreeQuota  int     `json:"vip_free_quota"`
		SvipFreeQuota int     `json:"svip_free_quota"`
		Enabled       bool    `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数解析失败",
		})
		return
	}

	var config model.TikHubPriceConfig
	if err := model.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "配置不存在",
		})
		return
	}

	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Description != "" {
		config.Description = req.Description
	}
	config.Price = req.Price
	config.VipPrice = req.VipPrice
	config.SvipPrice = req.SvipPrice
	config.FreeQuota = req.FreeQuota
	config.VipFreeQuota = req.VipFreeQuota
	config.SvipFreeQuota = req.SvipFreeQuota
	config.Enabled = req.Enabled

	if err := config.Update(); err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    config,
	})
}

// CreateTikHubPriceConfig 创建 TikHub 收费配置
// POST /api/admin/tikhub/prices
func CreateTikHubPriceConfig(c *gin.Context) {
	var req struct {
		Endpoint      string  `json:"endpoint" binding:"required"`
		Name          string  `json:"name" binding:"required"`
		Description   string  `json:"description"`
		Price         float64 `json:"price"`
		VipPrice      float64 `json:"vip_price"`
		SvipPrice     float64 `json:"svip_price"`
		FreeQuota     int     `json:"free_quota"`
		VipFreeQuota  int     `json:"vip_free_quota"`
		SvipFreeQuota int     `json:"svip_free_quota"`
		Enabled       bool    `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数解析失败",
		})
		return
	}

	config := model.TikHubPriceConfig{
		Endpoint:      req.Endpoint,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		VipPrice:      req.VipPrice,
		SvipPrice:     req.SvipPrice,
		FreeQuota:     req.FreeQuota,
		VipFreeQuota:  req.VipFreeQuota,
		SvipFreeQuota: req.SvipFreeQuota,
		Enabled:       req.Enabled,
	}

	if err := config.Insert(); err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    config,
	})
}

// DeleteTikHubPriceConfig 删除 TikHub 收费配置
// DELETE /api/admin/tikhub/prices/:id
func DeleteTikHubPriceConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的ID",
		})
		return
	}

	if err := model.DeleteTikHubPriceConfig(id); err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// ChargeTikHubAPI TikHub 接口扣费中间件
// 返回价格，如果不需要扣费返回 0
func ChargeTikHubAPI(c *gin.Context, endpoint string) float64 {
	// 获取用户
	userID := c.GetInt("user_id")
	if userID == 0 {
		return 0
	}

	// 获取配置
	config, err := model.GetTikHubPriceConfigByEndpoint(endpoint)
	if err != nil || config == nil {
		return 0
	}

	if config.Price <= 0 {
		return 0
	}

	// 扣费
	price := int(config.Price)
	err = model.DecreaseUserQuota(userID, price, false)
	if err != nil {
		logger.LogError(c.Request.Context(), "TikHub扣费失败: "+err.Error())
		// 扣费失败不阻止请求，但记录日志
		return 0
	}

	logger.LogInfo(c.Request.Context(), "TikHub接口扣费成功")
	common.SysLog("TikHub扣费: 用户"+strconv.Itoa(userID)+" 调用"+endpoint+" 扣"+strconv.Itoa(price)+"积分")

	return config.Price
}

// InitTikHubPriceConfigs 初始化默认收费配置
// GET /api/admin/tikhub/prices/init
func InitTikHubPriceConfigs(c *gin.Context) {
	if err := model.InitDefaultTikHubPriceConfigs(); err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "初始化失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "初始化成功",
	})
}
