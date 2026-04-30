package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func AsyncImageTaskFetch(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	task := service.GetAsyncImageTask(taskID)
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found or expired"})
		return
	}

	body, statusCode, err := service.PollAsyncImageTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Try to parse JSON and rewrite image URLs with proxy
	var result map[string]any
	if err := common.Unmarshal(body, &result); err == nil {
		rewriteImageURLsInResponse(result, c)
		c.JSON(statusCode, result)
		return
	}

	c.Data(statusCode, "application/json", body)
}

func rewriteImageURLsInResponse(v any, c *gin.Context) {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			if str, ok := item.(string); ok && (k == "url" || k == "image_url") && str != "" {
				val[k] = buildProxyURL(str, c)
			} else {
				rewriteImageURLsInResponse(item, c)
			}
		}
	case []any:
		for i, item := range val {
			rewriteImageURLsInResponse(item, c)
			val[i] = item
		}
	}
}

func buildProxyURL(upstreamURL string, c *gin.Context) string {
	proxyID := service.RegisterImageProxyURL(upstreamURL)
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s/image-proxy/%s.png", scheme, host, proxyID)
}
