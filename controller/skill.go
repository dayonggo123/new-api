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
		var supportedNodeTypes []string
		if skill.SupportedNodeTypes != "" {
			_ = common.Unmarshal([]byte(skill.SupportedNodeTypes), &supportedNodeTypes)
		}

		responses = append(responses, model.SkillResponse{
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
		})
	}

	common.ApiSuccess(c, responses)
}
