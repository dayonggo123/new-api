package model

import "github.com/QuantumNous/new-api/common"

// SEOKeywordResearchResult AI 关键词研究结果
type SEOKeywordResearchResult struct {
	SeedKeyword       string            `json:"seed_keyword"`        // 用户输入的种子关键词
	Language          string            `json:"language"`            // 目标语言
	SeedKeywords      []KeywordItem     `json:"seed_keywords"`       // 种子词列表（AI 扩展的）
	ExtendedKeywords  []KeywordItem     `json:"extended_keywords"`   // 扩展词列表
	LongTailKeywords  []KeywordItem     `json:"long_tail_keywords"`  // 长尾词列表
	HighROIKeywords   []KeywordItem     `json:"high_roi_keywords"`   // 高 ROI 关键词
	TopicClusters     []TopicCluster    `json:"topic_clusters"`      // 主题簇
	ContentGaps       []ContentGap      `json:"content_gaps"`        // 内容缺口
	TotalCount        int               `json:"total_count"`         // 总关键词数
	HighROICount      int               `json:"high_roi_count"`      // 高 ROI 词数
	ClusterCount      int               `json:"cluster_count"`       // 主题簇数
}

// KeywordItem 单个关键词项
type KeywordItem struct {
	Keyword      string  `json:"keyword"`       // 关键词
	SearchVolume int     `json:"search_volume"` // 预估月搜索量
	Intent       string  `json:"intent"`        // 搜索意图: informational/navigational/transactional/commercial
	Difficulty   string  `json:"difficulty"`    // 竞争难度: low/medium/high
	BusinessValue int    `json:"business_value"` // 商业价值 1-10
	ROIScore     int     `json:"roi_score"`     // ROI 评分 0-100
	SuggestedURL string  `json:"suggested_url"` // 建议的 URL slug
}

// TopicCluster 主题簇
type TopicCluster struct {
	Name          string       `json:"name"`           // 簇名称
	PillarKeyword string       `json:"pillar_keyword"` // 核心 pillar 关键词
	PillarVolume  int          `json:"pillar_volume"`  // pillar 搜索量
	ClusterKeywords []string   `json:"cluster_keywords"` // 簇内关键词
	ContentType   string       `json:"content_type"`   // 建议内容类型: article/prompt/tool/landing
	Priority      string       `json:"priority"`       // 优先级: P0/P1/P2
}

// ContentGap 内容缺口
type ContentGap struct {
	Keyword      string `json:"keyword"`       // 缺口关键词
	Volume       int    `json:"volume"`        // 搜索量
	Competitors  string `json:"competitors"`   // 已覆盖的竞品
	GapType      string `json:"gap_type"`      // 缺口类型: missing_topic/undercovered/poor_quality
	Priority     string `json:"priority"`      // 优先级: P0/P1/P2
	SuggestedAction string `json:"suggested_action"` // 建议行动
}

// SEOQuickTemplate 快速研究模板
type SEOQuickTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SeedKeyword  string `json:"seed_keyword"`
	ResearchMode string `json:"research_mode"` // ai / serp
}

// GetSEOQuickTemplates 获取预设的快速研究模板
func GetSEOQuickTemplates() []SEOQuickTemplate {
	return []SEOQuickTemplate{
		{
			ID:           "ai-video-prompts",
			Name:         "AI Video Prompt Library",
			Description:  "AI 视频提示词库关键词研究",
			SeedKeyword:  "AI video prompt library",
			ResearchMode: "serp",
		},
		{
			ID:          "node-canvas-video",
			Name:        "Node Canvas Video Creation",
			Description: "节点画布视频创作关键词研究",
			SeedKeyword: "node based AI video creation tool",
		},
		{
			ID:          "best-ai-video-tools",
			Name:        "Best AI Video Tools 2026",
			Description: "最佳 AI 视频工具对比关键词研究",
			SeedKeyword: "best AI video generators 2026",
		},
		{
			ID:          "comfyui-video",
			Name:        "ComfyUI Video Workflow",
			Description: "ComfyUI 视频工作流关键词研究",
			SeedKeyword: "ComfyUI video workflow tutorial",
		},
		{
			ID:          "sora-prompts",
			Name:        "Sora Prompt Templates",
			Description: "Sora 提示词模板关键词研究",
			SeedKeyword: "Sora AI video prompts examples",
		},
		{
			ID:          "kling-ai",
			Name:        "Kling AI Video",
			Description: "可灵 AI 视频创作关键词研究",
			SeedKeyword: "Kling AI video generation prompts",
		},
		{
			ID:          "runway-gen3",
			Name:        "Runway Gen-3 Prompts",
			Description: "Runway Gen-3 提示词关键词研究",
			SeedKeyword: "Runway Gen-3 prompt examples",
		},
		{
			ID:          "veo-ai-video",
			Name:        "Veo AI Video Guide",
			Description: "Google Veo AI 视频指南关键词研究",
			SeedKeyword: "Google Veo AI video generator",
		},
		{
			ID:          "ai-video-editor",
			Name:        "AI Video Editor Comparison",
			Description: "AI 视频编辑器对比关键词研究",
			SeedKeyword: "AI video editor comparison",
		},
		{
			ID:          "text-to-video",
			Name:        "Text to Video AI Tools",
			Description: "文本转视频工具关键词研究",
			SeedKeyword: "text to video AI tools free",
		},
		{
			ID:          "ai-video-workflow",
			Name:        "AI Video Workflow Automation",
			Description: "AI 视频工作流自动化关键词研究",
			SeedKeyword: "AI video workflow automation",
		},
		{
			ID:          "stable-video",
			Name:        "Stable Video Diffusion",
			Description: "Stable Video Diffusion 关键词研究",
			SeedKeyword: "Stable Video Diffusion tutorial",
		},
		{
			ID:          "prompt-engineering",
			Name:        "Prompt Engineering for Video",
			Description: "视频提示词工程关键词研究",
			SeedKeyword: "prompt engineering for AI video",
		},
		{
			ID:          "ai-video-marketing",
			Name:        "AI Video Marketing",
			Description: "AI 视频营销关键词研究",
			SeedKeyword: "AI video marketing strategy",
		},
		{
			ID:          "node-video-editor",
			Name:        "Node-based Video Editor",
			Description: "节点式视频编辑器关键词研究",
			SeedKeyword: "node based video editor free",
		},
	}
}

// SEOResearchHistory 关键词研究历史记录
type SEOResearchHistory struct {
	Id          int    `json:"id"`
	SeedKeyword string `json:"seed_keyword" gorm:"index"`
	Language    string `json:"language"`
	ResultJSON  string `json:"result_json" gorm:"type:longtext"`
	TotalCount  int    `json:"total_count"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

// SaveSEOResearchHistory 保存研究历史
func SaveSEOResearchHistory(seedKeyword, language string, result *SEOKeywordResearchResult) error {
	jsonData, err := common.Marshal(result)
	if err != nil {
		return err
	}
	history := &SEOResearchHistory{
		SeedKeyword: seedKeyword,
		Language:    language,
		ResultJSON:  string(jsonData),
		TotalCount:  result.TotalCount,
		CreatedTime: common.GetTimestamp(),
	}
	return DB.Create(history).Error
}

// GetSEOResearchHistories 获取研究历史列表（分页）
func GetSEOResearchHistories(page, pageSize int) ([]SEOResearchHistory, int64, error) {
	var histories []SEOResearchHistory
	var total int64
	offset := (page - 1) * pageSize

	if err := DB.Model(&SEOResearchHistory{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := DB.Order("created_time DESC").Limit(pageSize).Offset(offset).Find(&histories).Error; err != nil {
		return nil, 0, err
	}
	return histories, total, nil
}

// GetSEOResearchHistoryByID 根据 ID 获取历史记录
func GetSEOResearchHistoryByID(id int) (*SEOResearchHistory, error) {
	var history SEOResearchHistory
	if err := DB.First(&history, id).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

// DeleteSEOResearchHistory 删除研究历史
func DeleteSEOResearchHistory(id int) error {
	return DB.Delete(&SEOResearchHistory{}, id).Error
}
