package model

import (
	"github.com/QuantumNous/new-api/common"
)

type Skill struct {
	Id                   int    `json:"-" gorm:"primaryKey"`
	SkillId              string `json:"-" gorm:"column:skill_id;uniqueIndex"`
	Name                 string `json:"name"`
	NameEn               string `json:"nameEn" gorm:"column:name_en"`
	Icon                 string `json:"icon"`
	Cost                 int    `json:"cost"`
	SupportedNodeTypes   string `json:"-" gorm:"column:supported_node_types;type:text"`
	Description          string `json:"description"`
	ExecutionType        string `json:"-" gorm:"column:execution_type"`
	SystemPromptTemplate string `json:"-" gorm:"column:system_prompt_template;type:text"`
	UserPromptTemplate   string `json:"-" gorm:"column:user_prompt_template;type:text"`
	OverrideLocal        bool   `json:"overrideLocal" gorm:"column:override_local"`
	Status               int    `json:"-" gorm:"column:status;default:1"`
}

func (Skill) TableName() string {
	return "skill"
}

type SkillExecution struct {
	Type                 string `json:"type"`
	SystemPromptTemplate string `json:"systemPromptTemplate"`
	UserPromptTemplate   string `json:"userPromptTemplate"`
}

type SkillResponse struct {
	Id                 string         `json:"id"`
	Name               string         `json:"name"`
	NameEn             string         `json:"nameEn"`
	Icon               string         `json:"icon"`
	Cost               int            `json:"cost"`
	SupportedNodeTypes []string       `json:"supportedNodeTypes"`
	Description        string         `json:"description"`
	Execution          SkillExecution `json:"execution"`
	OverrideLocal      bool           `json:"overrideLocal"`
}

func GetAllActiveSkills() ([]Skill, error) {
	var skills []Skill
	err := DB.Where("status = ?", 1).Order("id ASC").Find(&skills).Error
	return skills, err
}

func initDefaultSkills() error {
	var count int64
	if err := DB.Model(&Skill{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaultSkills := []Skill{
		{
			SkillId:              "prompt-translate",
			Name:                 "翻译提示词",
			NameEn:               "Translate Prompt",
			Icon:                 "languages",
			Cost:                 0,
			SupportedNodeTypes:   `["imageEditNode","videoGenNode","llmAgentNode","textAnnotationNode"]`,
			Description:          "将提示词翻译成英文，提升 AI 生成效果",
			ExecutionType:        "llm",
			SystemPromptTemplate: "You are a professional translator. Translate the user's prompt into natural, fluent English optimized for AI generation. Output ONLY the translated text, no explanations.",
			UserPromptTemplate:   "Translate the following into English:\n\n\"\"\"\n{{prompt}}\n\"\"\"",
			OverrideLocal:        false,
			Status:               1,
		},
	}

	for _, skill := range defaultSkills {
		if err := DB.Create(&skill).Error; err != nil {
			common.SysLog("failed to create default skill: " + err.Error())
		}
	}
	return nil
}
