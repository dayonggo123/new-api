package controller

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// UploadPromptMedia handles POST /api/prompt-media
// Accepts multipart form with "file" field and optional "media_type" field (cover_image|video)
// Saves file as base64 in database, returns media URL like /api/public/prompt-media/:id
func UploadPromptMedia(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Validate content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}

	mediaType := c.PostForm("media_type")
	if mediaType == "" {
		// auto detect from content type
		if strings.HasPrefix(contentType, "video/") {
			mediaType = "video"
		} else {
			mediaType = "cover_image"
		}
	}

	// Validate: only image/video allowed
	isImage := strings.HasPrefix(contentType, "image/")
	isVideo := strings.HasPrefix(contentType, "video/")
	if !isImage && !isVideo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only image and video files are allowed"})
		return
	}

	// Limit size: images max 20MB, videos max 100MB
	maxSize := 20 * 1024 * 1024
	if isVideo {
		maxSize = 100 * 1024 * 1024
	}
	if len(data) > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file too large, max %d MB", maxSize/1024/1024)})
		return
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	pm := &model.PromptMedia{
		MediaType:   mediaType,
		MimeType:    contentType,
		Data:        base64Data,
		CreatedTime: common.GetTimestamp(),
	}

	if err := pm.Insert(); err != nil {
		common.SysLog(fmt.Sprintf("failed to insert prompt media: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save media"})
		return
	}

	baseURL := getUploadBaseURL(c)
	url := fmt.Sprintf("%s/api/public/prompt-media/%d", baseURL, pm.Id)

	c.JSON(http.StatusOK, gin.H{
		"url":       url,
		"id":        pm.Id,
		"type":      mediaType,
		"mime_type": contentType,
	})
}

// GetPromptMedia handles GET /api/public/prompt-media/:id
// Returns the binary data with proper Content-Type (public, no auth required)
func GetPromptMedia(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media id"})
		return
	}

	pm, err := model.GetPromptMediaById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	data, err := pm.DecodeData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode media"})
		return
	}

	c.Header("Content-Type", pm.MimeType)
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, pm.MimeType, data)
}

// DeletePromptMedia handles DELETE /api/prompt-media/:id
func DeletePromptMedia(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media id"})
		return
	}

	if err := model.DeletePromptMediaById(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete media"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
