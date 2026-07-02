package volcengine

import "strings"

var ModelList = []string{
	"Doubao-pro-128k",
	"Doubao-pro-32k",
	"Doubao-pro-4k",
	"Doubao-lite-128k",
	"Doubao-lite-32k",
	"Doubao-lite-4k",
	"Doubao-embedding",
	// Seedream 图片生成模型
	"Doubao-Seedream-5.0-lite",
	"Doubao-Seedream-4.5",
	"doubao-seedream-4-0-250828",
	"seedream-4-0-250828",
	// Seedance 视频生成模型（通过 VolcEngine 通道也可调用）
	"doubao-seedance-1-0-pro-250528",
	"seedance-1-0-pro-250528",
	"doubao-seedance-2-0-260128",
	"seedance-2-0-260128",
	"doubao-seedance-2-0-fast-260128",
	"seedance-2-0-fast-260128",
	// 文本模型
	"doubao-seed-1-6-thinking-250715",
	"seed-1-6-thinking-250715",
	// 文件上传伪模型（用于 /v1/files 路由选择 VolcEngine 渠道）
	"volcengine-files",
}

// SeedreamImageModelAliases maps user-facing model aliases to the real VolcEngine
// Ark model IDs used for image generation (Seedream).
//
// Keys include both the full new-api display names (e.g. "Doubao-Seedream-4.5")
// and the short lowercase aliases commonly used by clients (e.g. "4.5").
var SeedreamImageModelAliases = map[string]string{
	"Doubao-Seedream-5.0-lite": "doubao-seedream-5-0-260128",
	"Doubao-Seedream-4.5":      "doubao-seedream-4-5-251128",
	"Doubao-Seedream-4.0":      "doubao-seedream-4-0-250828",
	"5.0-lite":                 "doubao-seedream-5-0-260128",
	"4.5":                      "doubao-seedream-4-5-251128",
	"4.0":                      "doubao-seedream-4-0-250828",
}

// MapSeedreamImageModel translates a Seedream image generation model alias to the
// real VolcEngine Ark model ID. If the input is already a real VolcEngine ID or is
// not a known alias, it is returned unchanged.
func MapSeedreamImageModel(model string) string {
	if model == "" {
		return model
	}
	if realID, ok := SeedreamImageModelAliases[model]; ok {
		return realID
	}
	// Already looks like a real VolcEngine Seedream ID; leave it unchanged.
	if strings.HasPrefix(strings.ToLower(model), "doubao-seedream-") {
		return model
	}
	return model
}

var ChannelName = "volcengine"
