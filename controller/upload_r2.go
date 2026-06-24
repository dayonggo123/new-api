package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
