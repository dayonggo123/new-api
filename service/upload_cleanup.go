package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const uploadCleanupDir = "./uploads"

// StartUploadCleanupTask starts a background goroutine that periodically
// removes uploaded files (images/videos/proxy-cache) older than retentionDays.
func StartUploadCleanupTask(retentionDays int) {
	if retentionDays <= 0 {
		return
	}
	go func() {
		// Run once at startup after a short delay
		time.Sleep(5 * time.Minute)
		for {
			cleanUploads(retentionDays)
			// Run once per day
			time.Sleep(24 * time.Hour)
		}
	}()
}

func cleanUploads(retentionDays int) {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	removed := 0
	var totalSize int64

	err := filepath.Walk(uploadCleanupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			return nil
		}
		// Skip the proxy index file
		if info.Name() == "_index.json" {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if rmErr := os.Remove(path); rmErr == nil {
				removed++
				totalSize += size
			}
		}
		return nil
	})

	if err != nil {
		common.SysLog(fmt.Sprintf("upload cleanup walk error: %v", err))
	}
	if removed > 0 {
		common.SysLog(fmt.Sprintf("upload cleanup: removed %d files, freed %d MB", removed, totalSize/1024/1024))
	}
}
