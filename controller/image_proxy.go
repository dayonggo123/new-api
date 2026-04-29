package controller

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ImageProxy handles GET /image-proxy/:id
// Lazy-loads the image from upstream if not cached locally.
func ImageProxy(c *gin.Context) {
	id := c.Param("id")
	// Strip extension if present (e.g., xxx.png -> xxx)
	id = strings.TrimSuffix(id, filepath.Ext(id))
	if id == "" {
		c.String(http.StatusNotFound, "not found")
		return
	}

	cachePath, err := service.FetchAndCacheProxyImage(id)
	if err != nil {
		c.String(http.StatusNotFound, "image not found: "+err.Error())
		return
	}

	c.File(cachePath)
}
