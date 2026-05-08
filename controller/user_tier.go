package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ==================== Admin Tier APIs ====================

// AdminGetTiers 获取所有层级
func AdminGetTiers(c *gin.Context) {
	tiers, err := model.GetAllTiers()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tiers,
	})
}

// AdminCreateTier 创建层级
func AdminCreateTier(c *gin.Context) {
	var tier model.UserTier
	if err := c.ShouldBindJSON(&tier); err != nil {
		common.ApiError(c, err)
		return
	}
	if tier.Level < 1 || tier.Level > 10 {
		common.ApiErrorMsg(c, "层级必须在 1-10 之间")
		return
	}
	if tier.Name == "" {
		common.ApiErrorMsg(c, "层级名称不能为空")
		return
	}
	if err := model.CreateTier(&tier); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tier,
	})
}

// AdminUpdateTier 更新层级
func AdminUpdateTier(c *gin.Context) {
	var tier model.UserTier
	if err := c.ShouldBindJSON(&tier); err != nil {
		common.ApiError(c, err)
		return
	}
	if tier.Id == 0 {
		common.ApiErrorMsg(c, "层级 ID 不能为空")
		return
	}
	if err := model.UpdateTier(&tier); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// AdminDeleteTier 删除层级
func AdminDeleteTier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteTier(id); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// AdminSetUserTier 设置用户层级
func AdminSetUserTier(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req struct {
		TierLevel int `json:"tier_level" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SetUserTierLevel(userId, req.TierLevel); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// AdminAutoUpdateUserTier 根据积分自动更新用户层级
func AdminAutoUpdateUserTier(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.AutoUpdateUserTierByPoints(userId); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
