package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// chargeTikHubIfEnabled 如果启用收费则扣费，返回是否成功进行了扣费
// 美元转积分比例: 1 USD = 100 积分
const tikhubUSDToQuota = 100

func chargeTikHubIfEnabled(c *gin.Context, endpoint string) bool {
	userID := c.GetInt("user_id")
	if userID == 0 {
		return false
	}

	config, err := model.GetTikHubPriceConfigByEndpoint(endpoint)
	if err != nil || config == nil || config.Price <= 0 {
		return false
	}

	// 美元转换为积分
	quota := int(config.Price * tikhubUSDToQuota)
	err = model.DecreaseUserQuota(userID, quota, false)
	if err != nil {
		logger.LogError(c.Request.Context(), "TikHub扣费失败: "+err.Error())
		return false
	}

	// 记录使用日志到数据库
	logContent := fmt.Sprintf("TikHub接口 %s (%.2f USD)", config.Name, config.Price)
	model.RecordLog(userID, model.LogTypeConsume, logContent)

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("[TikHub] 用户 %d 调用接口 %s，消费 %.2f USD (%d 积分)", userID, endpoint, config.Price, quota))
	return true
}

// TikHubSingleVideo 代理 TikHub 获取单个 TikTok 作品数据 V2
// GET /api/public/tikhub/tiktok/video?aweme_id=7350810998023949599
func TikHubSingleVideo(c *gin.Context) {
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

	awemeID := c.Query("aweme_id")
	if awemeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "aweme_id 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubSingleVideo(c.Request.Context(), awemeID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "video")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubSettingStatus 返回 TikHub 配置状态（仅管理员）
func TikHubSettingStatus(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":      setting.TikHubEnabled,
			"base_url":     setting.TikHubBaseURL,
			"api_key_set":  setting.TikHubAPIKey != "",
		},
	})
}

// TikHubCommentKeywords 代理 TikHub 获取视频评论关键词分析
// GET /api/public/tikhub/tiktok/comment-keywords?item_id=7502551047378832671
func TikHubCommentKeywords(c *gin.Context) {
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

	itemID := c.Query("item_id")
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "item_id 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubCommentKeywords(c.Request.Context(), itemID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "comment-keywords")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubSingleVideoByShareURL 代理 TikHub 根据分享链接获取单个 TikTok 作品数据 V2
// GET /api/public/tikhub/tiktok/video-by-share-url?share_url=https://www.tiktok.com/@xxx/video/xxx
func TikHubSingleVideoByShareURL(c *gin.Context) {
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

	shareURL := c.Query("share_url")
	if shareURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "share_url 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubSingleVideoByShareURL(c.Request.Context(), shareURL)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "video-by-share-url")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubMusicChartList 代理 TikHub 获取 TikTok 音乐排行榜
// GET /api/public/tikhub/tiktok/music-chart-list?scene=0&cursor=0&count=50
func TikHubMusicChartList(c *gin.Context) {
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

	// scene: 0=Top 50, 1=Viral 50
	scene := 0
	if s := c.Query("scene"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil {
			scene = parsed
		}
	}

	// cursor: 分页游标
	cursor := 0
	if c := c.Query("cursor"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil {
			cursor = parsed
		}
	}

	// count: 每页数量，最大50
	count := 50
	if c := c.Query("count"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 && parsed <= 50 {
			count = parsed
		}
	}

	body, err := service.FetchTikHubMusicChartList(c.Request.Context(), scene, cursor, count)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "music-chart-list")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubTrendingSearchWords 代理 TikHub 获取每日趋势搜索关键词
// GET /api/public/tikhub/tiktok/trending-search-words
func TikHubTrendingSearchWords(c *gin.Context) {
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

	body, err := service.FetchTikHubTrendingSearchWords(c.Request.Context())
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "trending-search-words")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubAccountHealthStatus 代理 TikHub 获取创作者账号健康状态
// POST /api/public/tikhub/tiktok/account-health-status
func TikHubAccountHealthStatus(c *gin.Context) {
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

	// 解析请求体
	var reqBody struct {
		Cookie string `json:"cookie"`
		Proxy  string `json:"proxy"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "cookie 不能为空",
		})
		return
	}

	if reqBody.Cookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "cookie 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubAccountHealthStatus(c.Request.Context(), reqBody.Cookie, reqBody.Proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "account-health-status")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubAccountInsightsOverview 代理 TikHub 获取创作者账号概览
// POST /api/public/tikhub/tiktok/account-insights-overview
func TikHubAccountInsightsOverview(c *gin.Context) {
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

	// 解析请求体
	var reqBody struct {
		Cookie    string `json:"cookie"`
		StartDate string `json:"start_date"`
		Proxy     string `json:"proxy"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数解析失败",
		})
		return
	}

	if reqBody.Cookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "cookie 不能为空",
		})
		return
	}

	// 默认日期
	if reqBody.StartDate == "" {
		reqBody.StartDate = "04-01-2025"
	}

	body, err := service.FetchTikHubAccountInsightsOverview(c.Request.Context(), reqBody.Cookie, reqBody.StartDate, reqBody.Proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "account-insights-overview")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubVideoAnalyticsSummary 代理 TikHub 获取创作者视频概览
// POST /api/public/tikhub/tiktok/video-analytics-summary
func TikHubVideoAnalyticsSummary(c *gin.Context) {
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

	// 解析请求体
	var reqBody struct {
		Cookie string `json:"cookie"`
		Proxy  string `json:"proxy"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数解析失败",
		})
		return
	}

	if reqBody.Cookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "cookie 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubVideoAnalyticsSummary(c.Request.Context(), reqBody.Cookie, reqBody.Proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "video-analytics-summary")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubProductRelatedVideos 代理 TikHub 获取同款商品关联视频
// POST /api/public/tikhub/tiktok/product-related-videos
func TikHubProductRelatedVideos(c *gin.Context) {
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

	// 解析请求体
	var reqBody struct {
		Cookie    string `json:"cookie"`
		StartDate string `json:"start_date"`
		ItemID    string `json:"item_id"`
		ProductID string `json:"product_id"`
		Proxy     string `json:"proxy"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数解析失败",
		})
		return
	}

	if reqBody.Cookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "cookie 不能为空",
		})
		return
	}

	if reqBody.ItemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "item_id 不能为空",
		})
		return
	}

	if reqBody.ProductID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "product_id 不能为空",
		})
		return
	}

	// 默认日期
	if reqBody.StartDate == "" {
		reqBody.StartDate = "04-01-2025"
	}

	body, err := service.FetchTikHubProductRelatedVideos(c.Request.Context(), reqBody.Cookie, reqBody.StartDate, reqBody.ItemID, reqBody.ProductID, reqBody.Proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "product-related-videos")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubProductDetail 代理 TikHub 获取 TikTok 商品详情数据 V2
// GET /api/public/tikhub/tiktok/product?product_id=1729385239712731370
func TikHubProductDetail(c *gin.Context) {
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

	productID := c.Query("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "product_id 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubProductDetail(c.Request.Context(), productID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "product")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubTrendsHashtagList 代理 TikHub 获取热门标签榜单
// GET /api/public/tikhub/tiktok/trends-hashtag-list?time_range=7&country_code=US&page=1&limit=20
func TikHubTrendsHashtagList(c *gin.Context) {
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

	// time_range: 7/30/90，默认7
	timeRange := 7
	if tr := c.Query("time_range"); tr != "" {
		if parsed, err := strconv.Atoi(tr); err == nil && (parsed == 7 || parsed == 30 || parsed == 90) {
			timeRange = parsed
		}
	}

	// country_code: 国家代码，默认US
	countryCode := c.DefaultQuery("country_code", "US")

	// page: 页码，默认1
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// limit: 每页数量，默认20
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// industry_id: 行业ID，可选
	var industryID int64
	if i := c.Query("industry_id"); i != "" {
		if parsed, err := strconv.ParseInt(i, 10, 64); err == nil && parsed > 0 {
			industryID = parsed
		}
	}

	body, err := service.FetchTikHubTrendsHashtagList(c.Request.Context(), timeRange, countryCode, page, limit, industryID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "trends-hashtag-list")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubHotSellingProductsList 代理 TikHub 获取热卖商品列表
// GET /api/public/tikhub/tiktok/hot-selling-products-list?region=US&count=100
func TikHubHotSellingProductsList(c *gin.Context) {
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

	// region: 地区代码，默认 US
	region := c.DefaultQuery("region", "US")

	// count: 返回商品数量，默认 100
	count := 100
	if c := c.Query("count"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
			count = parsed
		}
	}

	body, err := service.FetchTikHubHotSellingProductsList(c.Request.Context(), region, count)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "hot-selling-products-list")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubVideoComments 代理 TikHub 获取单个视频评论数据
// GET /api/public/tikhub/tiktok/video-comments?aweme_id=xxx&cursor=0&count=20
func TikHubVideoComments(c *gin.Context) {
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

	awemeID := c.Query("aweme_id")
	if awemeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "aweme_id 不能为空",
		})
		return
	}

	// cursor: 分页游标
	cursor := 0
	if c := c.Query("cursor"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil {
			cursor = parsed
		}
	}

	// count: 数量
	count := 20
	if c := c.Query("count"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
			count = parsed
		}
	}

	body, err := service.FetchTikHubVideoComments(c.Request.Context(), awemeID, cursor, count)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "video-comments")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubVideoAudienceStats 代理 TikHub 获取视频受众分析数据
// POST /api/public/tikhub/tiktok/video-audience-stats
func TikHubVideoAudienceStats(c *gin.Context) {
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

	var reqBody struct {
		Cookie    string `json:"cookie"`
		StartDate string `json:"start_date"`
		ItemID    string `json:"item_id"`
		Proxy     string `json:"proxy"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数解析失败",
		})
		return
	}

	if reqBody.ItemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "item_id 不能为空",
		})
		return
	}

	if reqBody.Cookie == "" {
		reqBody.Cookie = c.GetHeader("Cookie")
	}

	// 默认日期
	if reqBody.StartDate == "" {
		reqBody.StartDate = "04-01-2025"
	}

	body, err := service.FetchTikHubVideoAudienceStats(c.Request.Context(), reqBody.Cookie, reqBody.StartDate, reqBody.ItemID, reqBody.Proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "video-audience-stats")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubPostComment 代理 TikHub 获取作品评论列表
// GET /api/public/tikhub/tiktok/post-comment?aweme_id=xxx&cursor=0&count=20
func TikHubPostComment(c *gin.Context) {
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

	awemeID := c.Query("aweme_id")
	if awemeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "aweme_id 不能为空",
		})
		return
	}

	// cursor: 分页游标
	cursor := 0
	if c := c.Query("cursor"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil {
			cursor = parsed
		}
	}

	// count: 数量
	count := 20
	if c := c.Query("count"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
			count = parsed
		}
	}

	// current_region: 当前地区
	currentRegion := c.Query("current_region")

	body, err := service.FetchTikHubPostComment(c.Request.Context(), awemeID, cursor, count, currentRegion)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "post-comment")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}
