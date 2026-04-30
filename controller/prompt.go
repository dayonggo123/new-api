package controller

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ==================== Admin: Prompt Category ====================

func GetAllPromptCategories(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	categories, total, err := model.GetAllPromptCategories(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(categories)
	common.ApiSuccess(c, pageInfo)
}

func GetEnabledPromptCategories(c *gin.Context) {
	categories, err := model.GetEnabledPromptCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

func GetPromptCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	category, err := model.GetPromptCategoryById(id)
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

func AddPromptCategory(c *gin.Context) {
	category := model.PromptCategory{}
	err := c.ShouldBindJSON(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(category.Name) == 0 || utf8.RuneCountInString(category.Name) > 50 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "分类名称不能为空且不能超过50个字符"})
		return
	}
	category.CreatedTime = common.GetTimestamp()
	category.UpdatedTime = common.GetTimestamp()
	err = category.Insert()
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

func UpdatePromptCategory(c *gin.Context) {
	category := model.PromptCategory{}
	err := c.ShouldBindJSON(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanCategory, err := model.GetPromptCategoryById(category.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanCategory.Name = category.Name
	cleanCategory.Description = category.Description
	cleanCategory.Icon = category.Icon
	cleanCategory.SortOrder = category.SortOrder
	cleanCategory.Status = category.Status
	cleanCategory.UpdatedTime = common.GetTimestamp()
	err = cleanCategory.Update()
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

func DeletePromptCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeletePromptCategoryById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// ==================== Admin: Prompt ====================

func GetAllPrompts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))

	prompts, total, err := model.SearchPromptsWithCategory(keyword, categoryId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(prompts)
	common.ApiSuccess(c, pageInfo)
}

func GetPrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prompt, err := model.GetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

func AddPrompt(c *gin.Context) {
	prompt := model.Prompt{}
	err := c.ShouldBindJSON(&prompt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(prompt.Title) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提示词标题不能为空"})
		return
	}
	if utf8.RuneCountInString(prompt.Content) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提示词内容不能为空"})
		return
	}
	prompt.CreatedTime = common.GetTimestamp()
	prompt.UpdatedTime = common.GetTimestamp()
	err = prompt.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

func UpdatePrompt(c *gin.Context) {
	prompt := model.Prompt{}
	err := c.ShouldBindJSON(&prompt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanPrompt, err := model.GetPromptById(prompt.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanPrompt.CategoryId = prompt.CategoryId
	cleanPrompt.Title = prompt.Title
	cleanPrompt.Content = prompt.Content
	cleanPrompt.Description = prompt.Description
	cleanPrompt.Variables = prompt.Variables
	cleanPrompt.Tags = prompt.Tags
	cleanPrompt.SortOrder = prompt.SortOrder
	cleanPrompt.Status = prompt.Status
	cleanPrompt.UpdatedTime = common.GetTimestamp()
	err = cleanPrompt.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanPrompt,
	})
}

func DeletePrompt(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeletePromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// ==================== Public API (no auth required) ====================

func GetPublicPrompts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))

	prompts, total, err := model.GetPublicPrompts(categoryId, keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(prompts)
	common.ApiSuccess(c, pageInfo)
}

func GetPublicPrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prompt, err := model.GetPublicPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Increment usage count asynchronously
	go model.IncrementPromptUsageCount(id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

func GetPublicPromptCategories(c *gin.Context) {
	categories, err := model.GetEnabledPromptCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}
