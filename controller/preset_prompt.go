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

	lang := c.Query("lang")
	prompt.ApplyLanguage(lang)

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
// 支持 ?lang= 显式指定语言；未指定且用户已登录时，自动使用用户语言偏好
func GetPublicPresetPrompts(c *gin.Context) {
	prompts, err := model.GetEnabledPresetPrompts()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	lang := c.Query("lang")
	if lang == "" {
		userId := c.GetInt("id")
		if userId > 0 {
			lang = model.GetUserLanguage(userId)
		}
	}
	for i := range prompts {
		prompts[i].ApplyLanguage(lang)
		// 对外隐藏 i18n 原始 JSON，减少响应体积
		prompts[i].I18n = ""
	}

	common.ApiSuccess(c, prompts)
}

// GetPublicPresetPromptUpdates 公开接口：获取提示词增量更新
// 支持 ?since= 参数（秒级时间戳），返回该时间之后有更新的启用提示词
// 不带 since 则返回全部启用提示词（但精简字段）
func GetPublicPresetPromptUpdates(c *gin.Context) {
	sinceStr := c.Query("since")
	since := int64(0)
	if sinceStr != "" {
		if v, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			since = v
		}
	}

	prompts, err := model.GetEnabledPresetPromptsUpdatedSince(since)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	lang := c.Query("lang")
	if lang == "" {
		userId := c.GetInt("id")
		if userId > 0 {
			lang = model.GetUserLanguage(userId)
		}
	}

	// 精简字段返回，只保留下游同步需要的核心信息
	type PromptUpdateItem struct {
		Id          int    `json:"id"`
		Name        string `json:"name"`
		Category    string `json:"category"`
		Status      int    `json:"status"`
		SortOrder   int    `json:"sort_order"`
		UpdatedTime int64  `json:"updated_time"`
		CreatedTime int64  `json:"created_time"`
	}

	items := make([]PromptUpdateItem, 0, len(prompts))
	for i := range prompts {
		prompts[i].ApplyLanguage(lang)
		items = append(items, PromptUpdateItem{
			Id:          prompts[i].Id,
			Name:        prompts[i].Name,
			Category:    prompts[i].Category,
			Status:      prompts[i].Status,
			SortOrder:   prompts[i].SortOrder,
			UpdatedTime: prompts[i].UpdatedTime,
			CreatedTime: prompts[i].CreatedTime,
		})
	}

	common.ApiSuccess(c, gin.H{
		"items":      items,
		"total":      len(items),
		"server_time": time.Now().Unix(),
	})
}
