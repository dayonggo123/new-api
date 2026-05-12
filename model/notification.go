package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
)

// NotificationI18n 消息多语言翻译项
type NotificationI18n struct {
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

// Notification 消息通知表
type Notification struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	UserId      int    `json:"user_id" gorm:"index"`                 // 0 表示全员广播
	Title       string `json:"title" gorm:"type:varchar(200);not null"`
	Content     string `json:"content" gorm:"type:text;not null"`
	I18n        string `json:"i18n,omitempty" gorm:"type:text"`     // JSON: {"en": {"title": "..."}, "fr": {...}}
	Type        string `json:"type" gorm:"type:varchar(32)"`        // system / promotion / announcement / task_status
	IsRead      bool   `json:"is_read" gorm:"default:false"`
	ActionUrl   string `json:"action_url" gorm:"type:varchar(500)"`
	CreatedTime int64  `json:"created_time" gorm:"bigint;index"`
}

// ApplyLanguage 根据语言代码替换标题和内容（缺失则保持默认中文）
func (n *Notification) ApplyLanguage(lang string) {
	if lang == "" || lang == "zh" || lang == "zh-CN" || lang == "zh-TW" {
		return
	}
	if n.I18n == "" {
		return
	}
	var i18nMap map[string]NotificationI18n
	if err := common.Unmarshal([]byte(n.I18n), &i18nMap); err != nil {
		return
	}
	if t, ok := i18nMap[lang]; ok {
		if t.Title != "" {
			n.Title = t.Title
		}
		if t.Content != "" {
			n.Content = t.Content
		}
	}
}

func (n *Notification) TableName() string {
	return "notifications"
}

// GetNotificationsByUserId 分页获取用户消息（包含广播消息）
func GetNotificationsByUserId(userId int, startIdx int, pageSize int) ([]*Notification, int64, error) {
	var notifications []*Notification
	var total int64
	err := DB.Model(&Notification{}).Where("user_id = ? OR user_id = 0", userId).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DB.Where("user_id = ? OR user_id = 0", userId).
		Order("created_time DESC").
		Limit(pageSize).Offset(startIdx).
		Find(&notifications).Error
	return notifications, total, err
}

// GetUnreadNotificationCount 获取用户未读消息数
func GetUnreadNotificationCount(userId int) (int64, error) {
	var count int64
	err := DB.Model(&Notification{}).
		Where("(user_id = ? OR user_id = 0) AND is_read = ?", userId, commonFalseVal).
		Count(&count).Error
	return count, err
}

// MarkNotificationAsRead 标记单条消息已读
func MarkNotificationAsRead(userId int, notificationId int) error {
	return DB.Model(&Notification{}).
		Where("id = ? AND (user_id = ? OR user_id = 0)", notificationId, userId).
		Update("is_read", commonTrueVal).Error
}

// MarkAllNotificationsAsRead 标记用户所有消息已读
func MarkAllNotificationsAsRead(userId int) (int64, error) {
	result := DB.Model(&Notification{}).
		Where("(user_id = ? OR user_id = 0) AND is_read = ?", userId, commonFalseVal).
		Update("is_read", commonTrueVal)
	return result.RowsAffected, result.Error
}

// CreateNotification 创建通知（管理员发布）
func CreateNotification(notification *Notification) error {
	if notification.CreatedTime == 0 {
		notification.CreatedTime = time.Now().Unix()
	}
	return DB.Create(notification).Error
}

// CreateNotificationsBatch 批量创建通知（广播给用户列表）
func CreateNotificationsBatch(notifications []*Notification) error {
	now := time.Now().Unix()
	for _, n := range notifications {
		if n.CreatedTime == 0 {
			n.CreatedTime = now
		}
	}
	return DB.CreateInBatches(notifications, 100).Error
}

// GetAllNotifications 获取所有通知（管理员后台）
func GetAllNotifications(startIdx int, pageSize int, notificationType string) ([]*Notification, int64, error) {
	var notifications []*Notification
	var total int64
	db := DB.Model(&Notification{})
	if notificationType != "" {
		db = db.Where("type = ?", notificationType)
	}
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Order("created_time DESC").Limit(pageSize).Offset(startIdx).Find(&notifications).Error
	return notifications, total, err
}

// UpdateNotification 更新通知内容
func UpdateNotification(id int, updates map[string]interface{}) error {
	return DB.Model(&Notification{}).Where("id = ?", id).Updates(updates).Error
}
