package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/storage"
	"github.com/gin-gonic/gin"
)

// UploadImageR2 handles POST /uapi/v1/r2/upload-image
// Accepts multipart form with "image" field.
// If R2 is configured, uploads to R2 and returns a presigned URL.
// If R2 is NOT configured, saves to local uploads/ directory and returns a public URL.
func UploadImageR2(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to get image file: %v", err),
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to open uploaded file: %v", err),
		})
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")

	// R2 已配置 -> 上传到 R2
	if storage.R2Enabled() {
		url, key, err := storage.UploadImage(file, contentType, fileHeader.Size)
		if err != nil {
			common.SysLog(fmt.Sprintf("UploadImageR2 failed: %v", err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"url":        url,
			"key":        key,
			"expires_in": int(storage.R2URLExpiry().Seconds()),
		})
		return
	}

	// R2 没配 -> 保存到本地 uploads/ 目录
	common.SysLog("R2 not configured, falling back to local uploads/")
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".png"
	}
	filename := fmt.Sprintf("ref_%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join("uploads", filename)
	if err := os.MkdirAll("uploads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to create uploads/ directory: %v", err),
		})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to read uploaded file: %v", err),
		})
		return
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to save file: %v", err),
		})
		return
	}

	// 构建公网 URL
	uploadsPublicURL := os.Getenv("UPLOADS_PUBLIC_URL")
	if uploadsPublicURL == "" {
		uploadsPublicURL = "http://localhost:3000/uploads/"
	}
	if !strings.HasSuffix(uploadsPublicURL, "/") {
		uploadsPublicURL += "/"
	}
	publicURL := uploadsPublicURL + filename

	common.SysLog(fmt.Sprintf("local upload: %s -> %s", filePath, publicURL))
	c.JSON(http.StatusOK, gin.H{
		"url":        publicURL,
		"key":        filePath,
		"expires_in": 0,
	})
}

// UploadVideoR2 handles POST /uapi/v1/r2/upload-video
// Accepts multipart form with "video" field.
// If R2 is configured, uploads to R2 with the correct video/* Content-Type and returns a presigned URL.
// If R2 is NOT configured, falls back to local ./uploads/videos/ directory.
func UploadVideoR2(c *gin.Context) {
	fileHeader, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to get video file: %v", err),
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to open uploaded file: %v", err),
		})
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		// Read a small buffer to detect real MIME type
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType = http.DetectContentType(buf[:n])
		// Rewind so UploadFileBytes can read from the beginning
		if seeker, ok := file.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
		}
	}

	// R2 已配置 -> 上传到 R2
	if storage.R2Enabled() {
		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("failed to read uploaded file: %v", err),
			})
			return
		}
		url, key, err := storage.UploadFileBytes(data, contentType)
		if err != nil {
			common.SysLog(fmt.Sprintf("UploadVideoR2 failed: %v", err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"url":        url,
			"key":        key,
			"expires_in": int(storage.R2URLExpiry().Seconds()),
		})
		return
	}

	// R2 没配 -> 保存到本地 uploads/videos 目录
	common.SysLog("R2 not configured, falling back to local uploads/videos/")
	videoDir := filepath.Join("uploads", "videos")
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to create video directory: %v", err),
		})
		return
	}

	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".mp4"
	}
	filename := fmt.Sprintf("video_%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(videoDir, filename)

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to read uploaded file: %v", err),
		})
		return
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to save file: %v", err),
		})
		return
	}

	uploadsPublicURL := os.Getenv("UPLOADS_PUBLIC_URL")
	if uploadsPublicURL == "" {
		uploadsPublicURL = "http://localhost:3000/uploads/"
	}
	if !strings.HasSuffix(uploadsPublicURL, "/") {
		uploadsPublicURL += "/"
	}
	publicURL := uploadsPublicURL + "videos/" + filename

	common.SysLog(fmt.Sprintf("local video upload: %s -> %s", filePath, publicURL))
	c.JSON(http.StatusOK, gin.H{
		"url":        publicURL,
		"key":        filePath,
		"expires_in": 0,
	})
}

// UploadImageR2Base64 handles POST /uapi/v1/r2/upload-image/base64
// Accepts JSON body: {"image": "data:image/png;base64,xxxxx"}
func UploadImageR2Base64(c *gin.Context) {
	if !storage.R2Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "R2 storage is not configured",
		})
		return
	}

	var body struct {
		Image string `json:"image" binding:"required"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid JSON body: %v", err),
		})
		return
	}

	data, ext, err := parseDataURL(body.Image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid base64 image data: %v", err),
		})
		return
	}
	if len(data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "empty image data",
		})
		return
	}

	contentType := "image/" + ext
	url, key, err := storage.UploadImageBytes(data, contentType)
	if err != nil {
		common.SysLog(fmt.Sprintf("UploadImageR2Base64 failed: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        url,
		"key":        key,
		"expires_in": int(storage.R2URLExpiry().Seconds()),
	})
}

// ProxyR2Object handles GET /api/public/r2
// 永久服务器地址：服务端根据 bucket/key 实时生成新的 presigned URL 后 302 重定向到 R2，
// 客户端直连 R2 拉取对象，不占用服务端带宽，且不依赖任何历史签名是否过期。
// 用于模板市场等场景的持久封面/资源 URL（DB 中只存 r2://bucket/key 短路径）。
//
// 对于视频文件（.mp4/.webm/.mov 等），改为服务端直接代理返回：
//  1. 修正 R2 上 application/octet-stream 导致的浏览器无法解码问题；
//  2. 添加 Access-Control-Allow-* 头部，允许跨域 <video> / fetch 访问。
func ProxyR2Object(c *gin.Context) {
	if !storage.R2Enabled() {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "R2 storage is not configured",
		})
		return
	}

	bucket := strings.TrimSpace(c.Query("bucket"))
	key := strings.TrimSpace(c.Query("key"))
	if bucket == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bucket and key are required",
		})
		return
	}
	// 防御性限制：R2 key 长度通常 < 512，避免异常入参
	if len(bucket) > 128 || len(key) > 2048 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid bucket or key",
		})
		return
	}

	// 对视频对象直接代理，确保浏览器拿到正确的 Content-Type 和 CORS 头
	if isR2VideoKey(key) {
		body, size, ct, err := storage.GetObject(bucket, key)
		if err != nil {
			common.SysLog(fmt.Sprintf("ProxyR2Object get object failed bucket=%s key=%s: %v", bucket, key, err))
			c.JSON(http.StatusNotFound, gin.H{
				"error": "object not found or access denied",
			})
			return
		}
		defer body.Close()

		setR2ProxyCORSHeaders(c)
		if ct != "" {
			c.Header("Content-Type", ct)
		}
		if size > 0 {
			c.Header("Content-Length", strconv.FormatInt(size, 10))
		}
		c.Header("Accept-Ranges", "bytes")
		c.Header("Cache-Control", "public, max-age=300")

		// 处理 CORS 预检
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}

		io.Copy(c.Writer, body)
		return
	}

	signed, err := storage.PresignBucketObject(bucket, key)
	if err != nil {
		common.SysLog(fmt.Sprintf("ProxyR2Object presign failed bucket=%s key=%s: %v", bucket, key, err))
		c.JSON(http.StatusNotFound, gin.H{
			"error": "object not found or access denied",
		})
		return
	}

	// 302 可被浏览器/CDN 缓存，减少服务端调用次数（签名有效期默认 600s，缓存 300s 安全）
	c.Header("Cache-Control", "public, max-age=300")
	c.Redirect(http.StatusFound, signed)
}

// isR2VideoKey 根据扩展名判断 R2 key 是否为视频对象
func isR2VideoKey(key string) bool {
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".mp4", ".webm", ".mov", ".avi", ".mkv", ".m4v":
		return true
	default:
		return false
	}
}

// setR2ProxyCORSHeaders 设置允许跨域访问 R2 代理资源的 CORS 头
func setR2ProxyCORSHeaders(c *gin.Context) {
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = "*"
	}
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Range")
	c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range")
	c.Header("Vary", "Origin")
}

// PresignImageR2 handles POST /uapi/v1/r2/presign
// Accepts JSON body: {"key": "tmp/uuid.png"}, returns a fresh presigned URL.
func PresignImageR2(c *gin.Context) {
	if !storage.R2Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "R2 storage is not configured",
		})
		return
	}

	var body struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid JSON body: %v", err),
		})
		return
	}

	url, err := storage.PresignedURL(strings.TrimSpace(body.Key))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        url,
		"key":        body.Key,
		"expires_in": int(storage.R2URLExpiry().Seconds()),
	})
}
