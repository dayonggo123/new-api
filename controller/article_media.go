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

// UploadArticleMedia handles POST /api/article-media
// Accepts multipart form with "file" field and optional "media_type" field (cover_image|content_image)
// Saves file as base64 in database, returns media URL like /api/public/article-media/:id
func UploadArticleMedia(c *gin.Context) {
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
		mediaType = "content_image"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large, max 20 MB"})
		return
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	am := &model.ArticleMedia{
		MediaType:   mediaType,
		MimeType:    contentType,
		Data:        base64Data,
		CreatedTime: common.GetTimestamp(),
	}

	if err := am.Insert(); err != nil {
		common.SysLog(fmt.Sprintf("failed to insert article media: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save media"})
		return
	}

	baseURL := getUploadBaseURL(c)
	url := fmt.Sprintf("%s/api/public/article-media/%d", baseURL, am.Id)

	c.JSON(http.StatusOK, gin.H{
		"url":       url,
		"id":        am.Id,
		"type":      mediaType,
		"mime_type": contentType,
	})
}

// GetArticleMedia handles GET /api/public/article-media/:id
// Returns the binary data with proper Content-Type (public, no auth required)
func GetArticleMedia(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media id"})
		return
	}

	am, err := model.GetArticleMediaById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	data, err := am.DecodeData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode media"})
		return
	}

	c.Header("Content-Type", am.MimeType)
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, am.MimeType, data)
}

// DeleteArticleMedia handles DELETE /api/article-media/:id
func DeleteArticleMedia(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media id"})
		return
	}

	if err := model.DeleteArticleMediaById(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete media"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
