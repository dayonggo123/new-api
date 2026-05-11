package model

import (
	"github.com/QuantumNous/new-api/common"
)

type Skill struct {
	Id                   int    `json:"-" gorm:"primaryKey"`
	SkillId              string `json:"-" gorm:"column:skill_id;size:64;uniqueIndex"`
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

func GetAllSkills() ([]Skill, error) {
	var skills []Skill
	err := DB.Order("id ASC").Find(&skills).Error
	return skills, err
}

func GetSkillBySkillId(skillId string) (*Skill, error) {
	var skill Skill
	err := DB.Where("skill_id = ?", skillId).First(&skill).Error
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

func CreateSkill(skill *Skill) error {
	return DB.Create(skill).Error
}

func UpdateSkill(skill *Skill) error {
	return DB.Model(skill).Updates(skill).Error
}

func DeleteSkillBySkillId(skillId string) error {
	return DB.Where("skill_id = ?", skillId).Delete(&Skill{}).Error
}

func initDefaultSkills() error {
	defaultSkills := []Skill{
		{
			SkillId:              "prompt-translate",
			Name:                 "翻译提示词",
			NameEn:               "Translate Prompt",
			Icon:                 "languages",
			Cost:                 0,
			SupportedNodeTypes:   `["imageEditNode","videoGenNode","llmAgentNode","textAnnotationNode"]`,
			Description:          "将提示词翻译成目标语言，提升 AI 生成效果",
			ExecutionType:        "llm",
			SystemPromptTemplate: "You are a professional translator. Your ONLY task is to translate text. You MUST respond entirely in {{targetLang}}. Do NOT respond in {{sourceLang}} or any other language. Do not add explanations, notes, or the original text — output ONLY the translated text in {{targetLang}}.",
			UserPromptTemplate:   "Translate the following text from {{sourceLang}} to {{targetLang}}. Your response must be ONLY the translated text in {{targetLang}}, nothing else:\n\n\"\"\"\n{{prompt}}\n\"\"\"",
			OverrideLocal:        false,
			Status:               1,
		},
		{
			SkillId:              "batch-translate",
			Name:                 "批量翻译",
			NameEn:               "Batch Translate",
			Icon:                 "languages",
			Cost:                 0,
			SupportedNodeTypes:   `[]`,
			Description:          "批量将多个字段翻译成目标语言",
			ExecutionType:        "llm",
			SystemPromptTemplate: "You are a professional translator. Your ONLY task is to translate text. You MUST respond entirely in {{targetLang}}. Do NOT respond in {{sourceLang}} or any other language. Do not add explanations, notes, or the original text — output ONLY the translated text in {{targetLang}}.",
			UserPromptTemplate:   "Translate ALL the following items from {{sourceLang}} to {{targetLang}}. You MUST translate every item into {{targetLang}}. Do NOT return the original {{sourceLang}} text under any circumstances. Return the translations in this exact format, one per line, with the key followed by a colon and a space, then the translated text. Do not add any extra text, explanations, markdown code blocks, or blank lines.\n\n{{items}}",
			OverrideLocal:        false,
			Status:               1,
		},
	}

	for _, skill := range defaultSkills {
		var existing Skill
		err := DB.Where("skill_id = ?", skill.SkillId).First(&existing).Error
		if err != nil {
			// 不存在则创建
			if err := DB.Create(&skill).Error; err != nil {
				common.SysLog("failed to create default skill " + skill.SkillId + ": " + err.Error())
			}
		}
		// 已存在则跳过，保留用户自定义配置
	}
	return nil
}
