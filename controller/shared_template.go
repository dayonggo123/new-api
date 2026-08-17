package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ========== 公开接口 ==========

// GetSharedTemplates 获取公开模板列表
func GetSharedTemplates(c *gin.Context) {
	var query dto.SharedTemplateListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid query parameters")
		return
	}

	resp, err := service.GetSharedTemplates(&query)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// GetSharedTemplateDetail 获取模板详情
func GetSharedTemplateDetail(c *gin.Context) {
	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	resp, err := service.GetSharedTemplateDetail(templateId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// ========== 用户接口 ==========

// ShareSharedTemplate 分享模板
func ShareSharedTemplate(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	userName, _ := c.Get("username")
	authorName := ""
	if userName != nil {
		authorName, _ = userName.(string)
	}
	if authorName == "" {
		// fallback: 用用户 ID
		authorName = "user_" + fmt.Sprintf("%d", userId)
	}

	var req dto.SharedTemplateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}

	resp, err := service.ShareTemplate(userId, authorName, &req)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// GetMySharedTemplates 获取我的模板列表
func GetMySharedTemplates(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	pageInfo := common.GetPageQuery(c)

	resp, err := service.GetMySharedTemplates(userId, pageInfo.Page, pageInfo.PageSize)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// DeleteSharedTemplate 删除模板
func DeleteSharedTemplate(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	if err := service.DeleteSharedTemplate(templateId, userId); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": true})
}

// RecordSharedTemplateUse 记录模板使用
func RecordSharedTemplateUse(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	if err := service.RecordSharedTemplateUse(templateId, userId); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, nil)
}

// ========== 管理员接口 ==========

// AdminGetPendingSharedTemplates 管理员获取待审核列表
func AdminGetPendingSharedTemplates(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	resp, err := service.GetPendingSharedTemplates(pageInfo.Page, pageInfo.PageSize)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// AdminGetSharedTemplateDetail 管理员获取模板详情（含 pending/rejected）
func AdminGetSharedTemplateDetail(c *gin.Context) {
	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	resp, err := service.GetPendingSharedTemplateDetail(templateId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// AdminAuditSharedTemplate 审核模板
func AdminAuditSharedTemplate(c *gin.Context) {
	adminId := c.GetInt("id")
	if adminId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	adminName, _ := c.Get("username")
	adminNameStr := ""
	if adminName != nil {
		adminNameStr, _ = adminName.(string)
	}
	if adminNameStr == "" {
		adminNameStr = "admin_" + fmt.Sprintf("%d", adminId)
	}

	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	var req dto.AuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}

	resp, err := service.AuditSharedTemplate(templateId, adminId, adminNameStr, &req)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}

// AdminDeleteSharedTemplate 管理员删除模板（任意状态，软删除）
func AdminDeleteSharedTemplate(c *gin.Context) {
	adminId := c.GetInt("id")
	if adminId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	adminName, _ := c.Get("username")
	adminNameStr := ""
	if adminName != nil {
		adminNameStr, _ = adminName.(string)
	}
	if adminNameStr == "" {
		adminNameStr = "admin_" + fmt.Sprintf("%d", adminId)
	}

	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	if err := service.AdminDeleteSharedTemplate(templateId, adminId, adminNameStr); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": true})
}

// AdminSetSharedTemplateHidden 管理员隐藏/取消隐藏模板
func AdminSetSharedTemplateHidden(c *gin.Context) {
	adminId := c.GetInt("id")
	if adminId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	adminName, _ := c.Get("username")
	adminNameStr := ""
	if adminName != nil {
		adminNameStr, _ = adminName.(string)
	}
	if adminNameStr == "" {
		adminNameStr = "admin_" + fmt.Sprintf("%d", adminId)
	}

	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	var req struct {
		Hidden bool `json:"hidden"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}

	if err := service.AdminSetSharedTemplateHidden(templateId, req.Hidden, adminId, adminNameStr); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"hidden": req.Hidden})
}

// AdminPermanentDeleteSharedTemplate 管理员彻底删除模板（物理删除，不可恢复）
func AdminPermanentDeleteSharedTemplate(c *gin.Context) {
	adminId := c.GetInt("id")
	if adminId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return
	}

	adminName, _ := c.Get("username")
	adminNameStr := ""
	if adminName != nil {
		adminNameStr, _ = adminName.(string)
	}
	if adminNameStr == "" {
		adminNameStr = "admin_" + fmt.Sprintf("%d", adminId)
	}

	templateId := c.Param("id")
	if templateId == "" {
		common.ApiErrorMsg(c, "template id is required")
		return
	}

	if err := service.AdminPermanentDeleteSharedTemplate(templateId, adminId, adminNameStr); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": true})
}

// AdminListSharedTemplates 管理员获取全部模板
func AdminListSharedTemplates(c *gin.Context) {
	var query dto.AdminSharedTemplateListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid query parameters")
		return
	}

	resp, err := service.AdminListSharedTemplates(&query)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, resp)
}
