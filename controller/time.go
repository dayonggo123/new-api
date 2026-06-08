package controller

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// GetServerTime returns the current server time in multiple formats.
func GetServerTime(c *gin.Context) {
	now := time.Now()
	common.ApiSuccess(c, gin.H{
		"unix":       now.Unix(),
		"unix_ms":    now.UnixMilli(),
		"iso":        now.UTC().Format(time.RFC3339),
		"timezone":   now.Location().String(),
		"local_time": now.Format("2006-01-02 15:04:05"),
	})
}
