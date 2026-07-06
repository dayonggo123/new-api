package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// GetCuratedCategories 获取分类列表（公开）
func GetCuratedCategories(c *gin.Context) {
	resp, err := service.GetCuratedCategories()
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// AdminListCuratedCategories 管理后台获取全部分类
func AdminListCuratedCategories(c *gin.Context) {
	categories, err := service.AdminListCuratedCategories()
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, categories)
}

// AdminCreateCuratedCategory 管理后台创建分类
func AdminCreateCuratedCategory(c *gin.Context) {
	var req dto.AdminUpsertCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	if req.Key == "" || req.Name == "" {
		common.ApiErrorMsg(c, "key and name are required")
		return
	}
	category, err := service.AdminCreateCategory(&req)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, category)
}

// AdminUpdateCuratedCategory 管理后台更新分类
func AdminUpdateCuratedCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid category id")
		return
	}
	var req dto.AdminUpsertCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	if err := service.AdminUpdateCategory(id, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminDeleteCuratedCategory 管理后台删除分类
func AdminDeleteCuratedCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid category id")
		return
	}
	if err := service.AdminDeleteCategory(id); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, nil)
}
