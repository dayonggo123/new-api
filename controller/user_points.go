package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// ==================== User Points & Signin ====================

// GetUserPoints 获取用户积分和签到状态
func GetUserPoints(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	up, err := model.GetOrCreateUserPoints(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 计算今天是否已签到
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	todaySigned := up.LastSigninDate >= todayStart

	// 计算下次签到可获得的积分
	consecutiveDays := up.ConsecutiveDays
	if !todaySigned {
		// 如果上次签到不是昨天，连续天数会重置为1
		if up.LastSigninDate > 0 {
			yesterdayStart := todayStart - 86400
			if up.LastSigninDate < yesterdayStart {
				consecutiveDays = 0
			}
		} else {
			consecutiveDays = 0
		}
	}
	nextDay := consecutiveDays + 1
	if todaySigned {
		nextDay = consecutiveDays
	}
	basePoints := operation_setting.GetSigninBasePoints()
	nextBonus := operation_setting.GetConsecutiveBonus(nextDay)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_points":        up.TotalPoints,
			"consecutive_days":    up.ConsecutiveDays,
			"last_signin_date":    up.LastSigninDate,
			"today_signed":        todaySigned,
			"next_signin_points":  basePoints + nextBonus,
		},
	})
}

// DoSignin 执行签到
func DoSignin(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	result, err := service.UserSignin(userId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetSigninHistory 获取签到历史
func GetSigninHistory(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	pageInfo := common.GetPageQuery(c)
	histories, total, err := model.GetSigninHistoryByUserId(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(histories)
	common.ApiSuccess(c, pageInfo)
}

// ==================== Prompt Unlock ====================

// UnlockPrompt 解锁提示词
func UnlockPrompt(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	var req struct {
		PromptId int `json:"prompt_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.UnlockPrompt(userId, req.PromptId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetUnlockedPrompts 获取已解锁提示词列表
func GetUnlockedPrompts(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "用户未登录")
		return
	}

	pageInfo := common.GetPageQuery(c)
	unlocked, total, err := model.GetUnlockedPromptByUserId(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 获取提示词详情
	var items []gin.H
	for _, u := range unlocked {
		prompt, err := model.GetPromptById(u.PromptId)
		if err != nil {
			continue
		}
		items = append(items, gin.H{
			"prompt_id":    u.PromptId,
			"title":        prompt.Title,
			"cover_image_url": prompt.CoverImageUrl,
			"cost":         u.Cost,
			"unlocked_at":  u.UnlockedAt,
		})
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// ==================== Admin: Points Management ====================

// AdminAdjustUserPoints 管理员调整用户积分
func AdminAdjustUserPoints(c *gin.Context) {
	var req struct {
		UserId      int    `json:"user_id" binding:"required"`
		Amount      int    `json:"amount" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	newTotal, err := service.AdjustUserPoints(req.UserId, req.Amount, req.Description)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":      req.UserId,
			"new_total":    newTotal,
			"adjust_amount": req.Amount,
		},
	})
}

// AdminGetUserPointsTransactions 管理员获取用户积分流水
func AdminGetUserPointsTransactions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	pageInfo := common.GetPageQuery(c)

	var transactions []*model.UserPointsTransaction
	var total int64
	var err error

	if userId > 0 {
		transactions, total, err = model.GetPointsTransactionsByUserId(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	} else {
		// 获取所有流水
		err = model.DB.Model(&model.UserPointsTransaction{}).Count(&total).Error
		if err == nil {
			err = model.DB.Order("created_time DESC").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&transactions).Error
		}
	}

	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(transactions)
	common.ApiSuccess(c, pageInfo)
}

// AdminGetSigninStats 管理员获取签到统计
func AdminGetSigninStats(c *gin.Context) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	yesterdayStart := todayStart - 86400

	var todayCount int64
	var yesterdayCount int64
	model.DB.Model(&model.UserSigninHistory{}).Where("signin_date = ?", todayStart).Count(&todayCount)
	model.DB.Model(&model.UserSigninHistory{}).Where("signin_date = ?", yesterdayStart).Count(&yesterdayCount)

	// 连续签到分布
	type Distribution struct {
		Days  int `json:"days"`
		Count int `json:"count"`
	}

	// 查询各连续天数的用户数
	var distributions []Distribution
	model.DB.Raw("SELECT consecutive_days as days, COUNT(*) as count FROM user_points WHERE consecutive_days > 0 GROUP BY consecutive_days ORDER BY consecutive_days").Scan(&distributions)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"today_signin_count":        todayCount,
			"yesterday_signin_count":    yesterdayCount,
			"consecutive_distribution":  distributions,
		},
	})
}
