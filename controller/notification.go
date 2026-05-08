package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== User Notification APIs ====================

// GetNotifications 分页获取用户消息列表
func GetNotifications(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	pageInfo := common.GetPageQuery(c)
	notifications, total, err := model.GetNotificationsByUserId(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(notifications)
	common.ApiSuccess(c, pageInfo)
}

// GetUnreadNotificationCount 获取未读消息数
func GetUnreadNotificationCount(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	count, err := model.GetUnreadNotificationCount(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"unread_count": count,
		},
	})
}

// MarkNotificationRead 标记单条消息已读
func MarkNotificationRead(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := model.MarkNotificationAsRead(userId, id); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// MarkAllNotificationsRead 标记全部消息已读
func MarkAllNotificationsRead(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	markedCount, err := model.MarkAllNotificationsAsRead(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"marked_count": markedCount,
		},
	})
}

// ==================== Admin Notification APIs ====================

// AdminSendNotification 管理员发送通知
func AdminSendNotification(c *gin.Context) {
	var req service.SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.SendNotification(&req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "发送成功",
	})
}

// AdminGetNotifications 管理员获取所有通知列表
func AdminGetNotifications(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	notificationType := c.Query("type")

	notifications, total, err := model.GetAllNotifications(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), notificationType)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(notifications)
	common.ApiSuccess(c, pageInfo)
}
