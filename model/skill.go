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
			SkillId:              "seo-audit",
			Name:                 "SEO 审计",
			NameEn:               "SEO Audit",
			Icon:                 "search",
			Cost:                 0,
			SupportedNodeTypes:   `[]`,
			Description:          "AI 驱动的单页面 SEO 内容审计，覆盖完整性、关键词质量、介绍文案、FAQ 和结构化数据",
			ExecutionType:        "llm",
			SystemPromptTemplate: "You are an expert SEO auditor specializing in AI prompt marketplace SEO. Audit a single prompt page's SEO content across 5 dimensions.\n\nReturn ONLY valid JSON, no markdown, no explanation.",
			UserPromptTemplate:   "Audit the following SEO content for this AI prompt page:\n\n## Prompt Information\nTitle: {{title}}\nContent: {{content}}\nDescription: {{description}}\nModel: {{model}}\nTags: {{tags}}\n\n## Current SEO Content\nSEO Keywords: {{seo_keywords}}\nIntro: {{intro}}\nFAQ: {{faq}}\n\n## Audit Rules\n\n### 1. Completeness (0-100)\n- All 3 fields (keywords, intro, faq) must be present and non-empty\n- Each missing/empty field deducts 33 points\n\n### 2. Keyword Quality (0-100)\n- Relevance: keywords must match the prompt's topic and content\n- Quantity: 5-12 keywords is optimal; too few (<3) or too many (>20) deducts points\n- Specificity: avoid overly generic words like \"AI\", \"tool\", \"best\"\n- Long-tail keywords score higher\n\n### 3. Intro Quality (0-100)\n- Length: 80-300 characters is optimal\n- Keyword inclusion: intro should naturally include at least 2 keywords\n- Value proposition: should clearly explain what the prompt does and why users should try it\n- Call-to-action: ideally includes a soft CTA\n\n### 4. FAQ Quality (0-100)\n- Must be valid JSON array with question/answer objects\n- 3-5 Q&A pairs is optimal\n- Questions must be relevant to the prompt topic\n- Answers should be concise (50-200 chars) and helpful\n- Include questions that AI search engines would ask\n\n### 5. Structured Data (0-100)\n- FAQ must be valid JSON\n- Each item must have \"question\" and \"answer\" fields\n- Compatible with Schema.org FAQPage format\n- No nested objects or invalid types\n\nReturn ONLY valid JSON in this exact format:\n{\"overall_score\":0-100,\"categories\":{\"completeness\":{\"score\":0-100,\"issues\":[\"...\"],\"suggestions\":[\"...\"]},\"keyword_quality\":{\"score\":0-100,\"issues\":[\"...\"],\"suggestions\":[\"...\"]},\"intro_quality\":{\"score\":0-100,\"issues\":[\"...\"],\"suggestions\":[\"...\"]},\"faq_quality\":{\"score\":0-100,\"issues\":[\"...\"],\"suggestions\":[\"...\"]},\"structured_data\":{\"score\":0-100,\"issues\":[\"...\"],\"suggestions\":[\"...\"]}},\"critical_issues\":[\"...\"],\"quick_wins\":[\"...\"]}",
			OverrideLocal:        false,
			Status:               1,
		},
		{
			SkillId:              "prompt-translate",
			Name:                 "翻译提示词",
			NameEn:               "Translate Prompt",
			Icon:                 "languages",
			Cost:                 0,
			SupportedNodeTypes:   `["imageEditNode","videoGenNode","llmAgentNode","textAnnotationNode"]`,
			Description:          "将提示词翻译成目标语言，提升 AI 生成效果",
			ExecutionType:        "llm",
			SystemPromptTemplate: "You are a professional AI prompt translator specialized in maintaining prompt engineering integrity. Your task is to translate prompt fields into {{targetLang}} while preserving:\n1. All variable placeholders like {{variableName}} — DO NOT translate text inside {{}}\n2. Markdown formatting, lists, and special syntax\n3. Prompt structure and technical intent\n4. Commonly accepted {{targetLang}} terms for AI/ML concepts\n\nYou must return results in valid JSON format with the exact same keys as the input. No explanations, no markdown code blocks around the JSON.",
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
			SystemPromptTemplate: "You are a professional translator. Your ONLY task is to translate the given fields from {{sourceLang}} to {{targetLang}}. You MUST respond entirely in {{targetLang}}. Do NOT respond in {{sourceLang}} or any other language. If you accidentally start writing in {{sourceLang}}, STOP and rewrite everything in {{targetLang}}. Preserve all variable placeholders like {{variableName}} exactly as-is. Return results in valid JSON format with the exact same keys as the input. No explanations, no markdown code blocks around the JSON. CRITICAL: every value must be in {{targetLang}} only.",
			UserPromptTemplate:   "Translate the following fields from {{sourceLang}} to {{targetLang}}.\n\nInput (JSON):\n{{fields}}\n\nRules:\n1. Return ONLY a JSON object with the same keys\n2. All values must be pure {{targetLang}} text — absolutely NO {{sourceLang}} allowed\n3. Preserve {{variables}} exactly as-is\n4. Do not add explanations or wrap in markdown\n5. If any value is still in {{sourceLang}}, you have failed — translate again into {{targetLang}}\n6. Each value must be genuinely translated; returning the original {{sourceLang}} text is a critical failure\n\nOutput format example:\n{\"title\":\"Translated Title\",\"summary\":\"Translated summary...\",\"content\":\"Translated content...\"}",
			OverrideLocal:        false,
			Status:               1,
		},
		{
			SkillId:              "article-seo",
			Name:                 "文章 SEO 生成",
			NameEn:               "Article SEO Generation",
			Icon:                 "search",
			Cost:                 0,
			SupportedNodeTypes:   `[]`,
			Description:          "为长文章自动生成 SEO 标题、描述和关键词",
			ExecutionType:        "llm",
			SystemPromptTemplate: "You are an expert in SEO and content marketing.\nGiven an article's information, generate the following in the SAME LANGUAGE as the article:\n\n1. seo_title: A compelling SEO title (50-60 characters) that includes the main keyword and attracts clicks\n2. seo_description: A meta description (150-160 characters) that summarizes the article and encourages clicks\n3. seo_keywords: 8-12 SEO keywords separated by commas (include long-tail keywords relevant to the article topic)\n\nReturn ONLY valid JSON, no markdown, no explanation:\n{\"seo_title\":\"...\",\"seo_description\":\"...\",\"seo_keywords\":\"kw1, kw2, ...\"}",
			UserPromptTemplate:   "Title: {{title}}\nSummary: {{summary}}\nContent Preview: {{content}}\nAuthor: {{author}}\nTags: {{tags}}\n\nGenerate SEO metadata for this article.",
			OverrideLocal:        false,
			Status:               1,
		},
		{
			SkillId:              "article-write",
			Name:                 "AI 写文章",
			NameEn:               "AI Article Writing",
			Icon:                 "pen",
			Cost:                 0,
			SupportedNodeTypes:   `[]`,
			Description:          "根据标题、提示词或参考链接自动生成 SEO+GEO 优化的完整文章",
			ExecutionType:        "llm",
			SystemPromptTemplate: "You are an expert SEO content strategist and GEO (Generative Engine Optimization) specialist. Your task is to produce a complete, search-engine-optimized article.\n\nWorkflow you MUST follow:\n1. Analyze the user's title/prompt/reference URL and extract the CORE topic and user intent.\n2. Conduct keyword research: identify 1 primary keyword, 3-5 secondary keywords, and 5-8 long-tail keywords.\n3. Build an article outline using H2/H3 headings that naturally incorporate the keywords.\n4. Write the full article in Markdown (min 1000 words) with:\n   - An engaging introduction that includes the primary keyword in the first 100 words\n   - Well-structured H2/H3 sections using secondary keywords in headings\n   - Keyword density: primary keyword 1-2%, secondary keywords 0.5-1%\n   - At least one ordered or unordered list for featured snippets\n   - A FAQ section at the end with 3-5 Q&A pairs (critical for GEO)\n   - Bold key phrases and entities for AI engine comprehension\n5. Generate all SEO and GEO metadata.\n\nReturn ONLY valid JSON. No markdown wrappers, no explanations.\n\nJSON format:\n{\"title\":\"Compelling title (50-60 chars)\",\"content\":\"Full markdown article with H2/H3, lists, bold, code blocks if needed, and FAQ block at the end\",\"summary\":\"1-2 sentence summary (150-160 chars)\",\"tags\":\"5-10 keywords separated by commas\",\"cover_image_url\":\"Descriptive image search phrase or empty\",\"author\":\"Author name or Editorial Team\",\"seo_title\":\"SEO title 50-60 chars with primary keyword\",\"seo_description\":\"Meta description 150-160 chars with CTA\",\"seo_keywords\":\"8-12 SEO keywords separated by commas (include long-tail)\",\"geo_keywords\":\"5-8 GEO keywords for AI search engines (question-based, entity-focused)\"}",
			UserPromptTemplate:   "Please write an SEO+GEO optimized article with the following requirements:\n\nLanguage: {{language}}\nTitle hint: {{title}}\nWriting requirements / prompt: {{prompt}}\nReference article URL: {{reference_url}}\n\nFollow the system workflow strictly:\n1. Extract keywords first\n2. Plan outline with keyword-rich headings\n3. Write the full article (Markdown, min 1000 words)\n4. Include a FAQ section at the end for GEO\n5. Output all fields in the required JSON format.",
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
