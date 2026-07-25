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
// 根据用户等级（普通/VIP/SVIP/管理员）计算价格
func chargeTikHubIfEnabled(c *gin.Context, endpoint string) bool {
	userID := c.GetInt("id")
	if userID == 0 {
		return false
	}

	config, err := model.GetTikHubPriceConfigByEndpoint(endpoint)
	if err != nil || config == nil {
		return false
	}

	// 获取用户等级
	tier := model.GetUserTikHubTier(userID)

	// 根据用户等级获取价格
	price, quota, freeQuota, shouldCharge := config.GetTikHubPriceWithTier(tier)

	// 不需要扣费的情况（仅当有免费额度且未用完时）
	if !shouldCharge && freeQuota > 0 {
		tierName := map[string]string{
			"root":   "管理员",
			"admin":  "管理员",
			"svip":   "SVIP",
			"vip":    "VIP",
			"common": "普通用户",
		}[tier]

		logger.LogInfo(c.Request.Context(), fmt.Sprintf("[TikHub] 用户 %d (%s) 调用接口 %s，免费条数: %d", userID, tierName, endpoint, freeQuota))
		return false
	}

	// 扣除积分
	err = model.DecreaseUserQuota(userID, quota, false)
	if err != nil {
		logger.LogError(c.Request.Context(), "TikHub扣费失败: "+err.Error())
		return false
	}

	// 记录使用日志到数据库
	tierName := map[string]string{
		"svip":   "SVIP",
		"vip":    "VIP",
		"common": "普通用户",
	}[tier]
	logContent := fmt.Sprintf("TikHub接口 %s (%s %.2f USD)", config.Name, tierName, price)
	model.RecordLog(userID, model.LogTypeConsume, logContent)

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("[TikHub] 用户 %d (%s) 调用接口 %s，消费 %.2f USD (%d 积分)", userID, tierName, endpoint, price, quota))
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

// TikHubUserCountryByUsername 代理 TikHub 通过用户名获取用户账号国家地区
// GET /api/public/tikhub/tiktok/user-country-by-username?username=xxx
func TikHubUserCountryByUsername(c *gin.Context) {
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

	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "username 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubUserCountryByUsername(c.Request.Context(), username)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "user-country-by-username")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubGeneralSearchResult 代理 TikHub 获取综合搜索结果
// GET /api/public/tikhub/tiktok/general-search-result
func TikHubGeneralSearchResult(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "keyword 不能为空"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	sortType, _ := strconv.Atoi(c.DefaultQuery("sort_type", "0"))
	publishTime, _ := strconv.Atoi(c.DefaultQuery("publish_time", "0"))

	body, err := service.FetchTikHubGeneralSearchResult(c.Request.Context(), keyword, offset, count, sortType, publishTime)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "general-search-result")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubVideoSearchResult 代理 TikHub 获取视频搜索结果
// GET /api/public/tikhub/tiktok/video-search-result
func TikHubVideoSearchResult(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "keyword 不能为空"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	sortType, _ := strconv.Atoi(c.DefaultQuery("sort_type", "0"))
	publishTime, _ := strconv.Atoi(c.DefaultQuery("publish_time", "0"))
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubVideoSearchResult(c.Request.Context(), keyword, offset, count, sortType, publishTime, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "video-search-result")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubUserSearchResult 代理 TikHub 获取用户搜索结果
// GET /api/public/tikhub/tiktok/user-search-result
func TikHubUserSearchResult(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "keyword 不能为空"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	followerCount := c.Query("user_search_follower_count")
	profileType := c.Query("user_search_profile_type")
	otherPref := c.Query("user_search_other_pref")

	body, err := service.FetchTikHubUserSearchResult(c.Request.Context(), keyword, offset, count, followerCount, profileType, otherPref)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "user-search-result")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubMusicSearchResult 代理 TikHub 获取音乐搜索结果
// GET /api/public/tikhub/tiktok/music-search-result
func TikHubMusicSearchResult(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "keyword 不能为空"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	filterBy, _ := strconv.Atoi(c.DefaultQuery("filter_by", "0"))
	sortType, _ := strconv.Atoi(c.DefaultQuery("sort_type", "0"))
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubMusicSearchResult(c.Request.Context(), keyword, offset, count, filterBy, sortType, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "music-search-result")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubHashtagSearchResult 代理 TikHub 获取话题搜索结果
// GET /api/public/tikhub/tiktok/hashtag-search-result
func TikHubHashtagSearchResult(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "keyword 不能为空"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))

	body, err := service.FetchTikHubHashtagSearchResult(c.Request.Context(), keyword, offset, count)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "hashtag-search-result")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubMusicDetail 代理 TikHub 获取音乐详情
// GET /api/public/tikhub/tiktok/music-detail
func TikHubMusicDetail(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	musicID := c.Query("music_id")
	if musicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "music_id 不能为空"})
		return
	}

	body, err := service.FetchTikHubMusicDetail(c.Request.Context(), musicID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "music-detail")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubMusicVideoList 代理 TikHub 获取音乐视频列表
// GET /api/public/tikhub/tiktok/music-video-list
func TikHubMusicVideoList(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	musicID := c.Query("music_id")
	if musicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "music_id 不能为空"})
		return
	}

	cursor, _ := strconv.Atoi(c.DefaultQuery("cursor", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))

	body, err := service.FetchTikHubMusicVideoList(c.Request.Context(), musicID, cursor, count)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "music-video-list")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubHashtagDetail 代理 TikHub 获取话题详情
// GET /api/public/tikhub/tiktok/hashtag-detail
func TikHubHashtagDetail(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	chID := c.Query("ch_id")
	if chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ch_id 不能为空"})
		return
	}

	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubHashtagDetail(c.Request.Context(), chID, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "hashtag-detail")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubHashtagVideoList 代理 TikHub 获取话题视频列表
// GET /api/public/tikhub/tiktok/hashtag-video-list
func TikHubHashtagVideoList(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	chID := c.Query("ch_id")
	if chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ch_id 不能为空"})
		return
	}

	cursor, _ := strconv.Atoi(c.DefaultQuery("cursor", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubHashtagVideoList(c.Request.Context(), chID, cursor, count, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "hashtag-video-list")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubCreatorSearchInsights 代理 TikHub 获取创作者搜索洞察
// GET /api/public/tikhub/tiktok/creator-search-insights
func TikHubCreatorSearchInsights(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	tab := c.DefaultQuery("tab", "all")
	languageFilters := c.DefaultQuery("language_filters", "en")
	categoryFilters := c.Query("category_filters")
	creatorSource := c.DefaultQuery("creator_source", "general_search")
	forceRefresh := c.Query("force_refresh") == "true"

	body, err := service.FetchTikHubCreatorSearchInsights(c.Request.Context(), offset, limit, tab, languageFilters, categoryFilters, creatorSource, forceRefresh)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "creator-search-insights")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubCreatorSearchInsightsDetail 代理 TikHub 获取创作者搜索洞察详情
// GET /api/public/tikhub/tiktok/creator-search-insights-detail
func TikHubCreatorSearchInsightsDetail(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	queryIDStr := c.Query("query_id_str")
	if queryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "query_id_str 不能为空"})
		return
	}

	timeRange := c.DefaultQuery("time_range", "past_30_days")
	startDate, _ := strconv.ParseInt(c.Query("start_date"), 10, 64)
	endDate, _ := strconv.ParseInt(c.Query("end_date"), 10, 64)
	dimensionList := c.DefaultQuery("dimension_list", "gender,age,country")

	body, err := service.FetchTikHubCreatorSearchInsightsDetail(c.Request.Context(), queryIDStr, timeRange, startDate, endDate, dimensionList)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "creator-search-insights-detail")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubCreatorSearchInsightsVideos 代理 TikHub 获取创作者搜索洞察相关视频
// GET /api/public/tikhub/tiktok/creator-search-insights-videos
func TikHubCreatorSearchInsightsVideos(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "keyword 不能为空"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))

	body, err := service.FetchTikHubCreatorSearchInsightsVideos(c.Request.Context(), keyword, offset, count)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "creator-search-insights-videos")
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

// TikHubVideoListAnalytics 代理 TikHub 获取创作者视频列表分析
// POST /api/public/tikhub/tiktok/video-list-analytics
func TikHubVideoListAnalytics(c *gin.Context) {
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
		Rules     string `json:"rules"`
		Proxy     string `json:"proxy"`
		Page      int    `json:"page"`
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

	// 默认排序规则
	if reqBody.Rules == "" {
		reqBody.Rules = "VIDEO_LIST_PUBLISH_TIME"
	}

	// 默认页码
	if reqBody.Page < 0 {
		reqBody.Page = 0
	}

	body, err := service.FetchTikHubVideoListAnalytics(c.Request.Context(), reqBody.Cookie, reqBody.StartDate, reqBody.Rules, reqBody.Proxy, reqBody.Page)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "video-list-analytics")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubProductAnalyticsList 代理 TikHub 获取创作者商品列表分析
// POST /api/public/tikhub/tiktok/product-analytics-list
func TikHubProductAnalyticsList(c *gin.Context) {
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
		EndDate   string `json:"end_date"`
		Proxy     string `json:"proxy"`
		Page      int    `json:"page"`
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
		reqBody.StartDate = "2025-04-01"
	}
	if reqBody.EndDate == "" {
		reqBody.EndDate = "2025-05-01"
	}

	// 默认页码
	if reqBody.Page < 0 {
		reqBody.Page = 0
	}

	body, err := service.FetchTikHubProductAnalyticsList(c.Request.Context(), reqBody.Cookie, reqBody.StartDate, reqBody.EndDate, reqBody.Proxy, reqBody.Page)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "product-analytics-list")

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

// TikHubProductDetailV1 代理 TikHub 获取 TikTok 商品详情数据 V1 (桌面端-数据完整)
// GET /api/public/tikhub/tiktok/shop-product-detail?product_id=xxx&seller_id=xxx&region=xxx
func TikHubProductDetailV1(c *gin.Context) {
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

	sellerID := c.Query("seller_id")
	region := c.DefaultQuery("region", "MY")

	body, err := service.FetchTikHubProductDetailV1(c.Request.Context(), productID, sellerID, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "shop-product-detail")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubProductReviewsV1 代理 TikHub 获取商品评论 V1
// GET /api/public/tikhub/tiktok/product-reviews?product_id=xxx
func TikHubProductReviewsV1(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	productID := c.Query("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "product_id 不能为空"})
		return
	}

	pageStart, _ := strconv.Atoi(c.DefaultQuery("page_start", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	sortRule, _ := strconv.Atoi(c.DefaultQuery("sort_rule", "1"))
	filterType, _ := strconv.Atoi(c.DefaultQuery("filter_type", "1"))
	filterValue, _ := strconv.Atoi(c.DefaultQuery("filter_value", "6"))
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubProductReviewsV1(c.Request.Context(), productID, pageStart, pageSize, sortRule, filterType, filterValue, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "product-reviews-v1")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubProductReviewsV2 代理 TikHub 获取商品评论 V2
// GET /api/public/tikhub/tiktok/product-reviews-v2?product_id=xxx
func TikHubProductReviewsV2(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	productID := c.Query("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "product_id 不能为空"})
		return
	}

	pageStart, _ := strconv.Atoi(c.DefaultQuery("page_start", "1"))
	sortRule, _ := strconv.Atoi(c.DefaultQuery("sort_rule", "2"))
	filterType, _ := strconv.Atoi(c.DefaultQuery("filter_type", "1"))
	filterValue, _ := strconv.Atoi(c.DefaultQuery("filter_value", "6"))
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubProductReviewsV2(c.Request.Context(), productID, pageStart, sortRule, filterType, filterValue, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "product-reviews-v2")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubSellerProductsList 代理 TikHub 获取商家商品列表 V1
// GET /api/public/tikhub/tiktok/seller-products-list?seller_id=xxx
func TikHubSellerProductsList(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "seller_id 不能为空"})
		return
	}

	searchParams := c.Query("search_params")
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubSellerProductsList(c.Request.Context(), sellerID, searchParams, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "seller-products-list")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubSearchProductsList 代理 TikHub 搜索商品列表 V1
// GET /api/public/tikhub/tiktok/search-products-list?search_word=xxx
func TikHubSearchProductsList(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	searchWord := c.Query("search_word")
	if searchWord == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "search_word 不能为空"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	pageToken := c.Query("page_token")
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchTikHubSearchProductsList(c.Request.Context(), searchWord, offset, pageToken, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "search-products-list")
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubHotSellingProductsListV1 代理 TikHub 获取热卖商品列表
// GET /api/public/tikhub/tiktok/hot-selling-products-list-v1?region=US&count=100
func TikHubHotSellingProductsListV1(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub 接口未启用"})
		return
	}
	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "TikHub API Key 未配置"})
		return
	}

	region := c.DefaultQuery("region", "US")
	count, _ := strconv.Atoi(c.DefaultQuery("count", "100"))

	body, err := service.FetchTikHubHotSellingProductsListV1(c.Request.Context(), region, count)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}

	chargeTikHubIfEnabled(c, "hot-selling-products-list-v1")
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

// TikHubVideoMetrics 代理 TikHub 获取视频统计数据
// GET /api/public/tikhub/tiktok/video-metrics?item_id=xxx
func TikHubVideoMetrics(c *gin.Context) {
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

	body, err := service.FetchTikHubVideoMetrics(c.Request.Context(), itemID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "video-metrics")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubDetectFakeViews 代理 TikHub 检测视频虚假流量分析
// GET /api/public/tikhub/tiktok/detect-fake-views?item_id=xxx&content_category=xxx
func TikHubDetectFakeViews(c *gin.Context) {
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

	contentCategory := c.DefaultQuery("content_category", "default")

	body, err := service.FetchTikHubDetectFakeViews(c.Request.Context(), itemID, contentCategory)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "detect-fake-views")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubCreatorInfoAndMilestones 代理 TikHub 获取创作者信息和里程碑数据
// GET /api/public/tikhub/tiktok/creator-info-milestones?user_id=xxx
func TikHubCreatorInfoAndMilestones(c *gin.Context) {
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

	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "user_id 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubCreatorInfoAndMilestones(c.Request.Context(), userID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "creator-info-milestones")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubAccountViolationList 代理 TikHub 获取创作者账号违规记录列表
// POST /api/public/tikhub/tiktok/account-violation-list
func TikHubAccountViolationList(c *gin.Context) {
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
		Cookie string `json:"cookie"`
		Proxy   string `json:"proxy"`
		Page    int    `json:"page"`
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

	body, err := service.FetchTikHubAccountViolationList(c.Request.Context(), reqBody.Cookie, reqBody.Proxy, reqBody.Page)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "account-violation-list")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// =============================================================================
// TikTok Ads API - 广告搜索与分析
// =============================================================================

// TikHubSearchAds 代理 TikHub 搜索广告
// GET /api/public/tikhub/tiktok/ads/search-ads
func TikHubSearchAds(c *gin.Context) {
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

	// objective: 广告目标类型，默认 1
	objective, _ := strconv.Atoi(c.DefaultQuery("objective", "1"))

	// like: 表现排名，默认 1
	like, _ := strconv.Atoi(c.DefaultQuery("like", "1"))

	// period: 时间段(天)，默认 180
	period, _ := strconv.Atoi(c.DefaultQuery("period", "180"))

	// industry: 行业ID
	industry, _ := strconv.Atoi(c.Query("industry"))

	// keyword: 搜索关键词
	keyword := c.Query("keyword")

	// page: 页码，默认 1
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	// limit: 每页数量，默认 20，最大 50
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	// order_by: 排序方式，默认 for_you
	orderBy := c.DefaultQuery("order_by", "for_you")

	// country_code: 国家代码，默认 US
	countryCode := c.DefaultQuery("country_code", "US")

	// ad_format: 广告格式，默认 1 (视频)
	adFormat, _ := strconv.Atoi(c.DefaultQuery("ad_format", "1"))

	// ad_language: 广告语言，默认 en
	adLanguage := c.DefaultQuery("ad_language", "en")

	// search_id: 搜索ID (可选)
	searchID := c.Query("search_id")

	body, err := service.FetchTikHubSearchAds(c.Request.Context(), keyword, objective, like, period, industry, page, limit, orderBy, countryCode, adFormat, adLanguage, searchID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "ads-search-ads")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubTopAdsSpotlight 代理 TikHub 获取热门广告聚光灯
// GET /api/public/tikhub/tiktok/ads/top-ads-spotlight
func TikHubTopAdsSpotlight(c *gin.Context) {
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

	// industry: 行业ID，可选
	industry := c.Query("industry")

	// page: 页码，默认 1
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	// limit: 每页数量，默认 20
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	body, err := service.FetchTikHubTopAdsSpotlight(c.Request.Context(), industry, page, limit)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "ads-top-ads-spotlight")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubAdKeyframeAnalysis 代理 TikHub 获取广告关键帧分析
// GET /api/public/tikhub/tiktok/ads/ad-keyframe-analysis
func TikHubAdKeyframeAnalysis(c *gin.Context) {
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

	// material_id: 广告素材ID，必填
	materialID := c.Query("material_id")
	if materialID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "material_id 不能为空",
		})
		return
	}

	// metric: 分析指标，默认 retain_ctr
	metric := c.DefaultQuery("metric", "retain_ctr")

	body, err := service.FetchTikHubAdKeyframeAnalysis(c.Request.Context(), materialID, metric)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "ads-ad-keyframe-analysis")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubAdPercentile 代理 TikHub 获取广告百分位数据
// GET /api/public/tikhub/tiktok/ads/ad-percentile
func TikHubAdPercentile(c *gin.Context) {
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

	// material_id: 广告素材ID，必填
	materialID := c.Query("material_id")
	if materialID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "material_id 不能为空",
		})
		return
	}

	// metric: 分析指标，默认 ctr_percentile
	metric := c.DefaultQuery("metric", "ctr_percentile")

	// period_type: 时间范围(天)，默认 180
	periodType, _ := strconv.Atoi(c.DefaultQuery("period_type", "180"))

	body, err := service.FetchTikHubAdPercentile(c.Request.Context(), materialID, metric, periodType)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "ads-ad-percentile")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubAdInteractiveAnalysis 代理 TikHub 获取广告互动分析
// GET /api/public/tikhub/tiktok/ads/ad-interactive-analysis
func TikHubAdInteractiveAnalysis(c *gin.Context) {
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

	// material_id: 广告素材ID，必填
	materialID := c.Query("material_id")
	if materialID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "material_id 不能为空",
		})
		return
	}

	// metric_type: 分析类型，默认 remain
	metricType := c.DefaultQuery("metric_type", "remain")

	// period_type: 时间范围，默认 180
	periodType, _ := strconv.Atoi(c.DefaultQuery("period_type", "180"))

	body, err := service.FetchTikHubAdInteractiveAnalysis(c.Request.Context(), materialID, metricType, periodType)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "ads-ad-interactive-analysis")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubTrendsHashtagDetail 代理 TikHub 获取热门标签详情
// GET /api/public/tikhub/tiktok/ads/trends-hashtag-detail
func TikHubTrendsHashtagDetail(c *gin.Context) {
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

	// hashtag_id: 标签ID，必填
	hashtagID := c.Query("hashtag_id")
	if hashtagID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "hashtag_id 不能为空",
		})
		return
	}

	// time_range: 时间范围(天)，默认 90
	timeRange := 90
	if tr := c.Query("time_range"); tr != "" {
		if parsed, err := strconv.Atoi(tr); err == nil && (parsed == 7 || parsed == 30 || parsed == 90) {
			timeRange = parsed
		}
	}

	// country_code: 国家代码，默认 US
	countryCode := c.DefaultQuery("country_code", "US")

	body, err := service.FetchTikHubTrendsHashtagDetail(c.Request.Context(), hashtagID, timeRange, countryCode)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "ads-trends-hashtag-detail")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// =============================================================================
// 整合报告 API
// =============================================================================

// TikHubProductAnalysisReport 代理 TikHub 获取商品分析报告
// GET /api/public/tikhub/report/product-analysis
func TikHubProductAnalysisReport(c *gin.Context) {
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

	// product_id: 商品ID，必填
	productID := c.Query("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "product_id 不能为空",
		})
		return
	}

	// region: 地区代码，默认 US
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchProductAnalysisReport(c.Request.Context(), productID, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "report-product-analysis")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubCreatorDiagnosisReport 代理 TikHub 获取创作者诊断报告
// POST /api/public/tikhub/report/creator-diagnosis
func TikHubCreatorDiagnosisReport(c *gin.Context) {
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

	body, err := service.FetchCreatorDiagnosisReport(c.Request.Context(), reqBody.Cookie, reqBody.Proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "report-creator-diagnosis")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubAdCreativeAnalysisReport 代理 TikHub 获取广告创意分析报告
// GET /api/public/tikhub/report/ad-creative-analysis
func TikHubAdCreativeAnalysisReport(c *gin.Context) {
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

	// material_id: 广告素材ID，必填
	materialID := c.Query("material_id")
	if materialID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "material_id 不能为空",
		})
		return
	}

	// industry: 行业ID，可选
	industry := c.Query("industry")

	body, err := service.FetchAdCreativeAnalysisReport(c.Request.Context(), materialID, industry)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "report-ad-creative-analysis")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubContentTrendsReport 代理 TikHub 获取内容趋势报告
// GET /api/public/tikhub/report/content-trends
func TikHubContentTrendsReport(c *gin.Context) {
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

	// country_code: 国家代码，默认 US
	countryCode := c.DefaultQuery("country_code", "US")

	// time_range: 时间范围(天)，默认 7
	timeRange := 7
	if tr := c.Query("time_range"); tr != "" {
		if parsed, err := strconv.Atoi(tr); err == nil && (parsed == 7 || parsed == 30 || parsed == 90) {
			timeRange = parsed
		}
	}

	// industry_id: 行业ID，可选
	industryID := 0
	if iid := c.Query("industry_id"); iid != "" {
		if parsed, err := strconv.Atoi(iid); err == nil {
			industryID = parsed
		}
	}

	body, err := service.FetchContentTrendsReport(c.Request.Context(), countryCode, timeRange, industryID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "report-content-trends")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubVideoAnalysisReport 代理 TikHub 获取视频深度分析报告
// GET /api/public/tikhub/report/video-analysis
func TikHubVideoAnalysisReport(c *gin.Context) {
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

	// aweme_id: 视频ID，必填
	awemeID := c.Query("aweme_id")
	if awemeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "aweme_id 不能为空",
		})
		return
	}

	// cookie: 可选，用于获取受众分析
	cookie := c.Query("cookie")
	proxy := c.Query("proxy")

	body, err := service.FetchVideoAnalysisReport(c.Request.Context(), awemeID, cookie, proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "report-video-analysis")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubCompetitorMonitorReport 代理 TikHub 获取竞品监控报告
// GET /api/public/tikhub/report/competitor-monitor
func TikHubCompetitorMonitorReport(c *gin.Context) {
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

	// seller_id: 商家ID，必填
	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "seller_id 不能为空",
		})
		return
	}

	// region: 地区代码，默认 US
	region := c.DefaultQuery("region", "US")

	body, err := service.FetchCompetitorMonitorReport(c.Request.Context(), sellerID, region)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 扣费
	chargeTikHubIfEnabled(c, "report-competitor-monitor")

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}
