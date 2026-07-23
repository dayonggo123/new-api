package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// GetTikHubPriceConfigs 获取 TikHub 收费配置列表（管理员）
// GET /api/admin/tikhub/prices
// 支持 category 参数筛选
func GetTikHubPriceConfigs(c *gin.Context) {
	category := c.Query("category")

	var configs []*model.TikHubPriceConfig
	var err error

	if category != "" && category != "all" {
		err = model.DB.Where("category = ?", category).Order("id ASC").Find(&configs).Error
	} else {
		configs, err = model.GetAllTikHubPriceConfigs()
	}

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
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		Category      string  `json:"category"`
		RequiresCookie bool   `json:"requires_cookie"`
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
	if req.Category != "" {
		config.Category = req.Category
	}
	config.RequiresCookie = req.RequiresCookie
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
		Endpoint        string  `json:"endpoint" binding:"required"`
		Name            string  `json:"name" binding:"required"`
		Description     string  `json:"description"`
		Category        string  `json:"category"`
		RequiresCookie  bool    `json:"requires_cookie"`
		Price           float64 `json:"price"`
		VipPrice        float64 `json:"vip_price"`
		SvipPrice       float64 `json:"svip_price"`
		FreeQuota       int     `json:"free_quota"`
		VipFreeQuota    int     `json:"vip_free_quota"`
		SvipFreeQuota   int     `json:"svip_free_quota"`
		Enabled         bool    `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数解析失败",
		})
		return
	}

	config := model.TikHubPriceConfig{
		Endpoint:        req.Endpoint,
		Name:            req.Name,
		Description:     req.Description,
		Category:        req.Category,
		RequiresCookie:  req.RequiresCookie,
		Price:           req.Price,
		VipPrice:        req.VipPrice,
		SvipPrice:       req.SvipPrice,
		FreeQuota:       req.FreeQuota,
		VipFreeQuota:    req.VipFreeQuota,
		SvipFreeQuota:   req.SvipFreeQuota,
		Enabled:         req.Enabled,
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

// TestTikHubEndpoint 测试 TikHub 接口是否能正常调用
// POST /api/admin/tikhub/prices/test
func TestTikHubEndpoint(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "TikHub 接口未启用",
		})
		return
	}

	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "TikHub API Key 未配置",
		})
		return
	}

	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
		Params   map[string]string `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "endpoint 不能为空",
		})
		return
	}

	// 根据 endpoint 调用对应的接口
	var body []byte
	var err error

	switch req.Endpoint {
	// 视频类
	case "video":
		awemeID := req.Params["aweme_id"]
		if awemeID == "" {
			awemeID = "7323948350548490502" // 默认测试视频
		}
		body, err = service.FetchTikHubSingleVideo(c.Request.Context(), awemeID)

	case "video-comments":
		awemeID := req.Params["aweme_id"]
		if awemeID == "" {
			awemeID = "7323948350548490502"
		}
		body, err = service.FetchTikHubVideoComments(c.Request.Context(), awemeID, 0, 10)

	case "video-metrics":
		awemeID := req.Params["aweme_id"]
		if awemeID == "" {
			awemeID = "7323948350548490502"
		}
		body, err = service.FetchTikHubVideoMetrics(c.Request.Context(), awemeID)

	case "detect-fake-views":
		awemeID := req.Params["aweme_id"]
		if awemeID == "" {
			awemeID = "7323948350548490502"
		}
		body, err = service.FetchTikHubDetectFakeViews(c.Request.Context(), awemeID, "")

	// 搜索类
	case "trending-search-words":
		body, err = service.FetchTikHubTrendingSearchWords(c.Request.Context())

	case "general-search-result":
		keyword := req.Params["keyword"]
		if keyword == "" {
			keyword = "dance"
		}
		body, err = service.FetchTikHubGeneralSearchResult(c.Request.Context(), keyword, 0, 10, 0, 0)

	// 音乐类
	case "music-chart-list":
		body, err = service.FetchTikHubMusicChartList(c.Request.Context(), 1, 0, 10)

	// 话题类
	case "trends-hashtag-list":
		body, err = service.FetchTikHubTrendsHashtagList(c.Request.Context(), 7, "US", 1, 10, 0)

	// 商品类
	case "hot-selling-products-list":
		region := req.Params["region"]
		if region == "" {
			region = "US"
		}
		body, err = service.FetchTikHubHotSellingProductsList(c.Request.Context(), region, 10)

	// 广告类
	case "ads-search-ads":
		body, err = service.FetchTikHubSearchAds(c.Request.Context(), "phone", 1, 1, 180, 0, 1, 10, "for_you", "US", 1, "en", "")

	case "ads-top-ads-spotlight":
		body, err = service.FetchTikHubTopAdsSpotlight(c.Request.Context(), "", 1, 10)

	// 整合报告类
	case "report-product-analysis":
		productID := req.Params["product_id"]
		if productID == "" {
			productID = "test"
		}
		body, err = service.FetchProductAnalysisReport(c.Request.Context(), productID, "US")

	case "report-content-trends":
		body, err = service.FetchContentTrendsReport(c.Request.Context(), "US", 7, 0)

	case "report-video-analysis":
		awemeID := req.Params["aweme_id"]
		if awemeID == "" {
			awemeID = "7323948350548490502"
		}
		body, err = service.FetchVideoAnalysisReport(c.Request.Context(), awemeID, "", "")

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("不支持测试此接口: %s", req.Endpoint),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "测试失败: " + err.Error(),
			"endpoint": req.Endpoint,
			"status": "failed",
		})
		return
	}

	// 成功
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试成功",
		"endpoint": req.Endpoint,
		"status": "success",
		"data_length": len(body),
	})
}
