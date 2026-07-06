package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// GetCuratedTemplates 获取模板列表（公开）
func GetCuratedTemplates(c *gin.Context) {
	var query dto.CuratedTemplateListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid query parameters")
		return
	}
	if query.PageSize > 100 {
		common.ApiErrorMsg(c, "pageSize must be between 1 and 100")
		return
	}

	resp, err := service.GetCuratedTemplates(&query)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// GetCuratedTemplate 获取模板详情（公开）
func GetCuratedTemplate(c *gin.Context) {
	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}
	resp, err := service.GetCuratedTemplateByTemplateId(templateId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// GetCuratedTemplateExecutionPlan 获取模板执行计划（公开）
func GetCuratedTemplateExecutionPlan(c *gin.Context) {
	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}
	plan, err := service.GetCuratedTemplateExecutionPlan(templateId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, plan)
}

// AdminListCuratedTemplates 管理后台获取模板列表
func AdminListCuratedTemplates(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	resp, err := service.AdminListCuratedTemplates(pageInfo.Page, pageInfo.PageSize)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// AdminCreateCuratedTemplate 管理后台创建模板
func AdminCreateCuratedTemplate(c *gin.Context) {
	var req dto.AdminUpsertTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	if req.TemplateId == "" || req.Title == "" || req.Category == "" {
		common.ApiErrorMsg(c, "id, title and category are required")
		return
	}
	template, err := service.AdminCreateTemplate(&req)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, template)
}

// AdminUpdateCuratedTemplate 管理后台更新模板
func AdminUpdateCuratedTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid template id")
		return
	}
	var req dto.AdminUpsertTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	if err := service.AdminUpdateTemplate(id, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminDeleteCuratedTemplate 管理后台删除模板
func AdminDeleteCuratedTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid template id")
		return
	}
	if err := service.AdminDeleteTemplate(id); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminUpdateCuratedTemplateStatus 管理后台更新模板状态
func AdminUpdateCuratedTemplateStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid template id")
		return
	}
	var req dto.AdminUpdateTemplateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	if err := service.AdminUpdateTemplateStatus(id, *req.Enabled); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, nil)
}
