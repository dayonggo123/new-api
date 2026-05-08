package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ==================== Admin Tag APIs ====================

// AdminGetTags 获取所有标签
func AdminGetTags(c *gin.Context) {
	tags, err := model.GetAllTags()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tags,
	})
}

// AdminCreateTag 创建标签
func AdminCreateTag(c *gin.Context) {
	var tag model.Tag
	if err := c.ShouldBindJSON(&tag); err != nil {
		common.ApiError(c, err)
		return
	}
	if tag.Name == "" {
		common.ApiErrorMsg(c, "标签名称不能为空")
		return
	}
	if err := model.CreateTag(&tag); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tag,
	})
}

// AdminDeleteTag 删除标签
func AdminDeleteTag(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteTag(id); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ==================== Admin User Tag APIs ====================

// AdminGetUserTags 获取用户标签
func AdminGetUserTags(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tags, err := model.GetUserTags(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tags,
	})
}

// AdminSetUserTags 设置用户标签（全量替换）
func AdminSetUserTags(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req struct {
		TagIds []int `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SetUserTags(userId, req.TagIds, "manual"); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// AdminAddUserTag 给用户添加单个标签
func AdminAddUserTag(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req struct {
		TagId int `json:"tag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.AddUserTag(userId, req.TagId, "manual"); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// AdminRemoveUserTag 移除用户单个标签
func AdminRemoveUserTag(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tagId, err := strconv.Atoi(c.Param("tag_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RemoveUserTag(userId, tagId); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
