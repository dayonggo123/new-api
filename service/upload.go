package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	uploadRootDir = "./uploads"
	permanentDir  = "./uploads/permanent"
)

// UploadType 支持的文件类型
var uploadTypes = map[string]bool{
	"image":    true,
	"video":    true,
	"audio":    true,
	"file":     true,
	"material": true,
}

// UploadConfig 上传配置
type UploadConfig struct {
	Type       string // image/video/audio/file/material
	Permanent  bool
	Prefix     string
	RetainName bool
}

// UploadResult 统一上传结果
type UploadResult struct {
	URL          string `json:"url"`
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
	Type         string `json:"type"`
	Permanent    bool   `json:"permanent"`
}

// defaultUploadLimits 默认大小限制（字节）
var defaultUploadLimits = map[string]int64{
	"image":    50 << 20,  // 50MB
	"video":    500 << 20, // 500MB
	"audio":    100 << 20, // 100MB
	"file":     100 << 20, // 100MB
	"material": 100 << 20, // 100MB
}

// allowedMimeTypes 允许上传的 MIME 类型前缀
type mimeMatcher func(mime string) bool

var allowedMimeMatchers = map[string]mimeMatcher{
	"image":    func(m string) bool { return strings.HasPrefix(m, "image/") },
	"video":    func(m string) bool { return strings.HasPrefix(m, "video/") || m == "application/mp4" },
	"audio":    func(m string) bool { return strings.HasPrefix(m, "audio/") },
	"file":     func(m string) bool { return true },
	"material": func(m string) bool { return strings.HasPrefix(m, "image/") || strings.HasPrefix(m, "video/") },
}

var prefixRegexp = regexp.MustCompile("^[a-zA-Z0-9_-]*$")

// UploadFile 保存单个 multipart 文件并返回 URL
func UploadFile(ctx context.Context, c *gin.Context, file *multipart.FileHeader, cfg UploadConfig) (*UploadResult, error) {
	if err := validateUploadConfig(cfg); err != nil {
		return nil, err
	}
	if err := validateFileSize(file.Size, cfg.Type); err != nil {
		return nil, err
	}

	contentType, err := detectContentType(file)
	if err != nil {
		return nil, err
	}
	if err := validateMimeType(contentType, cfg.Type); err != nil {
		return nil, err
	}

	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open file failed: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	return uploadBytes(ctx, c, data, contentType, file.Filename, cfg)
}

// UploadFiles 批量保存 multipart 文件
func UploadFiles(ctx context.Context, c *gin.Context, files []*multipart.FileHeader, cfg UploadConfig) ([]*UploadResult, error) {
	var results []*UploadResult
	var firstErr error
	for _, file := range files {
		res, err := UploadFile(ctx, c, file, cfg)
		if err != nil && firstErr == nil {
			firstErr = err
			continue
		}
		if res != nil {
			results = append(results, res)
		}
	}
	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

// UploadBase64 保存 base64 data URI 图片
func UploadBase64(ctx context.Context, c *gin.Context, dataURI string, cfg UploadConfig) (*UploadResult, error) {
	if err := validateUploadConfig(cfg); err != nil {
		return nil, err
	}

	data, mimeType, err := parseBase64DataURI(dataURI)
	if err != nil {
		return nil, err
	}
	if err := validateFileSize(int64(len(data)), cfg.Type); err != nil {
		return nil, err
	}
	if err := validateMimeType(mimeType, cfg.Type); err != nil {
		return nil, err
	}

	return uploadBytes(ctx, c, data, mimeType, "", cfg)
}

// uploadBytes 核心保存逻辑
func uploadBytes(ctx context.Context, c *gin.Context, data []byte, contentType, originalName string, cfg UploadConfig) (*UploadResult, error) {
	ext := uploadExtFromMime(contentType)
	if ext == "bin" && originalName != "" {
		if e := filepath.Ext(originalName); e != "" {
			ext = strings.TrimPrefix(e, ".")
		}
	}

	filename := generateFilename(originalName, ext, cfg)

	// 构建相对路径
	relPath := buildRelativePath(filename, cfg)
	fullPath := filepath.Join(uploadRootDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("create directory failed: %w", err)
	}

	// 优先尝试 R2（如果启用且是图片/视频类型）
	if storage.R2Enabled() && (cfg.Type == "image" || cfg.Type == "video") {
		var url string
		var err error
		if cfg.Type == "video" {
			url, _, err = storage.UploadFileBytes(data, contentType)
		} else {
			url, _, err = storage.UploadImageBytes(data, contentType)
		}
		if err == nil {
			return &UploadResult{
				URL:          url,
				Filename:     filename,
				OriginalName: originalName,
				Size:         int64(len(data)),
				MimeType:     contentType,
				Type:         cfg.Type,
				Permanent:    cfg.Permanent,
			}, nil
		}
		common.SysLog(fmt.Sprintf("R2 upload failed, fallback to local: %v", err))
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write file failed: %w", err)
	}

	url := BuildUploadURL(c, relPath)
	return &UploadResult{
		URL:          url,
		Filename:     filename,
		OriginalName: originalName,
		Size:         int64(len(data)),
		MimeType:     contentType,
		Type:         cfg.Type,
		Permanent:    cfg.Permanent,
	}, nil
}

// BuildUploadURL 根据请求上下文构建上传文件的公网 URL
func BuildUploadURL(c *gin.Context, relativePath string) string {
	// 优先使用显式配置的公网 URL（推荐生产环境配置），避免 localhost/内网 IP 泄漏
	// 注意：调用处拼接 /uploads/%s，因此这里返回站点根（去掉 /uploads 后缀）
	if publicURL := os.Getenv("UPLOADS_PUBLIC_URL"); publicURL != "" {
		publicURL = strings.TrimRight(publicURL, "/")
		publicURL = strings.TrimSuffix(publicURL, "/uploads")
		return fmt.Sprintf("%s/uploads/%s", publicURL, relativePath)
	}
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s/uploads/%s", scheme, host, relativePath)
}

// validateUploadConfig 校验上传配置
func validateUploadConfig(cfg UploadConfig) error {
	if cfg.Type == "" {
		return fmt.Errorf("upload type is required")
	}
	if !uploadTypes[cfg.Type] {
		return fmt.Errorf("unsupported upload type: %s", cfg.Type)
	}
	if cfg.Prefix != "" && !prefixRegexp.MatchString(cfg.Prefix) {
		return fmt.Errorf("invalid prefix: %s", cfg.Prefix)
	}
	return nil
}

// validateFileSize 校验文件大小
func validateFileSize(size int64, uploadType string) error {
	limit := defaultUploadLimits[uploadType]
	if size > limit {
		return fmt.Errorf("file size exceeds %dMB limit", limit>>20)
	}
	return nil
}

// validateMimeType 校验 MIME 类型
func validateMimeType(mimeType string, uploadType string) error {
	matcher := allowedMimeMatchers[uploadType]
	if matcher != nil && !matcher(mimeType) {
		return fmt.Errorf("MIME type %s not allowed for type %s", mimeType, uploadType)
	}
	return nil
}

// detectContentType 检测文件真实类型
func detectContentType(file *multipart.FileHeader) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	return contentType, nil
}

// buildRelativePath 构建文件相对路径
func buildRelativePath(filename string, cfg UploadConfig) string {
	date := time.Now().Format("20060102")
	parts := []string{}
	if cfg.Permanent {
		parts = append(parts, "permanent", cfg.Type)
	} else if cfg.Type == "material" {
		parts = append(parts, cfg.Type)
	} else if cfg.Type == "image" || cfg.Type == "video" || cfg.Type == "audio" || cfg.Type == "file" {
		parts = append(parts, cfg.Type, date)
	} else {
		parts = append(parts, cfg.Type, date)
	}
	if cfg.Prefix != "" {
		parts = append(parts, cfg.Prefix)
	}
	parts = append(parts, filename)
	return filepath.Join(parts...)
}

// generateFilename 生成文件名
func generateFilename(originalName, ext string, cfg UploadConfig) string {
	if cfg.RetainName && originalName != "" {
		base := filepath.Base(originalName)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		return fmt.Sprintf("%s-%s.%s", base, uuid.New().String()[:8], ext)
	}
	return fmt.Sprintf("%s.%s", uuid.New().String(), ext)
}

// uploadExtFromMime 根据 MIME 类型推断扩展名
func uploadExtFromMime(mime string) string {
	mime = strings.ToLower(mime)
	switch {
	case strings.Contains(mime, "image/png"):
		return "png"
	case strings.Contains(mime, "image/jpeg"), strings.Contains(mime, "image/jpg"):
		return "jpg"
	case strings.Contains(mime, "image/gif"):
		return "gif"
	case strings.Contains(mime, "image/webp"):
		return "webp"
	case strings.Contains(mime, "image/bmp"):
		return "bmp"
	case strings.Contains(mime, "image/heic"):
		return "heic"
	case strings.Contains(mime, "image/heif"):
		return "heif"
	case strings.Contains(mime, "image/svg"):
		return "svg"
	case strings.Contains(mime, "video/mp4") || strings.Contains(mime, "application/mp4"):
		return "mp4"
	case strings.Contains(mime, "video/webm"):
		return "webm"
	case strings.Contains(mime, "video/quicktime"):
		return "mov"
	case strings.Contains(mime, "video/x-msvideo"):
		return "avi"
	case strings.Contains(mime, "video/mpeg"):
		return "mpeg"
	case strings.Contains(mime, "audio/mpeg"):
		return "mp3"
	case strings.Contains(mime, "audio/wav"):
		return "wav"
	case strings.Contains(mime, "audio/ogg"):
		return "ogg"
	case strings.Contains(mime, "audio/aac"):
		return "aac"
	case strings.Contains(mime, "audio/flac"):
		return "flac"
	case strings.Contains(mime, "application/pdf"):
		return "pdf"
	case strings.Contains(mime, "application/zip"):
		return "zip"
	case strings.Contains(mime, "text/plain"):
		return "txt"
	case strings.Contains(mime, "application/json"):
		return "json"
	default:
		return "bin"
	}
}

// parseBase64DataURI 解析 base64 data URI
func parseBase64DataURI(dataURI string) ([]byte, string, error) {
	dataURI = strings.TrimSpace(dataURI)
	if !strings.HasPrefix(dataURI, "data:") {
		return nil, "", fmt.Errorf("invalid data URI format")
	}

	rest := strings.TrimPrefix(dataURI, "data:")
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return nil, "", fmt.Errorf("invalid data URI: no comma separator")
	}

	meta := rest[:commaIdx]
	b64data := rest[commaIdx+1:]

	mime := meta
	if semiIdx := strings.Index(meta, ";"); semiIdx >= 0 {
		mime = meta[:semiIdx]
	}
	if mime == "" {
		mime = "application/octet-stream"
	}

	b64data = strings.TrimSpace(b64data)
	b64data = strings.ReplaceAll(b64data, " ", "")
	b64data = strings.ReplaceAll(b64data, "\n", "")

	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(b64data)
	}
	if err != nil {
		return nil, "", fmt.Errorf("decode base64 failed: %w", err)
	}
	return data, mime, nil
}
