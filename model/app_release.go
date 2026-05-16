package model

import (
	"errors"
	"fmt"
	"time"
)

// AppRelease 安装包版本管理
type AppRelease struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	Version      string `json:"version" gorm:"not null;size:32"`
	Tag          string `json:"tag" gorm:"not null;size:32"`
	Platform     string `json:"platform" gorm:"not null;size:20"`
	Arch         string `json:"arch" gorm:"not null;size:20"`
	FileName     string `json:"file_name" gorm:"not null;size:255"`
	FilePath     string `json:"-" gorm:"not null;size:500"`
	FileSize     int64  `json:"file_size" gorm:"not null"`
	DownloadUrl  string `json:"download_url" gorm:"not null;size:500"`
	ReleaseNotes string `json:"release_notes" gorm:"type:text"`
	IsLatest     bool   `json:"is_latest" gorm:"default:false"`
	IsForce      bool   `json:"is_force" gorm:"default:false"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
}

func (AppRelease) TableName() string {
	return "app_releases"
}

func (release *AppRelease) Insert() error {
	release.CreatedAt = time.Now().Unix()
	return DB.Create(release).Error
}

func (release *AppRelease) Delete() error {
	return DB.Delete(release).Error
}

func DeleteAppReleaseById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	release := AppRelease{Id: id}
	err := DB.Where(release).First(&release).Error
	if err != nil {
		return err
	}
	return release.Delete()
}

func GetAppReleaseById(id int) (*AppRelease, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	var release AppRelease
	err := DB.First(&release, "id = ?", id).Error
	return &release, err
}

func GetLatestAppReleases() ([]*AppRelease, error) {
	var releases []*AppRelease
	err := DB.Where("is_latest = ?", true).Order("platform asc, arch asc").Find(&releases).Error
	return releases, err
}

func GetLatestAppReleaseByPlatformArch(platform, arch string) (*AppRelease, error) {
	var release AppRelease
	err := DB.Where("platform = ? AND arch = ? AND is_latest = ?", platform, arch, true).First(&release).Error
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func GetAllAppReleases(startIdx int, num int) (releases []*AppRelease, total int64, err error) {
	err = DB.Model(&AppRelease{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if num > 0 {
		err = DB.Order("created_at desc").Limit(num).Offset(startIdx).Find(&releases).Error
	} else {
		err = DB.Order("created_at desc").Find(&releases).Error
	}
	return releases, total, err
}

// MarkAsLatest 标记为最新版本
// 将同平台同架构的旧版本 is_latest 设为 0
func MarkAppReleaseAsLatest(id int) error {
	release, err := GetAppReleaseById(id)
	if err != nil {
		return err
	}

	// 开启事务
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 将同平台同架构的旧版本 is_latest 设为 0
	if err := tx.Model(&AppRelease{}).Where("platform = ? AND arch = ? AND id != ?", release.Platform, release.Arch, id).Update("is_latest", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 将当前版本 is_latest 设为 1
	if err := tx.Model(&AppRelease{}).Where("id = ?", id).Update("is_latest", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetAppReleaseDownloadPath 返回安装包存储目录
func GetAppReleaseDownloadPath() string {
	return "./downloads"
}

// BuildAppReleaseDownloadURL 构建下载 URL
func BuildAppReleaseDownloadURL(baseURL, platform, arch string) string {
	return fmt.Sprintf("%s/api/public/releases/download/%s/%s", baseURL, platform, arch)
}
