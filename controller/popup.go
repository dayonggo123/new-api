package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetDailyPopup 返回当前生效的每日弹窗
// GET /api/popups/daily
func GetDailyPopup(c *gin.Context) {
	popup, err := model.GetActivePopup()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if popup == nil {
		c.JSON(http.StatusOK, gin.H{
			"id":         0,
			"title":      "",
			"content":    "",
			"image_url":  "",
			"type":       "none",
			"created_at": 0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         popup.Id,
		"title":      popup.Title,
		"content":    popup.Content,
		"image_url":  popup.ImageUrl,
		"type":       popup.Type,
		"created_at": popup.CreatedAt,
	})
}

// AdminListPopups 管理员获取所有弹窗
func AdminListPopups(c *gin.Context) {
	popups, err := model.GetAllPopups()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, popups)
}

// AdminCreatePopup 管理员创建弹窗
type CreatePopupRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	ImageUrl string `json:"image_url"`
	Type     string `json:"type"`
	Enabled  bool   `json:"enabled"`
}

func AdminCreatePopup(c *gin.Context) {
	var req CreatePopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	popup := &model.Popup{
		Title:    req.Title,
		Content:  req.Content,
		ImageUrl: req.ImageUrl,
		Type:     req.Type,
		Enabled:  req.Enabled,
	}
	if popup.Type == "" {
		popup.Type = "announcement"
	}

	if err := popup.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, popup)
}

// AdminUpdatePopup 管理员更新弹窗
type UpdatePopupRequest struct {
	Id       int    `json:"id" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	ImageUrl string `json:"image_url"`
	Type     string `json:"type"`
	Enabled  bool   `json:"enabled"`
}

func AdminUpdatePopup(c *gin.Context) {
	var req UpdatePopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	popup, err := model.GetPopupById(req.Id)
	if err != nil {
		common.ApiErrorMsg(c, "弹窗不存在")
		return
	}

	popup.Title = req.Title
	popup.Content = req.Content
	popup.ImageUrl = req.ImageUrl
	popup.Type = req.Type
	popup.Enabled = req.Enabled

	if err := popup.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, popup)
}

// AdminDeletePopup 管理员删除弹窗
func AdminDeletePopup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DeletePopup(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
