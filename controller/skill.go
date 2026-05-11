package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetSkills 返回所有启用的 Skill 配置列表
func GetSkills(c *gin.Context) {
	skills, err := model.GetAllActiveSkills()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	responses := make([]model.SkillResponse, 0, len(skills))
	for _, skill := range skills {
		responses = append(responses, skillToResponse(skill))
	}

	common.ApiSuccess(c, responses)
}

// AdminListSkills 返回所有 Skill（含禁用）
func AdminListSkills(c *gin.Context) {
	skills, err := model.GetAllSkills()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	responses := make([]model.SkillResponse, 0, len(skills))
	for _, skill := range skills {
		responses = append(responses, skillToResponse(skill))
	}

	common.ApiSuccess(c, responses)
}

type skillRequest struct {
	Id                 string              `json:"id" binding:"required"`
	Name               string              `json:"name" binding:"required"`
	NameEn             string              `json:"nameEn"`
	Icon               string              `json:"icon" binding:"required"`
	Cost               int                 `json:"cost"`
	SupportedNodeTypes []string            `json:"supportedNodeTypes"`
	Description        string              `json:"description"`
	Execution          model.SkillExecution `json:"execution"`
	OverrideLocal      bool                `json:"overrideLocal"`
	Status             int                 `json:"status"`
}

// AdminCreateSkill 创建 Skill
func AdminCreateSkill(c *gin.Context) {
	var req skillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	nodeTypesJSON, _ := common.Marshal(req.SupportedNodeTypes)
	skill := &model.Skill{
		SkillId:              req.Id,
		Name:                 req.Name,
		NameEn:               req.NameEn,
		Icon:                 req.Icon,
		Cost:                 req.Cost,
		SupportedNodeTypes:   string(nodeTypesJSON),
		Description:          req.Description,
		ExecutionType:        req.Execution.Type,
		SystemPromptTemplate: req.Execution.SystemPromptTemplate,
		UserPromptTemplate:   req.Execution.UserPromptTemplate,
		OverrideLocal:        req.OverrideLocal,
		Status:               req.Status,
	}
	if skill.Status == 0 {
		skill.Status = 1
	}

	if err := model.CreateSkill(skill); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, skillToResponse(*skill))
}

// AdminUpdateSkill 更新 Skill
func AdminUpdateSkill(c *gin.Context) {
	skillId := c.Param("id")
	if skillId == "" {
		common.ApiErrorMsg(c, "id is required")
		return
	}

	var req skillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	existing, err := model.GetSkillBySkillId(skillId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	nodeTypesJSON, _ := common.Marshal(req.SupportedNodeTypes)
	existing.Name = req.Name
	existing.NameEn = req.NameEn
	existing.Icon = req.Icon
	existing.Cost = req.Cost
	existing.SupportedNodeTypes = string(nodeTypesJSON)
	existing.Description = req.Description
	existing.ExecutionType = req.Execution.Type
	existing.SystemPromptTemplate = req.Execution.SystemPromptTemplate
	existing.UserPromptTemplate = req.Execution.UserPromptTemplate
	existing.OverrideLocal = req.OverrideLocal
	existing.Status = req.Status
	if existing.Status == 0 {
		existing.Status = 1
	}

	if err := model.UpdateSkill(existing); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, skillToResponse(*existing))
}

// AdminDeleteSkill 删除 Skill
func AdminDeleteSkill(c *gin.Context) {
	skillId := c.Param("id")
	if skillId == "" {
		common.ApiErrorMsg(c, "id is required")
		return
	}

	if err := model.DeleteSkillBySkillId(skillId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func skillToResponse(skill model.Skill) model.SkillResponse {
	var supportedNodeTypes []string
	if skill.SupportedNodeTypes != "" {
		_ = common.Unmarshal([]byte(skill.SupportedNodeTypes), &supportedNodeTypes)
	}
	return model.SkillResponse{
		Id:                 skill.SkillId,
		Name:               skill.Name,
		NameEn:             skill.NameEn,
		Icon:               skill.Icon,
		Cost:               skill.Cost,
		SupportedNodeTypes: supportedNodeTypes,
		Description:        skill.Description,
		Execution: model.SkillExecution{
			Type:                 skill.ExecutionType,
			SystemPromptTemplate: skill.SystemPromptTemplate,
			UserPromptTemplate:   skill.UserPromptTemplate,
		},
		OverrideLocal: skill.OverrideLocal,
	}
}
