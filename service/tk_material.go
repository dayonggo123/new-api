package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	tkMaterialUploadDir  = "./uploads/tk_materials"
	tkMaterialPermDir    = "./uploads/permanent"
	tkMaterialPermSubDir = "tk_materials"
)

// TKMaterialUploadResult stores the result of a single uploaded file.
type TKMaterialUploadResult struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Filename     string `json:"filename"`
	FileType     string `json:"file_type"`
	Size         int64  `json:"size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Error        string `json:"error,omitempty"`
}

// UploadTKMaterialFiles saves uploaded image files and returns metadata.
// If permanent is true, files are stored under ./uploads/permanent/tk_materials/.
func UploadTKMaterialFiles(c *gin.Context, files []*multipart.FileHeader, permanent bool) ([]TKMaterialUploadResult, error) {
	dir := tkMaterialUploadDir
	if permanent {
		dir = filepath.Join(tkMaterialPermDir, tkMaterialPermSubDir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	baseURL := getTKMaterialUploadBaseURL(c)
	results := make([]TKMaterialUploadResult, 0, len(files))

	for _, fh := range files {
		res := TKMaterialUploadResult{}
		file, err := fh.Open()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		contentType := fh.Header.Get("Content-Type")
		if contentType == "" || contentType == "application/octet-stream" {
			contentType = http.DetectContentType(data)
		}
		if !strings.HasPrefix(contentType, "image/") {
			res.Error = "not an image file"
			results = append(results, res)
			continue
		}

		ext := extFromMime(contentType)
		filename := fmt.Sprintf("%s.%s", uuid.New().String(), ext)
		filePath := filepath.Join(dir, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		urlPath := filename
		if permanent {
			urlPath = tkMaterialPermSubDir + "/" + filename
		}
		res.URL = fmt.Sprintf("%s/uploads/%s", baseURL, urlPath)
		res.ThumbnailURL = res.URL
		res.Filename = filename
		res.FileType = contentType
		res.Size = int64(len(data))
		res.Width, res.Height = 0, 0
		results = append(results, res)
	}
	return results, nil
}

func getTKMaterialUploadBaseURL(c *gin.Context) string {
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

// SaveTKMaterialFromUpload persists a single uploaded file as a TKMaterial row.
func SaveTKMaterialFromUpload(category string, res *TKMaterialUploadResult, source string) (*model.TKMaterial, error) {
	if category == "" {
		category = "其他"
	}
	m := &model.TKMaterial{
		Category:     category,
		URL:          res.URL,
		ThumbnailURL: res.ThumbnailURL,
		Filename:     res.Filename,
		FileType:     res.FileType,
		Size:         res.Size,
		Width:        res.Width,
		Height:       res.Height,
		Source:       source,
		Status:       1,
	}
	if err := model.TKMaterialCreate(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SaveTKMaterialFromNotion saves a material imported from Notion.
func SaveTKMaterialFromNotion(category, url, notionPageID string) (*model.TKMaterial, error) {
	exists, err := model.TKMaterialExistsByURL(url, category)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, nil
	}
	m := &model.TKMaterial{
		Category:     category,
		URL:          url,
		ThumbnailURL: url,
		Filename:     filepath.Base(url),
		FileType:     "",
		Source:       "notion",
		NotionPageID: notionPageID,
		Status:       1,
	}
	if err := model.TKMaterialCreate(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TKMaterialImportNotionResult is the summary of a Notion import.
type TKMaterialImportNotionResult struct {
	TotalPages     int      `json:"total_pages"`
	ImportedCount  int      `json:"imported_count"`
	DuplicateCount int      `json:"duplicate_count"`
	ErrorCount     int      `json:"error_count"`
	Errors         []string `json:"errors,omitempty"`
}

// ImportTKMaterialsFromNotion fetches the specified Notion database and imports
// image URLs from the given scene columns.
func ImportTKMaterialsFromNotion(token, databaseID string, categories []string) (*TKMaterialImportNotionResult, error) {
	client := NewNotionClient(token)
	pages, err := client.QueryDatabaseAll(databaseID)
	if err != nil {
		return nil, err
	}

	res := &TKMaterialImportNotionResult{
		TotalPages: len(pages),
	}

	for _, page := range pages {
		pageID := page.ID
		for _, category := range categories {
			urls := extractNotionImageURLs(page.Properties[category])
			for _, url := range urls {
				if url == "" {
					continue
				}
				exists, err := model.TKMaterialExistsByURL(url, category)
				if err != nil {
					res.ErrorCount++
					res.Errors = append(res.Errors, fmt.Sprintf("check exists %s/%s: %v", category, url, err))
					continue
				}
				if exists {
					res.DuplicateCount++
					continue
				}
				if _, err := SaveTKMaterialFromNotion(category, url, pageID); err != nil {
					res.ErrorCount++
					res.Errors = append(res.Errors, fmt.Sprintf("save %s/%s: %v", category, url, err))
					continue
				}
				res.ImportedCount++
			}
		}
	}
	return res, nil
}

// extractNotionImageURLs extracts external/internal image URLs from a Notion property value.
func extractNotionImageURLs(v interface{}) []string {
	var urls []string
	if v == nil {
		return urls
	}

	propMap, ok := v.(map[string]interface{})
	if !ok {
		return urls
	}

	typ, _ := propMap["type"].(string)
	switch typ {
	case "files":
		files, _ := propMap["files"].([]interface{})
		for _, f := range files {
			fm, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			if fileType, _ := fm["type"].(string); fileType == "external" {
				if ext, ok := fm["external"].(map[string]interface{}); ok {
					if u, ok := ext["url"].(string); ok && u != "" {
						urls = append(urls, u)
					}
				}
			} else if fileType == "file" {
				if intFile, ok := fm["file"].(map[string]interface{}); ok {
					if u, ok := intFile["url"].(string); ok && u != "" {
						urls = append(urls, u)
					}
				}
			}
		}
	case "url":
		if u, ok := propMap["url"].(string); ok && u != "" {
			urls = append(urls, u)
		}
	case "rich_text":
		if rts, ok := propMap["rich_text"].([]interface{}); ok {
			for _, rt := range rts {
				rtm, ok := rt.(map[string]interface{})
				if !ok {
					continue
				}
				if txt, ok := rtm["text"].(map[string]interface{}); ok {
					if content, ok := txt["content"].(string); ok && content != "" {
						if strings.HasPrefix(content, "http") {
							urls = append(urls, content)
						}
					}
				}
			}
		}
	}
	return urls
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
