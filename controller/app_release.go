package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

const appReleaseDownloadDir = "./downloads"

// ==================== Public API ====================

// GetLatestAppRelease 获取最新版本信息（兼容 GitHub Releases API 格式）
func GetLatestAppRelease(c *gin.Context) {
	releases, err := model.GetLatestAppReleases()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(releases) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	assets := make([]gin.H, 0, len(releases))
	for _, r := range releases {
		assets = append(assets, gin.H{
			"id":                   r.Id,
			"name":                 r.FileName,
			"size":                 r.FileSize,
			"browser_download_url": r.DownloadUrl,
		})
	}

	latest := releases[0]
	c.JSON(http.StatusOK, gin.H{
		"tag_name":     latest.Tag,
		"name":         latest.Tag,
		"body":         latest.ReleaseNotes,
		"published_at": time.Unix(latest.CreatedAt, 0).Format(time.RFC3339),
		"assets":       assets,
	})
}

// GetLatestReleaseJSON 返回 Tauri updater 格式的 latest.json
func GetLatestReleaseJSON(c *gin.Context) {
	releases, err := model.GetLatestAppReleases()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(releases) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	latest := releases[0]

	// Tauri updater platform key: windows-x86_64, darwin-x86_64, darwin-aarch64, linux-x86_64
	platforms := make(map[string]gin.H)
	for _, r := range releases {
		platformKey := fmt.Sprintf("%s-%s", r.Platform, r.Arch)
		platforms[platformKey] = gin.H{
			"signature": r.Signature,
			"url":       r.DownloadUrl,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"version":   latest.Version,
		"notes":     latest.ReleaseNotes,
		"pub_date":  time.Unix(latest.CreatedAt, 0).Format(time.RFC3339),
		"platforms": platforms,
	})
}

// DownloadAppRelease 下载安装包
func DownloadAppRelease(c *gin.Context) {
	platform := c.Param("platform")
	arch := c.Param("arch")

	release, err := model.GetLatestAppReleaseByPlatformArch(platform, arch)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(release.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, release.FileName))
	c.Header("Content-Type", "application/octet-stream")
	c.File(release.FilePath)
}

// ==================== Admin API ====================

// GetAllAppReleases 获取版本列表
func GetAllAppReleases(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	releases, total, err := model.GetAllAppReleases(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(releases)
	common.ApiSuccess(c, pageInfo)
}

// UploadAppRelease 上传新版本
func UploadAppRelease(c *gin.Context) {
	version := c.PostForm("version")
	tag := c.PostForm("tag")
	platform := c.PostForm("platform")
	arch := c.PostForm("arch")
	releaseNotes := c.PostForm("release_notes")
	isForce := c.PostForm("is_force") == "true"

	if version == "" || tag == "" || platform == "" || arch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version, tag, platform, arch are required"})
		return
	}

	// 校验 platform 和 arch
	validPlatforms := map[string]bool{"windows": true, "darwin": true, "linux": true}
	validArchs := map[string]bool{"x86_64": true, "aarch64": true}
	if !validPlatforms[platform] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid platform, must be windows|darwin|linux"})
		return
	}
	if !validArchs[arch] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid arch, must be x86_64|aarch64"})
		return
	}

	// 获取上传文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// 获取签名文件（可选）
	var sigContent string
	if sigFile, err := c.FormFile("signature_file"); err == nil {
		sigPath := filepath.Join(appReleaseDownloadDir, sigFile.Filename)
		if err := c.SaveUploadedFile(sigFile, sigPath); err == nil {
			data, _ := os.ReadFile(sigPath)
			sigContent = string(data)
			_ = os.Remove(sigPath) // 读取后删除临时文件
		}
	} else if sigText := c.PostForm("signature"); sigText != "" {
		sigContent = sigText
	}

	// 创建下载目录
	if err := os.MkdirAll(appReleaseDownloadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create download directory: %v", err)})
		return
	}

	// 生成文件名: harsetv_1.2.3_windows_x86_64.exe
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("harsetv_%s_%s_%s%s", version, platform, arch, ext)
	filePath := filepath.Join(appReleaseDownloadDir, fileName)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file: %v", err)})
		return
	}

	// 获取文件大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get file info: %v", err)})
		return
	}

	// 构建下载 URL
	serverAddr := strings.TrimSuffix(system_setting.ServerAddress, "/")
	if serverAddr == "" {
		serverAddr = "https://heharse.cloud"
	}
	downloadURL := model.BuildAppReleaseDownloadURL(serverAddr, platform, arch)

	// 写入数据库
	release := &model.AppRelease{
		Version:      version,
		Tag:          tag,
		Platform:     platform,
		Arch:         arch,
		FileName:     fileName,
		FilePath:     filePath,
		FileSize:     fileInfo.Size(),
		DownloadUrl:  downloadURL,
		ReleaseNotes: releaseNotes,
		Signature:    sigContent,
		IsForce:      isForce,
	}

	if err := release.Insert(); err != nil {
		// 清理已保存的文件
		_ = os.Remove(filePath)
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "上传成功",
		"data":    release,
	})
}

// DeleteAppRelease 删除版本
func DeleteAppRelease(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	release, err := model.GetAppReleaseById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 删除文件
	if release.FilePath != "" {
		_ = os.Remove(release.FilePath)
	}

	// 删除数据库记录
	if err := model.DeleteAppReleaseById(id); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"message": "删除成功"})
}

// MarkAppReleaseAsLatest 标记为最新版
func MarkAppReleaseAsLatest(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	if err := model.MarkAppReleaseAsLatest(id); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"message": "已标记为最新版"})
}
