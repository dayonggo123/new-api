package router

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetWebRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())

	// Custom static file server: skip relay API paths
	efs, _ := fs.Sub(buildFS, "web/dist")

	router.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/uapi") || strings.HasPrefix(path, "/mj") || strings.HasPrefix(path, "/v1") || strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/assets") {
			c.Next()
			return
		}
		// Try to serve static file
		f, err := efs.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			http.FileServer(http.FS(efs)).ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}
		// Fallback to index.html (SPA)
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
		c.Abort()
	})

	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/uapi") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
	})
}
