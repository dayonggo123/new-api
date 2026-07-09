package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

func ChannelType2APIType(channelType int) (int, bool) {
	return ChannelType2APITypeWithModel(channelType, "")
}

func ChannelType2APITypeWithModel(channelType int, modelName string) (int, bool) {
	apiType := -1
	switch channelType {
	case constant.ChannelTypeOpenAI:
		apiType = constant.APITypeOpenAI
	case constant.ChannelTypeAnthropic:
		apiType = constant.APITypeAnthropic
	case constant.ChannelTypeBaidu:
		apiType = constant.APITypeBaidu
	case constant.ChannelTypePaLM:
		apiType = constant.APITypePaLM
	case constant.ChannelTypeZhipu:
		apiType = constant.APITypeZhipu
	case constant.ChannelTypeAli:
		apiType = constant.APITypeAli
	case constant.ChannelTypeXunfei:
		apiType = constant.APITypeXunfei
	case constant.ChannelTypeAIProxyLibrary:
		apiType = constant.APITypeAIProxyLibrary
	case constant.ChannelTypeTencent:
		apiType = constant.APITypeTencent
	case constant.ChannelTypeGemini:
		apiType = constant.APITypeGemini
	case constant.ChannelTypeZhipu_v4:
		apiType = constant.APITypeZhipuV4
	case constant.ChannelTypeOllama:
		apiType = constant.APITypeOllama
	case constant.ChannelTypePerplexity:
		apiType = constant.APITypePerplexity
	case constant.ChannelTypeAws:
		apiType = constant.APITypeAws
	case constant.ChannelTypeCohere:
		apiType = constant.APITypeCohere
	case constant.ChannelTypeDify:
		apiType = constant.APITypeDify
	case constant.ChannelTypeJina:
		apiType = constant.APITypeJina
	case constant.ChannelCloudflare:
		apiType = constant.APITypeCloudflare
	case constant.ChannelTypeSiliconFlow:
		apiType = constant.APITypeSiliconFlow
	case constant.ChannelTypeVertexAi:
		apiType = constant.APITypeVertexAi
	case constant.ChannelTypeMistral:
		apiType = constant.APITypeMistral
	case constant.ChannelTypeDeepSeek:
		apiType = constant.APITypeDeepSeek
	case constant.ChannelTypeMokaAI:
		apiType = constant.APITypeMokaAI
	case constant.ChannelTypeVolcEngine:
		apiType = constant.APITypeVolcEngine
	case constant.ChannelTypeBaiduV2:
		apiType = constant.APITypeBaiduV2
	case constant.ChannelTypeOpenRouter:
		apiType = constant.APITypeOpenRouter
	case constant.ChannelTypeXinference:
		apiType = constant.APITypeXinference
	case constant.ChannelTypeXai:
		apiType = constant.APITypeXai
	case constant.ChannelTypeCoze:
		apiType = constant.APITypeCoze
	case constant.ChannelTypeJimeng:
		apiType = constant.APITypeJimeng
	case constant.ChannelTypeMoonshot:
		apiType = constant.APITypeMoonshot
	case constant.ChannelTypeSubmodel:
		apiType = constant.APITypeSubmodel
	case constant.ChannelTypeMiniMax:
		apiType = constant.APITypeMiniMax
	case constant.ChannelTypeReplicate:
		apiType = constant.APITypeReplicate
	case constant.ChannelTypeCodex:
		apiType = constant.APITypeCodex
	case constant.ChannelTypeVeo:
		apiType = constant.APITypeVeo
	case constant.ChannelTypeEasyRouter:
		apiType = constant.APITypeGemini
	case constant.ChannelTypeAPIMart:
		// APIMart 同时支持 task 图像模型（gpt-image 系列）和 OpenAI 兼容接口（Whisper 等）。
		// 图像任务在 image_handler 中按 ChannelType 直接拦截，不走到此处。
		apiType = constant.APITypeOpenAI
	case constant.ChannelTypeWanXiangAI:
		// WanXiangAI media models (image/video) use APITypeVeo for task-based generation
		if modelName != "" && (strings.Contains(modelName, "veo") || strings.Contains(modelName, "gemini")) {
			apiType = constant.APITypeVeo
		} else {
			apiType = constant.APITypeOpenAI
		}
	}
	if apiType == -1 {
		return constant.APITypeOpenAI, false
	}
	return apiType, true
}
