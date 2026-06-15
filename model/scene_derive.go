package model

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// Scene 业务场景标签
type Scene struct {
	Scene string `json:"scene"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// SceneRule 业务场景派生规则
type SceneRule struct {
	Scene    string
	Icon     string
	Color    string
	Keywords []string
}

// SceneRules 预置业务场景规则
// 与 web/src/helpers/scene-derive.js 保持一致
var SceneRules = []SceneRule{
	{Scene: "电商", Icon: "🛒", Color: "orange", Keywords: []string{"电商", "e-commerce", "ecommerce", "shop", "产品图", "产品照", "商品", "listing", "product photo", "white background", "clean background", "studio shot", "商品图", "淘宝", "天猫", "amazon", "shein", "temu"}},
	{Scene: "短视频", Icon: "📱", Color: "pink", Keywords: []string{"短视频", "short video", "tiktok", "reels", "shorts", "抖音", "快手", "竖屏", "vertical video", "vlog"}},
	{Scene: "TVC", Icon: "📺", Color: "red", Keywords: []string{"tvc", "电视广告", "commercial", "brand film", "广告片", "产品视频", "广告"}},
	{Scene: "广告", Icon: "📢", Color: "red", Keywords: []string{"advertisement", "advertising", "campaign", "poster ad", "banner", "户外广告", "信息流", "feed ad"}},
	{Scene: "纪录片", Icon: "🎬", Color: "blue", Keywords: []string{"纪录片", "documentary", "interview", "访谈", "纪实", "candid", "real life"}},
	{Scene: "教学", Icon: "🎓", Color: "green", Keywords: []string{"教学", "tutorial", "education", "course", "教程", "how-to", "培训", "课件", "微课", "mooc"}},
	{Scene: "社交媒体", Icon: "💬", Color: "lime", Keywords: []string{"社交媒体", "social media", "instagram", "facebook", "twitter", "小红书", "微博", "朋友圈", "种草", "post"}},
	{Scene: "品牌设计", Icon: "🎨", Color: "gold", Keywords: []string{"品牌", "branding", "brand identity", "logo", "vi", "visual identity", "海报", "poster", "品牌视觉"}},
	{Scene: "个人IP", Icon: "⭐", Color: "purple", Keywords: []string{"个人ip", "personal brand", "influencer", "头像", "profile picture", "形象照", "自我展示", "portrait"}},
	{Scene: "自媒体", Icon: "📡", Color: "cyan", Keywords: []string{"自媒体", "self-media", "内容创作", "博主", "up主", "创作者", "content creator"}},
	{Scene: "直播", Icon: "🔴", Color: "red", Keywords: []string{"直播", "live stream", "livestream", "直播间", "主播", "live commerce"}},
	{Scene: "出版印刷", Icon: "📖", Color: "grey", Keywords: []string{"出版", "print", "printing", "杂志", "magazine", "书籍", "book", "画册", "brochure"}},
	{Scene: "游戏CG", Icon: "🎮", Color: "violet", Keywords: []string{"游戏", "game", "cg", "character design", "concept art", "unreal", "unity", "游戏角色", "场景设计"}},
	{Scene: "影视后期", Icon: "🎞️", Color: "indigo", Keywords: []string{"影视后期", "vfx", "visual effects", "特效", "合成", "compositing", "电影", "film", "cinema"}},
	{Scene: "动画制作", Icon: "🐾", Color: "magenta", Keywords: []string{"动画", "animation", "anime", "cartoon", "motion graphics", "mg动画", "motion design"}},
	{Scene: "产品摄影", Icon: "📷", Color: "orange", Keywords: []string{"产品摄影", "product photography", "静物", "still life", "商品摄影", "catalog"}},
	{Scene: "人像写真", Icon: "👤", Color: "purple", Keywords: []string{"人像", "portrait", "写真", "headshot", "model", "人物", "face", "beauty"}},
	{Scene: "风景摄影", Icon: "🏞️", Color: "teal", Keywords: []string{"风景", "landscape", "nature", "风光", "旅行", "travel", "mountain", "ocean", "sunset", "航拍"}},
	{Scene: "美食", Icon: "🍜", Color: "amber", Keywords: []string{"美食", "food", "cuisine", "dish", "restaurant", "menu", "食物摄影", "food photography"}},
	{Scene: "时尚", Icon: "👗", Color: "fuchsia", Keywords: []string{"时尚", "fashion", "runway", "couture", "街拍", "streetwear", "lookbook", "model fashion"}},
}

// DeriveScenes 从 Prompt 的 tags/title/description/content 派生业务场景
func DeriveScenes(prompt *Prompt) []Scene {
	if prompt == nil {
		return nil
	}

	haystackParts := []string{
		prompt.Title,
		prompt.Description,
		prompt.Content,
		prompt.ContentEn,
	}

	// 解析 tags 字段（JSON 数组或逗号分隔）
	if prompt.Tags != "" {
		var tagList []string
		if err := json.Unmarshal([]byte(prompt.Tags), &tagList); err == nil {
			haystackParts = append(haystackParts, tagList...)
		} else {
			haystackParts = append(haystackParts, strings.Split(prompt.Tags, ",")...)
		}
	}

	var haystack strings.Builder
	for _, part := range haystackParts {
		if part = strings.TrimSpace(part); part != "" {
			haystack.WriteString(strings.ToLower(part))
			haystack.WriteString(" ")
		}
	}
	lowerHaystack := haystack.String()

	var matched []Scene
	seen := make(map[string]struct{})
	for _, rule := range SceneRules {
		for _, kw := range rule.Keywords {
			if kw == "" {
				continue
			}
			if strings.Contains(lowerHaystack, strings.ToLower(kw)) {
				if _, ok := seen[rule.Scene]; !ok {
					matched = append(matched, Scene{
						Scene: rule.Scene,
						Icon:  rule.Icon,
						Color: rule.Color,
					})
					seen[rule.Scene] = struct{}{}
				}
				break
			}
		}
	}

	return matched
}

// DeriveScenesForPublic 给公共 API 使用的简化包装，记录日志等可扩展
func DeriveScenesForPublic(prompt *Prompt) []Scene {
	scenes := DeriveScenes(prompt)
	if len(scenes) == 0 {
		return []Scene{}
	}
	return scenes
}

// MatchScene 判断 prompt 是否命中指定场景
func MatchScene(prompt *Prompt, sceneName string) bool {
	if sceneName == "" {
		return true
	}
	for _, s := range DeriveScenes(prompt) {
		if s.Scene == sceneName {
			return true
		}
	}
	return false
}

func init() {
	// 避免 common 包未使用报错（如果后续加日志）
	_ = common.SysLog
}
