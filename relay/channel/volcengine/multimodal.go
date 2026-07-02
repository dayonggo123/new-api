package volcengine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
)

// convertOpenAIRequestMessages walks through the messages of a GeneralOpenAIRequest and
// normalizes multi-modal content parts so they are compatible with VolcEngine Ark Chat API.
// It supports image_url, video_url, input_audio, and file parts with url/base64/file_id sources.
func convertOpenAIRequestMessages(c *gin.Context, req *dto.GeneralOpenAIRequest) error {
	if req == nil || len(req.Messages) == 0 {
		return nil
	}

	for i := range req.Messages {
		message := &req.Messages[i]
		if message.IsStringContent() {
			continue
		}

		parts := message.ParseContent()
		if len(parts) == 0 {
			continue
		}

		convertedParts := make([]any, 0, len(parts))
		for _, part := range parts {
			converted, err := convertContentPart(part)
			if err != nil {
				return fmt.Errorf("message %d content conversion failed: %w", i, err)
			}
			convertedParts = append(convertedParts, converted)
		}

		message.Content = convertedParts
	}

	return nil
}

// convertContentPart normalizes a single content part to VolcEngine-compatible JSON structure.
func convertContentPart(part dto.MediaContent) (any, error) {
	switch part.Type {
	case dto.ContentTypeText:
		return map[string]any{
			"type": "text",
			"text": part.Text,
		}, nil

	case dto.ContentTypeImageURL:
		return convertImageURLPart(part)

	case dto.ContentTypeVideoUrl:
		return convertVideoURLPart(part)

	case dto.ContentTypeInputAudio:
		return convertInputAudioPart(part)

	case dto.ContentTypeFile:
		return convertFilePart(part)

	default:
		return part, nil
	}
}

func convertImageURLPart(part dto.MediaContent) (any, error) {
	img := part.GetImageMedia()
	if img == nil {
		return nil, errors.New("image_url part is missing url")
	}

	url := normalizeMediaSource(img.Url)
	if url == "" {
		return nil, errors.New("image_url is empty")
	}

	if isFileID(url) {
		return map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"file_id": url},
		}, nil
	}

	return map[string]any{
		"type":      "image_url",
		"image_url": map[string]any{"url": url},
	}, nil
}

func convertVideoURLPart(part dto.MediaContent) (any, error) {
	video := part.GetVideoUrl()
	if video == nil {
		return nil, errors.New("video_url part is missing url")
	}

	url := normalizeMediaSource(video.Url)
	if url == "" {
		return nil, errors.New("video_url is empty")
	}

	if isFileID(url) {
		return map[string]any{
			"type":      "video_url",
			"video_url": map[string]any{"file_id": url},
		}, nil
	}

	return map[string]any{
		"type":      "video_url",
		"video_url": map[string]any{"url": url},
	}, nil
}

func convertInputAudioPart(part dto.MediaContent) (any, error) {
	audio := part.GetInputAudio()
	if audio == nil {
		return nil, errors.New("input_audio part is missing data")
	}

	// If the data is already a file_id, pass it as-is.
	if isFileID(audio.Data) {
		return map[string]any{
			"type":        "input_audio",
			"input_audio": map[string]any{"file_id": audio.Data},
		}, nil
	}

	// If the data looks like a URL, pass it as a url.
	if isURL(audio.Data) {
		return map[string]any{
			"type":        "input_audio",
			"input_audio": map[string]any{"url": audio.Data},
		}, nil
	}

	// Otherwise treat as base64 data. Ensure it has a data URL prefix if missing.
	data := normalizeMediaSource(audio.Data)
	if !strings.HasPrefix(data, "data:audio/") {
		format := audio.Format
		if format == "" {
			format = "mp3"
		}
		data = fmt.Sprintf("data:audio/%s;base64,%s", format, data)
	}

	return map[string]any{
		"type": "input_audio",
		"input_audio": map[string]any{
			"data":   data,
			"format": audio.Format,
		},
	}, nil
}

func convertFilePart(part dto.MediaContent) (any, error) {
	file := part.GetFile()
	if file == nil {
		return nil, errors.New("file part is missing file data")
	}

	if file.FileId != "" {
		return map[string]any{
			"type": "file",
			"file": map[string]any{"file_id": file.FileId},
		}, nil
	}

	if isURL(file.FileData) {
		return map[string]any{
			"type": "file",
			"file": map[string]any{"url": file.FileData},
		}, nil
	}

	if file.FileName == "" {
		file.FileName = "upload.pdf"
	}

	data := normalizeMediaSource(file.FileData)
	if !strings.HasPrefix(data, "data:") {
		data = fmt.Sprintf("data:application/pdf;base64,%s", data)
	}

	return map[string]any{
		"type": "file",
		"file": map[string]any{
			"filename":  file.FileName,
			"file_data": data,
		},
	}, nil
}

// normalizeMediaSource trims spaces and ensures the source is normalized.
func normalizeMediaSource(source string) string {
	return strings.TrimSpace(source)
}

func isFileID(s string) bool {
	return strings.HasPrefix(s, "file-")
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "tos://")
}

func isBase64Data(s string) bool {
	return strings.HasPrefix(s, "data:") || strings.HasPrefix(s, "base64,")
}
