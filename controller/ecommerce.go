package controller

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== Public API (TryUserAuth) ====================

func GetEnabledModelPoses(c *gin.Context) {
	poses, err := service.GetEnabledModelPoses()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, poses)
}

func GetEnabledCaseCategories(c *gin.Context) {
	categories, err := service.GetEnabledCaseCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

func GetCaseDetail(c *gin.Context) {
	categoryId := c.Query("category")
	platformId := c.Query("platform")
	if categoryId == "" || platformId == "" {
		common.ApiErrorMsg(c, "category and platform are required")
		return
	}
	detail, err := service.GetCaseDetailByCategoryAndPlatform(categoryId, platformId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

// ==================== Admin: Model Pose ====================

func GetAllModelPoses(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	poses, total, err := service.GetAllModelPoses(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(poses)
	common.ApiSuccess(c, pageInfo)
}

func CreateModelPose(c *gin.Context) {
	pose := model.EcommerceModelPose{}
	err := c.ShouldBindJSON(&pose)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(pose.PoseId) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "pose_id cannot be empty"})
		return
	}
	if utf8.RuneCountInString(pose.Label) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "label cannot be empty"})
		return
	}
	pose.CreatedTime = common.GetTimestamp()
	pose.UpdatedTime = common.GetTimestamp()
	err = service.CreateModelPose(&pose)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    pose,
	})
}

func UpdateModelPose(c *gin.Context) {
	pose := model.EcommerceModelPose{}
	err := c.ShouldBindJSON(&pose)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanPose, err := service.GetModelPoseById(pose.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanPose.PoseId = pose.PoseId
	cleanPose.Label = pose.Label
	cleanPose.Description = pose.Description
	cleanPose.CoverImageUrl = pose.CoverImageUrl
	cleanPose.SortOrder = pose.SortOrder
	cleanPose.Status = pose.Status
	cleanPose.UpdatedTime = common.GetTimestamp()
	err = service.UpdateModelPose(cleanPose)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanPose,
	})
}

func DeleteModelPose(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := service.DeleteModelPoseById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// ==================== Admin: Case Category ====================

func GetAllCaseCategories(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	categories, total, err := service.GetAllCaseCategories(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(categories)
	common.ApiSuccess(c, pageInfo)
}

func CreateCaseCategory(c *gin.Context) {
	category := model.EcommerceCaseCategory{}
	err := c.ShouldBindJSON(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(category.CategoryId) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "category_id cannot be empty"})
		return
	}
	if utf8.RuneCountInString(category.CategoryName) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "category_name cannot be empty"})
		return
	}
	category.CreatedTime = common.GetTimestamp()
	category.UpdatedTime = common.GetTimestamp()
	err = service.CreateCaseCategory(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    category,
	})
}

func UpdateCaseCategory(c *gin.Context) {
	category := model.EcommerceCaseCategory{}
	err := c.ShouldBindJSON(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanCategory, err := service.GetCaseCategoryById(category.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanCategory.CategoryId = category.CategoryId
	cleanCategory.CategoryName = category.CategoryName
	cleanCategory.CoverImageUrl = category.CoverImageUrl
	cleanCategory.RequiresModel = category.RequiresModel
	cleanCategory.SortOrder = category.SortOrder
	cleanCategory.Status = category.Status
	cleanCategory.UpdatedTime = common.GetTimestamp()
	err = service.UpdateCaseCategory(cleanCategory)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanCategory,
	})
}

func DeleteCaseCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := service.DeleteCaseCategoryById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// ==================== Admin: Case Detail ====================

func GetCaseDetails(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	categoryId := c.Query("category_id")
	platformId := c.Query("platform_id")
	details, total, err := service.GetCaseDetails(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), categoryId, platformId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(details)
	common.ApiSuccess(c, pageInfo)
}

func CreateCaseDetail(c *gin.Context) {
	detail := model.EcommerceCaseDetail{}
	err := c.ShouldBindJSON(&detail)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(detail.CategoryId) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "category_id cannot be empty"})
		return
	}
	if utf8.RuneCountInString(detail.PlatformId) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "platform_id cannot be empty"})
		return
	}
	detail.CreatedTime = common.GetTimestamp()
	detail.UpdatedTime = common.GetTimestamp()
	err = service.CreateCaseDetail(&detail)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    detail,
	})
}

func UpdateCaseDetail(c *gin.Context) {
	detail := model.EcommerceCaseDetail{}
	err := c.ShouldBindJSON(&detail)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanDetail, err := service.GetCaseDetailById(detail.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanDetail.CategoryId = detail.CategoryId
	cleanDetail.PlatformId = detail.PlatformId
	cleanDetail.PlatformName = detail.PlatformName
	cleanDetail.VisualFeatures = detail.VisualFeatures
	cleanDetail.Composition = detail.Composition
	cleanDetail.Lighting = detail.Lighting
	cleanDetail.BackgroundStyle = detail.BackgroundStyle
	cleanDetail.CaseReference = detail.CaseReference
	cleanDetail.UpdatedTime = common.GetTimestamp()
	err = service.UpdateCaseDetail(cleanDetail)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanDetail,
	})
}

func DeleteCaseDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := service.DeleteCaseDetailById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
