package controller

import (
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetAllPresetPrompts 管理员获取预设提示词列表
func GetAllPresetPrompts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	category := c.Query("category")
	status := 0
	if s := c.Query("status"); s != "" {
		status, _ = strconv.Atoi(s)
	}

	prompts, total, err := model.SearchPresetPrompts(keyword, category, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(prompts)
	common.ApiSuccess(c, pageInfo)
}

// GetPresetPrompt 获取单个预设提示词
func GetPresetPrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	prompt, err := model.GetPresetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, prompt)
}

// AddPresetPrompt 创建预设提示词
func AddPresetPrompt(c *gin.Context) {
	var prompt model.PresetPrompt
	if err := c.ShouldBindJSON(&prompt); err != nil {
		common.ApiError(c, err)
		return
	}

	if err := prompt.Validate(); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	prompt.CreatedTime = time.Now().Unix()
	prompt.UpdatedTime = time.Now().Unix()

	if err := prompt.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, prompt)
}

// UpdatePresetPrompt 更新预设提示词
func UpdatePresetPrompt(c *gin.Context) {
	var prompt model.PresetPrompt
	if err := c.ShouldBindJSON(&prompt); err != nil {
		common.ApiError(c, err)
		return
	}

	if prompt.Id == 0 {
		common.ApiErrorMsg(c, "id is required")
		return
	}

	if err := prompt.Validate(); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	prompt.UpdatedTime = time.Now().Unix()

	if err := prompt.Update(); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, prompt)
}

// DeletePresetPrompt 删除预设提示词
func DeletePresetPrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	prompt := &model.PresetPrompt{Id: id}
	if err := prompt.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, nil)
}

// GetPresetPromptCategories 获取所有分类（去重）
func GetPresetPromptCategories(c *gin.Context) {
	categories, err := model.GetPresetPromptCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, categories)
}

// GetPublicPresetPrompts 公开接口：获取启用的预设提示词列表
func GetPublicPresetPrompts(c *gin.Context) {
	prompts, err := model.GetEnabledPresetPrompts()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, prompts)
}
