package model

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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SeedKeyword string `json:"seed_keyword"`
}

// GetSEOQuickTemplates 获取预设的快速研究模板
func GetSEOQuickTemplates() []SEOQuickTemplate {
	return []SEOQuickTemplate{
		{
			ID:          "ai-video-prompts",
			Name:        "AI Video Prompt Library",
			Description: "AI 视频提示词库关键词研究",
			SeedKeyword: "AI video prompt library",
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
	}
}
