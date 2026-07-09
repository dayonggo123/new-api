package gemini

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

// OpenAIContent2GeminiParts converts OpenAI message content parts into Gemini
// parts. It supports text, image_url, video_url, input_audio and file content
// types with url/base64/file_uri sources.
func OpenAIContent2GeminiParts(c *gin.Context, content []dto.MediaContent, info *relaycommon.RelayInfo) ([]dto.GeminiPart, error) {
	parts := make([]dto.GeminiPart, 0, len(content))
	for _, part := range content {
		switch part.Type {
		case dto.ContentTypeText:
			if part.Text == "" {
				continue
			}
			parts = append(parts, dto.GeminiPart{Text: part.Text})
		case dto.ContentTypeImageURL:
			geminiPart, err := buildImageGeminiPart(c, part.ToFileSource(), info)
			if err != nil {
				return nil, err
			}
			if geminiPart != nil {
				parts = append(parts, *geminiPart)
			}
		case dto.ContentTypeVideoUrl:
			geminiPart, err := buildMediaGeminiPart(c, part.ToFileSource(), "video", info)
			if err != nil {
				return nil, err
			}
			if geminiPart != nil {
				parts = append(parts, *geminiPart)
			}
		case dto.ContentTypeInputAudio:
			geminiPart, err := buildMediaGeminiPart(c, part.ToFileSource(), "audio", info)
			if err != nil {
				return nil, err
			}
			if geminiPart != nil {
				parts = append(parts, *geminiPart)
			}
		case dto.ContentTypeFile:
			geminiPart, err := buildMediaGeminiPart(c, part.ToFileSource(), "file", info)
			if err != nil {
				return nil, err
			}
			if geminiPart != nil {
				parts = append(parts, *geminiPart)
			}
		default:
			// Skip unknown content types
		}
	}
	return parts, nil
}

func buildImageGeminiPart(c *gin.Context, source types.FileSource, info *relaycommon.RelayInfo) (*dto.GeminiPart, error) {
	if source == nil {
		return nil, nil
	}
	// Google file_uri (files/xxx) is sent as fileData.
	if urlSource, ok := source.(*types.URLSource); ok && strings.HasPrefix(urlSource.URL, "files/") {
		if info == nil || info.ChannelType != constant.ChannelTypeEasyRouter {
			return &dto.GeminiPart{FileData: BuildFileData(urlSource.URL, "")}, nil
		}
	}
	base64Data, mimeType, err := service.GetBase64Data(c, source, "formatting image for Gemini")
	if err != nil {
		return nil, fmt.Errorf("get image data failed: %w", err)
	}
	inlineData, err := BuildInlineData(base64Data)
	if err != nil {
		return nil, err
	}
	inlineData.MimeType = mimeType
	return &dto.GeminiPart{InlineData: inlineData}, nil
}

func buildMediaGeminiPart(c *gin.Context, source types.FileSource, mediaType string, info *relaycommon.RelayInfo) (*dto.GeminiPart, error) {
	if source == nil {
		return nil, nil
	}
	// EasyRouter only supports inlineData; treat all sources as base64.
	forceInlineData := info != nil && info.ChannelType == constant.ChannelTypeEasyRouter
	// Google file URIs (files/xxx) are always sent as fileData regardless of
	// whether the source was parsed as a URLSource or Base64Source.
	if url := source.GetRawData(); strings.HasPrefix(url, "files/") && !forceInlineData {
		return &dto.GeminiPart{FileData: BuildFileData(url, "")}, nil
	}

	// Prefer fileData for URL-based media (video / audio / PDF) to avoid
	// downloading huge payloads into memory.
	if source.IsURL() && !forceInlineData {
		url := source.GetRawData()
		if strings.HasPrefix(url, "files/") {
			return &dto.GeminiPart{FileData: BuildFileData(url, "")}, nil
		}
		// Google Files API requires a file URI; remote URLs are not directly
		// accepted by Gemini for video/audio. Treat as fileData anyway and let
		// upstream decide.
		mimeType, err := service.GetMimeType(c, source)
		if err != nil {
			mimeType = ParseMimeType(url)
		}
		return &dto.GeminiPart{FileData: BuildFileData(url, mimeType)}, nil
	}
	base64Data, mimeType, err := service.GetBase64Data(c, source, fmt.Sprintf("formatting %s for Gemini", mediaType))
	if err != nil {
		return nil, fmt.Errorf("get %s data failed: %w", mediaType, err)
	}
	inlineData, err := BuildInlineData(base64Data)
	if err != nil {
		return nil, err
	}
	inlineData.MimeType = mimeType
	return &dto.GeminiPart{InlineData: inlineData}, nil
}

// BuildInlineData constructs a GeminiInlineData from base64 encoded bytes.
// If the input is a data URI, it is decoded into mimeType + base64 string.
func BuildInlineData(urlOrBase64 string) (*dto.GeminiInlineData, error) {
	urlOrBase64 = strings.TrimSpace(urlOrBase64)
	if urlOrBase64 == "" {
		return nil, errors.New("empty data")
	}
	mimeType := ParseMimeType(urlOrBase64)
	data := urlOrBase64
	if strings.HasPrefix(urlOrBase64, "data:") {
		format, base64String, err := service.DecodeBase64FileData(urlOrBase64)
		if err != nil {
			return nil, fmt.Errorf("decode data uri failed: %w", err)
		}
		if mimeType == "" || mimeType == "application/octet-stream" {
			mimeType = format
		}
		data = base64String
	} else if strings.HasPrefix(urlOrBase64, "http://") || strings.HasPrefix(urlOrBase64, "https://") {
		return nil, errors.New("BuildInlineData does not accept URL, use BuildFileData")
	}
	return &dto.GeminiInlineData{
		MimeType: mimeType,
		Data:     data,
	}, nil
}

// BuildFileData constructs a GeminiFileData from a file URI and optional MIME type.
func BuildFileData(uri string, mimeType string) *dto.GeminiFileData {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil
	}
	// Ensure file URI uses the full googleapis file URI format.
	if strings.HasPrefix(uri, "files/") {
		uri = "https://generativelanguage.googleapis.com/" + uri
	}
	return &dto.GeminiFileData{
		MimeType: mimeType,
		FileUri:  uri,
	}
}

// ParseMimeType guesses a MIME type from a data URL, file URI or base64 string.
func ParseMimeType(urlOrBase64 string) string {
	urlOrBase64 = strings.TrimSpace(urlOrBase64)
	if urlOrBase64 == "" {
		return ""
	}
	if strings.HasPrefix(urlOrBase64, "data:") {
		rest := urlOrBase64[len("data:"):]
		idx := strings.Index(rest, ",")
		if idx < 0 {
			idx = len(rest)
		}
		meta := rest[:idx]
		parts := strings.Split(meta, ";")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
		return ""
	}
	if strings.HasPrefix(urlOrBase64, "http://") || strings.HasPrefix(urlOrBase64, "https://") {
		return http.DetectContentType([]byte(urlOrBase64))
	}
	// Try to decode base64 and detect content type.
	decoded, err := base64.StdEncoding.DecodeString(urlOrBase64)
	if err == nil && len(decoded) > 0 {
		return http.DetectContentType(decoded)
	}
	return "application/octet-stream"
}

// isGoogleFileURI reports whether the source is a Google Files API file URI.
func isGoogleFileURI(uri string) bool {
	uri = strings.TrimSpace(uri)
	return strings.HasPrefix(uri, "files/") || strings.Contains(uri, "generativelanguage.googleapis.com/files/")
}
