package veo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	channelType int
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	a.channelType = info.ChannelType
}

func isWanXiangAI(info *relaycommon.RelayInfo) bool {
	if info.ChannelType == constant.ChannelTypeWanXiangAI {
		return true
	}
	// Fallback: detect by baseURL for cases where channel type may be misconfigured
	if strings.Contains(info.ChannelBaseUrl, "lk888.ai") {
		return true
	}
	// Fallback: detect by request path for channel test scenarios
	if strings.Contains(info.RequestURLPath, "/v1/media/generate") {
		return true
	}
	return false
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimSuffix(info.ChannelBaseUrl, "/")

	// WanXiangAI uses its own task endpoint
	if isWanXiangAI(info) {
		return baseURL + "/v1/media/generate", nil
	}

	model := info.UpstreamModelName
	// 如果 model 为空（如 GET 任务查询），尝试从请求路径中提取
	if model == "" && info.RequestURLPath != "" {
		parts := strings.Split(strings.Trim(info.RequestURLPath, "/"), "/")
		if len(parts) >= 3 && parts[0] == "uapi" && parts[1] == "v1" {
			if parts[2] == "video-gen" && len(parts) >= 4 {
				model = parts[3]
			} else {
				// For non-video-gen paths (generate_image, imagen, meta_ai), use path directly
				return baseURL + info.RequestURLPath, nil
			}
		}
	}
	path, ok := modelToPath[model]
	if !ok {
		path = fmt.Sprintf("/uapi/v1/video-gen/%s", model)
	}
	return baseURL + path, nil
}

var modelToPath = map[string]string{
	// Video generation
	"veo-3.1":          "/uapi/v1/video-gen/veo",
	"veo-3.1-fast":     "/uapi/v1/video-gen/veo",
	"veo-2":            "/uapi/v1/video-gen/veo",
	"veo-3.1-lite":     "/uapi/v1/video-gen/veo",
	"grok-3":           "/uapi/v1/video-gen/grok",
	"grok-video":       "/uapi/v1/video-gen/grok",
	"seedance-2":       "/uapi/v1/video-gen/seedance",
	"seedance-2-remix": "/uapi/v1/video-gen/seedance",
	"seedance-2-omni":  "/uapi/v1/video-gen/seedance",
	// Kling video generation
	"kling":                   "/uapi/v1/video-gen/kling",
	"kling-video-3-0":         "/uapi/v1/video-gen/kling",
	"kling-video-2-6":         "/uapi/v1/video-gen/kling",
	"kling-video-2-5":         "/uapi/v1/video-gen/kling",
	"kling-video-2-1-5s":      "/uapi/v1/video-gen/kling",
	"kling-video-2-1-10s":     "/uapi/v1/video-gen/kling",
	"kling-video-1-6-10s":     "/uapi/v1/video-gen/kling",
	"kling-video-1-6-5s":      "/uapi/v1/video-gen/kling",
	"kling-video-o1":          "/uapi/v1/video-gen/kling",
	"kling-video-motion-3":    "/uapi/v1/video-gen/kling",
	"kling-video-motion":      "/uapi/v1/video-gen/kling",
	"kling-video-3-0-edit":    "/uapi/v1/video-gen/kling",
	"kling-video-o1-edit":     "/uapi/v1/video-gen/kling",
	"kling-video-lipsync":     "/uapi/v1/video-gen/kling",
	// Image generation
	"nano-banana-pro": "/uapi/v1/generate_image",
	"nano-banana-2":   "/uapi/v1/generate_image",
	"imagen-4":        "/uapi/v1/generate_image",
	"grok-image":      "/uapi/v1/imagen/grok",
	"meta-ai-image":   "/uapi/v1/meta_ai/generate",
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error {
	if isWanXiangAI(info) {
		header.Set("Content-Type", "application/json")
		header.Set("Accept", "application/json")
		header.Set("Authorization", "Bearer "+info.ApiKey)
		return nil
	}
	header.Set("x-api-key", info.ApiKey)
	if info.ApiKey != "" {
		header.Set("Authorization", "Bearer "+info.ApiKey)
	}
	return nil
}

// ========== Kling 校验常量 ==========

var (
	klingModelsRequireVideo = map[string]bool{
		"kling-video-motion":   true,
		"kling-video-motion-3": true,
		"kling-video-3-0-edit": true,
		"kling-video-o1-edit":  true,
	}

	klingModelValidModes = map[string][]string{
		"kling-video-3-0":      {"standard", "professional"},
		"kling-video-motion-3": {"standard", "professional"},
		"kling-video-3-0-edit": {"standard", "professional"},
		"kling-video-o1-edit":  {"standard", "professional"},
		"kling-video-motion":   {"standard", "professional"},
		"kling-video-2-6":      {"standard", "professional", "professional_audio"},
		"kling-video-o1":       {"standard", "professional"},
		"kling-video-2-5":      {"relax", "standard", "professional"},
		"kling-video-lipsync":  {"default"},
		"kling-video-2-1-10s":  {"standard", "professional"},
		"kling-video-2-1-5s":   {"standard", "professional"},
		"kling-video-1-6-10s":  {"standard", "professional"},
		"kling-video-1-6-5s":   {"standard", "professional"},
	}

	maxRefImageSize     = int64(10 * 1024 * 1024)  // 10MB
	maxRefVideoSize     = int64(100 * 1024 * 1024) // 100MB
	maxRefVideoDuration = 30.0                     // 30 seconds
	maxRefImages        = 4
	maxRefVideos        = 1

	allowedImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
	}
	allowedVideoTypes = map[string]bool{
		"video/mp4":       true,
		"video/quicktime": true,
		"video/webm":      true,
	}
)

type filePart struct {
	fieldName   string
	fileName    string
	content     []byte
	contentType string
}

func isKlingModel(model string) bool {
	return strings.HasPrefix(model, "kling-video-") || model == "kling"
}

func downloadFile(fileURL string) ([]byte, string, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body failed: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" || ct == "application/octet-stream" {
		ct = http.DetectContentType(data)
	}
	return data, ct, nil
}

func parseMP4Duration(data []byte) (float64, error) {
	if len(data) < 8 {
		return 0, errors.New("file too small")
	}

	for i := 0; i < len(data)-8; {
		if i+8 > len(data) {
			break
		}
		size := binary.BigEndian.Uint32(data[i : i+4])
		boxType := string(data[i+4 : i+8])

		if boxType == "moov" {
			end := i + int(size)
			if size == 0 {
				end = len(data)
			} else if size == 1 {
				if i+16 > len(data) {
					break
				}
				end = i + int(binary.BigEndian.Uint64(data[i+8:i+16]))
			}

			for j := i + 8; j < end-8; {
				if j+8 > len(data) {
					break
				}
				subSize := binary.BigEndian.Uint32(data[j : j+4])
				subType := string(data[j+4 : j+8])
				if subType == "mvhd" {
					if j+int(subSize) > len(data) {
						break
					}
					mvhdData := data[j+8 : j+int(subSize)]
					if len(mvhdData) < 4 {
						break
					}
					version := mvhdData[0]
					var timescale, duration uint64
					if version == 0 {
						if len(mvhdData) < 20 {
							return 0, errors.New("mvhd too short")
						}
						timescale = uint64(binary.BigEndian.Uint32(mvhdData[12:16]))
						duration = uint64(binary.BigEndian.Uint32(mvhdData[16:20]))
					} else {
						if len(mvhdData) < 32 {
							return 0, errors.New("mvhd too short")
						}
						timescale = uint64(binary.BigEndian.Uint32(mvhdData[20:24]))
						duration = binary.BigEndian.Uint64(mvhdData[24:32])
					}
					if timescale == 0 {
						return 0, errors.New("invalid timescale")
					}
					return float64(duration) / float64(timescale), nil
				}
				if subSize == 0 {
					break
				}
				j += int(subSize)
			}
		}

		if size == 0 {
			break
		}
		if size == 1 {
			if i+16 > len(data) {
				break
			}
			i += int(binary.BigEndian.Uint64(data[i+8 : i+16]))
		} else {
			i += int(size)
		}
	}
	return 0, errors.New("mvhd not found")
}

func extractImageUrlsFromMessages(messages []dto.Message) []string {
	var urls []string
	for _, msg := range messages {
		for _, mc := range msg.ParseContent() {
			if mc.Type == "image_url" {
				if img := mc.GetImageMedia(); img != nil && img.Url != "" {
					urls = append(urls, img.Url)
				}
			}
		}
	}
	return urls
}

func validateKlingInput(model string, files []filePart, mode string) error {
	if !isKlingModel(model) {
		return nil
	}

	// 1. mode validity
	if mode != "" {
		if validModes, ok := klingModelValidModes[model]; ok {
			found := false
			for _, vm := range validModes {
				if vm == mode {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("INVALID_INPUT: Mode '%s' is not valid for model '%s'. Allowed: %s",
					mode, model, strings.Join(validModes, ", "))
			}
		}
	}

	// 2. motion/edit models require ref_video
	if klingModelsRequireVideo[model] {
		hasVideo := false
		for _, f := range files {
			if f.fieldName == "ref_videos" {
				hasVideo = true
				break
			}
		}
		if !hasVideo {
			return fmt.Errorf("INVALID_VIDEO_FILE: Model '%s' requires at least one reference video.", model)
		}
	}

	// 3. file count limits
	imgCount := 0
	vidCount := 0
	for _, f := range files {
		switch f.fieldName {
		case "ref_images", "files":
			imgCount++
		case "ref_videos":
			vidCount++
		}
	}
	if imgCount > maxRefImages {
		return fmt.Errorf("INVALID_VIDEO_FILE: Too many reference images. Max: %d, Got: %d", maxRefImages, imgCount)
	}
	if vidCount > maxRefVideos {
		return fmt.Errorf("INVALID_VIDEO_FILE: Too many reference videos. Max: %d, Got: %d", maxRefVideos, vidCount)
	}

	// 4. file format / size / duration
	for _, f := range files {
		switch f.fieldName {
		case "ref_images", "files":
			if !allowedImageTypes[f.contentType] {
				return fmt.Errorf("INVALID_VIDEO_FILE: Unsupported image format '%s'. Allowed: JPG, PNG", f.contentType)
			}
			if int64(len(f.content)) > maxRefImageSize {
				return fmt.Errorf("FILE_TOO_LARGE: Image file size exceeds limit of %dMB", maxRefImageSize/(1024*1024))
			}
		case "ref_videos":
			if !allowedVideoTypes[f.contentType] {
				return fmt.Errorf("INVALID_VIDEO_FILE: Unsupported video format '%s'. Allowed: MP4, MOV, WebM", f.contentType)
			}
			if int64(len(f.content)) > maxRefVideoSize {
				return fmt.Errorf("FILE_TOO_LARGE: Video file size exceeds limit of %dMB", maxRefVideoSize/(1024*1024))
			}
			duration, err := parseMP4Duration(f.content)
			if err == nil && duration > maxRefVideoDuration {
				return fmt.Errorf("VIDEO_DURATION_TOO_LONG: Reference video duration (%.0fs) exceeds maximum allowed (%.0fs)", duration, maxRefVideoDuration)
			}
		}
	}

	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}

	model := info.UpstreamModelName

	prompt := ""
	for i := len(request.Messages) - 1; i >= 0; i-- {
		msg := request.Messages[i]
		if msg.Role == "user" {
			if content, ok := msg.Content.(string); ok && content != "" {
				prompt = content
				break
			}
		}
	}

	// Collect reference file URLs
	var refUrls []struct {
		url       string
		fieldName string
	}
	for _, imgUrl := range extractImageUrlsFromMessages(request.Messages) {
		refUrls = append(refUrls, struct{ url, fieldName string }{url: imgUrl, fieldName: "ref_images"})
	}
	for _, u := range request.RefImages {
		refUrls = append(refUrls, struct{ url, fieldName string }{url: u, fieldName: "ref_images"})
	}
	for _, u := range request.RefVideos {
		refUrls = append(refUrls, struct{ url, fieldName string }{url: u, fieldName: "ref_videos"})
	}

	// Download reference files
	var files []filePart
	for _, ru := range refUrls {
		data, ct, err := downloadFile(ru.url)
		if err != nil {
			return nil, fmt.Errorf("download ref file failed (%s): %w", ru.url, err)
		}
		filename := path.Base(ru.url)
		if filename == "" || filename == "." {
			filename = "ref"
		}
		files = append(files, filePart{
			fieldName:   ru.fieldName,
			fileName:    filename,
			content:     data,
			contentType: ct,
		})
	}

	if err := validateKlingInput(model, files, ""); err != nil {
		return nil, err
	}

	common.SysLog(fmt.Sprintf("veo ConvertOpenAIRequest: model=%s prompt=%q size=%v messages=%d refFiles=%d",
		model, prompt, request.Size, len(request.Messages), len(files)))

	buf, _, err := buildMultipartBody(model, prompt, request.Size, "", "", "", files)
	return buf, err
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	model := request.Model
	if model == "" {
		model = info.UpstreamModelName
	}

	// WanXiangAI uses JSON body instead of multipart
	if isWanXiangAI(info) {
		return a.buildWanXiangJSONBody(model, request.Prompt, request.Size)
	}

	buf, _, err := buildMultipartBody(model, request.Prompt, request.Size, "", "", "", nil)
	return buf, err
}

func (a *Adaptor) buildWanXiangJSONBody(model, prompt, size string) (any, error) {
	params := make(map[string]interface{})

	// imageSize mapping
	imageSize := "1K"
	switch {
	case strings.HasPrefix(size, "480") || strings.HasPrefix(size, "720"):
		imageSize = "1K"
	case strings.HasPrefix(size, "1080"):
		imageSize = "2K"
	case strings.Contains(size, "x"):
		// Default for size like "1024x1024"
		imageSize = "1K"
	case strings.HasPrefix(size, "0.5"):
		imageSize = "0.5K"
	case strings.HasPrefix(size, "1K"):
		imageSize = "1K"
	case strings.HasPrefix(size, "2K"):
		imageSize = "2K"
	case strings.HasPrefix(size, "4K"):
		imageSize = "4K"
	}
	params["imageSize"] = imageSize

	// aspectRatio
	if size != "" {
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			w, _ := strconv.Atoi(parts[0])
			h, _ := strconv.Atoi(parts[1])
			if w > 0 && h > 0 {
				ratio := float64(w) / float64(h)
				switch {
				case ratio > 1.3:
					params["aspectRatio"] = "16:9"
				case ratio < 0.8:
					params["aspectRatio"] = "9:16"
				default:
					params["aspectRatio"] = "1:1"
				}
			}
		}
	}

	// quality for veo3.1-lite
	if model == "veo3.1-lite" {
		params["quality"] = "sd"
	}

	// generation_mode for veo3.1 non-lite
	if strings.Contains(model, "veo3.1") && !strings.Contains(model, "lite") {
		params["generation_mode"] = "fast"
	}

	body := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"params": params,
	}
	return body, nil
}

var imageModels = map[string]bool{
	"nano-banana-pro": true,
	"nano-banana-2":   true,
	"imagen-4":        true,
	"grok-image":      true,
	"meta-ai-image":   true,
}

func buildMultipartBody(model, prompt, resolution, mode, aspectRatio, duration string, files []filePart) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("prompt", prompt)
	writer.WriteField("model", model)

	if mode != "" {
		writer.WriteField("mode", mode)
	}
	if aspectRatio != "" {
		writer.WriteField("aspect_ratio", aspectRatio)
	}
	if duration != "" {
		writer.WriteField("duration", duration)
	}

	isImageModel := imageModels[model]
	switch {
	case resolution == "":
		// no resolution field
	case isImageModel:
		switch {
		case strings.HasPrefix(resolution, "480"):
			writer.WriteField("resolution", "1K")
		case strings.HasPrefix(resolution, "720"):
			writer.WriteField("resolution", "1K")
		case strings.HasPrefix(resolution, "1080"):
			writer.WriteField("resolution", "2K")
		case strings.Contains(resolution, "x"):
			writer.WriteField("resolution", "1K")
		case strings.HasPrefix(resolution, "1K") || strings.HasPrefix(resolution, "2K") || strings.HasPrefix(resolution, "4K"):
			writer.WriteField("resolution", resolution)
		default:
			writer.WriteField("resolution", resolution)
		}
	case strings.HasPrefix(resolution, "480"):
		writer.WriteField("resolution", "480p")
	case strings.HasPrefix(resolution, "720"):
		writer.WriteField("resolution", "720p")
	case strings.HasPrefix(resolution, "1080"):
		writer.WriteField("resolution", "1080p")
	case strings.Contains(resolution, "x"):
		writer.WriteField("resolution", "720p")
	case strings.HasPrefix(resolution, "square-"):
		writer.WriteField("resolution", "720p")
	}

	for _, f := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, f.fieldName, f.fileName))
		h.Set("Content-Type", f.contentType)
		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, "", fmt.Errorf("create part failed: %w", err)
		}
		if _, err := part.Write(f.content); err != nil {
			return nil, "", fmt.Errorf("write part failed: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer failed: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	// WanXiangAI uses standard JSON API request, not multipart
	if isWanXiangAI(info) {
		return channel.DoApiRequest(a, c, info, requestBody)
	}

	if buf, ok := requestBody.(*bytes.Buffer); ok && buf.Len() > 0 {
		contentType := "multipart/form-data; boundary=" + extractBoundaryFromMultipartBody(buf)
		c.Set("veo_multipart_content_type", contentType)
		return channel.DoFormRequestWithContentType(a, c, info, buf, contentType)
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return channel.DoApiRequest(a, c, info, requestBody)
	}

	cachedBody, err := storage.Bytes()
	if err != nil {
		return channel.DoApiRequest(a, c, info, requestBody)
	}

	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
			return nil, fmt.Errorf("unmarshal json body failed: %w", err)
		}

		model := info.UpstreamModelName

		prompt := ""
		if p, ok := bodyMap["prompt"].(string); ok {
			prompt = p
		}

		resolution := ""
		if r, ok := bodyMap["resolution"].(string); ok {
			resolution = r
		}
		mode := ""
		if m, ok := bodyMap["mode"].(string); ok {
			mode = m
		}
		aspectRatio := ""
		if ar, ok := bodyMap["aspect_ratio"].(string); ok {
			aspectRatio = ar
		}
		duration := ""
		if d, ok := bodyMap["duration"].(string); ok {
			duration = d
		}

		var files []filePart
		for fieldName, key := range map[string]string{
			"ref_images": "ref_images",
			"ref_videos": "ref_videos",
		} {
			if arr, ok := bodyMap[key].([]interface{}); ok {
				for _, item := range arr {
					if urlStr, ok := item.(string); ok && urlStr != "" {
						data, ct, err := downloadFile(urlStr)
						if err != nil {
							return nil, fmt.Errorf("download ref file failed (%s): %w", urlStr, err)
						}
						filename := path.Base(urlStr)
						if filename == "" || filename == "." {
							filename = "ref"
						}
						files = append(files, filePart{
							fieldName:   fieldName,
							fileName:    filename,
							content:     data,
							contentType: ct,
						})
					}
				}
			}
		}

		if err := validateKlingInput(model, files, mode); err != nil {
			return nil, err
		}

		multipartBody, multipartContentType, err := buildMultipartBody(model, prompt, resolution, mode, aspectRatio, duration, files)
		if err != nil {
			return nil, err
		}

		c.Set("veo_multipart_content_type", multipartContentType)
		return channel.DoFormRequestWithContentType(a, c, info, multipartBody, multipartContentType)

	} else if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, fmt.Errorf("parse multipart form failed: %w", err)
		}
		defer formData.RemoveAll()

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		model := info.UpstreamModelName
		writer.WriteField("model", model)

		mode := ""
		if v, ok := formData.Value["prompt"]; ok && len(v) > 0 {
			writer.WriteField("prompt", v[0])
		}
		if v, ok := formData.Value["resolution"]; ok && len(v) > 0 {
			writer.WriteField("resolution", v[0])
		}
		if v, ok := formData.Value["aspect_ratio"]; ok && len(v) > 0 {
			writer.WriteField("aspect_ratio", v[0])
		}
		if v, ok := formData.Value["mode"]; ok && len(v) > 0 {
			mode = v[0]
			writer.WriteField("mode", v[0])
		}
		if v, ok := formData.Value["mode_image"]; ok && len(v) > 0 {
			writer.WriteField("mode_image", v[0])
		}
		if v, ok := formData.Value["duration"]; ok && len(v) > 0 {
			writer.WriteField("duration", v[0])
		}

		var files []filePart
		for fieldName, fileHeaders := range formData.File {
			if fieldName != "ref_images" && fieldName != "files" && fieldName != "ref_videos" && fieldName != "ref_audios" {
				continue
			}
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(f)
				f.Close()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					ct = http.DetectContentType(data)
				}
				files = append(files, filePart{
					fieldName:   fieldName,
					fileName:    fh.Filename,
					content:     data,
					contentType: ct,
				})

				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					continue
				}
				part.Write(data)
			}
		}

		if err := validateKlingInput(model, files, mode); err != nil {
			return nil, err
		}

		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close multipart writer failed: %w", err)
		}

		multipartBody := &buf
		multipartContentType := writer.FormDataContentType()

		c.Set("veo_multipart_content_type", multipartContentType)
		return channel.DoFormRequestWithContentType(a, c, info, multipartBody, multipartContentType)
	}

	return channel.DoApiRequest(a, c, info, requestBody)
}

func extractBoundaryFromMultipartBody(buf *bytes.Buffer) string {
	data := buf.String()
	eol := strings.Index(data, "\r\n")
	if eol == -1 {
		return ""
	}
	firstLine := data[:eol]
	const prefix = "--"
	if !strings.HasPrefix(firstLine, prefix) {
		return ""
	}
	return firstLine[len(prefix):]
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if resp == nil {
		return nil, types.NewError(errors.New("empty response"), types.ErrorCodeBadResponse)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewError(readErr, types.ErrorCodeReadResponseBodyFailed)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, types.NewError(fmt.Errorf("bad status: %d body: %s", resp.StatusCode, string(body)), types.ErrorCodeBadResponse)
	}

	// WanXiangAI response format: {code, msg, data: {task_id}}
	if a.channelType == constant.ChannelTypeWanXiangAI {
		var wxResp wanxiangSubmitResponse
		if unmarshalErr := common.Unmarshal(body, &wxResp); unmarshalErr != nil {
			return nil, types.NewError(fmt.Errorf("unmarshal failed: %w body: %s", unmarshalErr, string(body)), types.ErrorCodeBadResponseBody)
		}
		if wxResp.Code != 200 {
			return nil, types.NewError(fmt.Errorf("submit failed: %s", wxResp.Msg), types.ErrorCodeBadResponse)
		}

		openaiResp := dto.NewOpenAIVideo()
		openaiResp.ID = info.PublicTaskID
		if openaiResp.ID == "" {
			openaiResp.ID = fmt.Sprintf("task_%v", wxResp.Data.TaskID)
		}
		openaiResp.Model = info.OriginModelName
		openaiResp.CreatedAt = info.StartTime.Unix()

		c.JSON(http.StatusOK, openaiResp)

		usage = &dto.Usage{}
		return usage, nil
	}

	var submitResp submitResponse
	if err := common.Unmarshal(body, &submitResp); err != nil {
		return nil, types.NewError(fmt.Errorf("unmarshal failed: %w body: %s", err, string(body)), types.ErrorCodeBadResponseBody)
	}

	openaiResp := dto.NewOpenAIVideo()
	openaiResp.ID = submitResp.UUID
	openaiResp.Model = submitResp.ModelName
	openaiResp.CreatedAt = info.StartTime.Unix()

	c.JSON(http.StatusOK, openaiResp)

	usage = &dto.Usage{}
	return usage, nil
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

type submitResponse struct {
	ID               int    `json:"id"`
	UUID             string `json:"uuid"`
	UserID           int    `json:"user_id"`
	ModelName        string `json:"model_name"`
	InputText        string `json:"input_text"`
	Type             string `json:"type"`
	Status           int    `json:"status"`
	StatusDesc       string `json:"status_desc"`
	StatusPercentage int    `json:"status_percentage"`
	ErrorCode        string `json:"error_code"`
	ErrorMessage     string `json:"error_message"`
	EstimatedCredit  int    `json:"estimated_credit"`
	MediaType        string `json:"media_type"`
	CreatedAt        string `json:"created_at"`
	DelaySeconds     int    `json:"delay_seconds"`
}

type wanxiangSubmitResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskID interface{} `json:"task_id"`
	} `json:"data"`
}

var ModelList = []string{
	// Video generation
	"veo-3.1",
	"veo-3.1-fast",
	"veo-2",
	"veo-3.1-lite",
	"grok-3",
	"grok-video",
	"seedance-2",
	"seedance-2-remix",
	"seedance-2-omni",
	// Kling video generation
	"kling",
	"kling-video-3-0",
	"kling-video-2-6",
	"kling-video-2-5",
	"kling-video-2-1-5s",
	"kling-video-2-1-10s",
	"kling-video-1-6-10s",
	"kling-video-1-6-5s",
	"kling-video-o1",
	"kling-video-motion-3",
	"kling-video-motion",
	"kling-video-3-0-edit",
	"kling-video-o1-edit",
	"kling-video-lipsync",
	// Image generation
	"nano-banana-pro",
	"nano-banana-2",
	"imagen-4",
	"grok-image",
	"meta-ai-image",
}

var ChannelName = "GeminiGen"
