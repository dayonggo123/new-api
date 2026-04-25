package controller

import (
	"fmt"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const uploadDir = "./uploads"

// UploadImages handles POST /uapi/v1/upload_images
// Accepts multipart form with "images" field (multiple files)
// Returns URLs for the uploaded files
func UploadImages(c *gin.Context) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to create upload directory: %v", err),
		})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to parse multipart form: %v", err),
		})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no images provided in 'images' field",
		})
		return
	}

	// Get base URL for constructing public URLs
	baseURL := getUploadBaseURL(c)

	var urls []string
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue
		}

		// Detect and validate content type
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" || contentType == "application/octet-stream" {
			contentType = http.DetectContentType(data)
		}
		if !strings.HasPrefix(contentType, "image/") {
			continue
		}

		// Generate unique filename preserving extension
		ext := extFromMime(contentType)
		filename := fmt.Sprintf("%s.%s", uuid.New().String(), ext)
		filePath := filepath.Join(uploadDir, filename)

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			common.SysLog(fmt.Sprintf("failed to write uploaded file: %v", err))
			continue
		}

		url := fmt.Sprintf("%s/uploads/%s", baseURL, filename)
		urls = append(urls, url)
	}

	if len(urls) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no valid image files were uploaded",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"urls": urls,
	})
}

func getUploadBaseURL(c *gin.Context) string {
	// Use X-Forwarded-Host/Proto if behind proxy, otherwise infer from request
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host
}

func extFromMime(mime string) string {
	switch strings.ToLower(mime) {
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/bmp":
		return "bmp"
	case "image/heic":
		return "heic"
	case "image/heif":
		return "heif"
	default:
		return "bin"
	}
}

// UploadImagesJSON handles POST /uapi/v1/upload_images/json
// Accepts JSON body: {"images": ["data:image/png;base64,xxxxx", ...]}
// Returns: {"urls": ["https://host/uploads/uuid.png", ...]}
func UploadImagesJSON(c *gin.Context) {
	var body struct {
		Images []string `json:"images"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if len(body.Images) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no images provided"})
		return
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory: " + err.Error()})
		return
	}

	baseURL := getUploadBaseURL(c)
	var urls []string

	for _, img := range body.Images {
		data, ext, err := parseDataURL(img)
		if err != nil {
			continue
		}
		if len(data) == 0 {
			continue
		}

		filename := fmt.Sprintf("%s.%s", uuid.New().String(), ext)
		filePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			common.SysLog(fmt.Sprintf("upload json: failed to write file: %v", err))
			continue
		}
		urls = append(urls, fmt.Sprintf("%s/uploads/%s", baseURL, filename))
	}

	if len(urls) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid image data provided"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"urls": urls})
}

// parseDataURL extracts raw bytes and extension from a data URL.
// Accepts: "data:image/png;base64,xxxxx" or "image/png;base64,xxxxx"
func parseDataURL(dataURL string) ([]byte, string, error) {
	header := "data:"
	if !strings.HasPrefix(dataURL, header) {
		// Allow shorthand without "data:" prefix
		header = ""
	}
	rest := strings.TrimPrefix(dataURL, header)

	// Find the comma
	commaIdx := strings.LastIndex(rest, ",")
	if commaIdx < 0 {
		return nil, "", fmt.Errorf("no comma separator")
	}

	meta := rest[:commaIdx]
	b64data := rest[commaIdx+1:]

	// Extract MIME type from meta (e.g. "image/png;base64")
	mime := meta
	if semiIdx := strings.Index(meta, ";"); semiIdx >= 0 {
		mime = strings.TrimPrefix(meta[:semiIdx], "image/")
	}

	// Strip base64 padding markers
	b64data = strings.TrimRight(b64data, " \t\n")
	b64data = strings.ReplaceAll(b64data, " ", "")
	b64data = strings.ReplaceAll(b64data, "\n", "")
	b64data = strings.ReplaceAll(b64data, "\t", "")

	decoded, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		// Try URL-safe base64
		decoded, err = base64.URLEncoding.DecodeString(b64data)
	}
	return decoded, extFromMime("image/"+mime), err
}
