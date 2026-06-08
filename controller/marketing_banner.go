package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetMarketingBanners 返回当前生效的运营 banner 列表
// GET /api/marketing/banners
func GetMarketingBanners(c *gin.Context) {
	now := common.GetTimestamp()
	banners, err := model.GetActiveBanners(now)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	result := make([]gin.H, 0, len(banners))
	for _, b := range banners {
		var contentMap map[string]model.BannerContent
		if err := common.Unmarshal([]byte(b.Content), &contentMap); err != nil {
			// 解析失败时返回原始 JSON 字符串作为兜底
			result = append(result, gin.H{
				"id":                b.Id,
				"priority":          b.Priority,
				"enabled":           b.Enabled,
				"start_at":          b.StartAt,
				"end_at":            b.EndAt,
				"max_dismiss_hours": b.MaxDismissHours,
				"content":           b.Content,
			})
			continue
		}
		result = append(result, gin.H{
			"id":                b.Id,
			"priority":          b.Priority,
			"enabled":           b.Enabled,
			"start_at":          b.StartAt,
			"end_at":            b.EndAt,
			"max_dismiss_hours": b.MaxDismissHours,
			"content":           contentMap,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"banners": result,
	})
}

// AdminListBanners 管理员获取所有 banner
func AdminListBanners(c *gin.Context) {
	banners, err := model.GetAllBanners()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, banners)
}

// AdminCreateBanner 管理员创建 banner
type CreateBannerRequest struct {
	Priority        int                    `json:"priority"`
	Enabled         bool                   `json:"enabled"`
	StartAt         int64                  `json:"start_at"`
	EndAt           int64                  `json:"end_at"`
	MaxDismissHours int                    `json:"max_dismiss_hours"`
	Content         map[string]interface{} `json:"content" binding:"required"`
}

func AdminCreateBanner(c *gin.Context) {
	var req CreateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	contentBytes, err := common.Marshal(req.Content)
	if err != nil {
		common.ApiErrorMsg(c, "内容格式错误")
		return
	}

	banner := &model.MarketingBanner{
		Priority:        req.Priority,
		Enabled:         req.Enabled,
		StartAt:         req.StartAt,
		EndAt:           req.EndAt,
		MaxDismissHours: req.MaxDismissHours,
		Content:         string(contentBytes),
	}
	if banner.MaxDismissHours <= 0 {
		banner.MaxDismissHours = 24
	}

	if err := banner.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, banner)
}

// AdminUpdateBanner 管理员更新 banner
type UpdateBannerRequest struct {
	Id              int                    `json:"id" binding:"required"`
	Priority        int                    `json:"priority"`
	Enabled         bool                   `json:"enabled"`
	StartAt         int64                  `json:"start_at"`
	EndAt           int64                  `json:"end_at"`
	MaxDismissHours int                    `json:"max_dismiss_hours"`
	Content         map[string]interface{} `json:"content" binding:"required"`
}

func AdminUpdateBanner(c *gin.Context) {
	var req UpdateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	banner, err := model.GetBannerById(req.Id)
	if err != nil {
		common.ApiErrorMsg(c, "Banner 不存在")
		return
	}

	contentBytes, err := common.Marshal(req.Content)
	if err != nil {
		common.ApiErrorMsg(c, "内容格式错误")
		return
	}

	banner.Priority = req.Priority
	banner.Enabled = req.Enabled
	banner.StartAt = req.StartAt
	banner.EndAt = req.EndAt
	banner.MaxDismissHours = req.MaxDismissHours
	if banner.MaxDismissHours <= 0 {
		banner.MaxDismissHours = 24
	}
	banner.Content = string(contentBytes)

	if err := banner.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, banner)
}

// AdminDeleteBanner 管理员删除 banner
func AdminDeleteBanner(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DeleteBanner(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
