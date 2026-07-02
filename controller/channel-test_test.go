package controller

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

func TestNormalizeChannelTestEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		endpointType string
		channel      *model.Channel
		want         string
	}{
		{
			name:         "explicit endpoint type takes precedence",
			modelName:    "Doubao-Seedream-4.5",
			endpointType: string(constant.EndpointTypeOpenAI),
			channel:      nil,
			want:         string(constant.EndpointTypeOpenAI),
		},
		{
			name:         "seedream lower case is detected as image generation",
			modelName:    "seedream-4-0-250828",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeImageGeneration),
		},
		{
			name:         "Seedream mixed case is detected as image generation",
			modelName:    "Doubao-Seedream-4.5",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeImageGeneration),
		},
		{
			name:         "doubao-seedream-4-5-251128 is detected as image generation",
			modelName:    "doubao-seedream-4-5-251128",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeImageGeneration),
		},
		{
			name:         "gpt-image remains detected as image generation",
			modelName:    "gpt-image-1",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeImageGeneration),
		},
		{
			name:         "dall-e remains detected as image generation",
			modelName:    "dall-e-3",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeImageGeneration),
		},
		{
			name:         "imagen remains detected as image generation",
			modelName:    "imagen-3.0-generate-001",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeImageGeneration),
		},
		{
			name:         "nano-banana remains detected as image generation",
			modelName:    "nano-banana-pro",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeImageGeneration),
		},
		{
			name:         "whisper remains detected as audio transcription",
			modelName:    "whisper-1",
			endpointType: "",
			channel:      nil,
			want:         string(constant.EndpointTypeAudioTranscription),
		},
		{
			name:         "embedding model remains detected as embeddings via fallback path",
			modelName:    "text-embedding-3-small",
			endpointType: "",
			channel:      nil,
			want:         "",
		},
		{
			name:         "general chat model falls through with empty endpoint type",
			modelName:    "gpt-4o-mini",
			endpointType: "",
			channel:      nil,
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeChannelTestEndpoint(tt.channel, tt.modelName, tt.endpointType)
			if got != tt.want {
				t.Errorf("normalizeChannelTestEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildTestRequest_SeedreamImageDetection(t *testing.T) {
	volcEngineChannel := &model.Channel{Type: constant.ChannelTypeVolcEngine}
	openAIChannel := &model.Channel{Type: constant.ChannelTypeOpenAI}

	tests := []struct {
		name         string
		model        string
		endpointType string
		channel      *model.Channel
		wantType     string
		wantSize     string
	}{
		{
			name:         "seedream auto detected as image request",
			model:        "doubao-seedream-4-5-251128",
			endpointType: "",
			channel:      volcEngineChannel,
			wantType:     "*dto.ImageRequest",
			wantSize:     "1024x1024",
		},
		{
			name:         "Seedream mixed case auto detected as image request",
			model:        "Doubao-Seedream-4.5",
			endpointType: "",
			channel:      openAIChannel,
			wantType:     "*dto.ImageRequest",
			wantSize:     "1024x1024",
		},
		{
			name:         "seedream-4-0-250828 auto detected as image request",
			model:        "seedream-4-0-250828",
			endpointType: "",
			channel:      openAIChannel,
			wantType:     "*dto.ImageRequest",
			wantSize:     "1024x1024",
		},
		{
			name:         "explicit image endpoint overrides size",
			model:        "custom-seedream",
			endpointType: string(constant.EndpointTypeImageGeneration),
			channel:      openAIChannel,
			wantType:     "*dto.ImageRequest",
			wantSize:     "1024x1024",
		},
		{
			name:         "gpt-image remains detected as image request",
			model:        "gpt-image-1",
			endpointType: "",
			channel:      openAIChannel,
			wantType:     "*dto.ImageRequest",
			wantSize:     "1024x1024",
		},
		{
			name:         "embedding remains detected as embedding request",
			model:        "text-embedding-3-small",
			endpointType: "",
			channel:      openAIChannel,
			wantType:     "*dto.EmbeddingRequest",
			wantSize:     "",
		},
		{
			name:         "whisper remains detected as audio request",
			model:        "whisper-1",
			endpointType: "",
			channel:      openAIChannel,
			wantType:     "*dto.GeneralOpenAIRequest",
			wantSize:     "",
		},
		{
			name:         "general chat model returns GeneralOpenAIRequest",
			model:        "gpt-4o-mini",
			endpointType: "",
			channel:      openAIChannel,
			wantType:     "*dto.GeneralOpenAIRequest",
			wantSize:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildTestRequest(tt.model, tt.endpointType, tt.channel, false)
			gotType := ""
			if req != nil {
				gotType = typeName(req)
			}
			if gotType != tt.wantType {
				t.Errorf("buildTestRequest() returned %s, want %s", gotType, tt.wantType)
			}
			if imgReq, ok := req.(*dto.ImageRequest); ok {
				if imgReq.Size != tt.wantSize {
					t.Errorf("ImageRequest.Size = %q, want %q", imgReq.Size, tt.wantSize)
				}
				if imgReq.Model != tt.model {
					t.Errorf("ImageRequest.Model = %q, want %q", imgReq.Model, tt.model)
				}
			}
		})
	}
}

func typeName(v any) string {
	return fmt.Sprintf("%T", v)
}
