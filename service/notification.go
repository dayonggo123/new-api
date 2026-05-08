package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// SendNotificationRequest 发送通知请求
type SendNotificationRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	Type        string `json:"type" binding:"required"` // system / promotion / announcement / task_status
	TargetType  string `json:"target_type"`             // all / users / group
	TargetUsers []int  `json:"target_users"`            // target_type=users 时有效
	TargetGroup string `json:"target_group"`            // target_type=group 时有效
	ActionUrl   string `json:"action_url"`
}

// SendNotification 发送通知
func SendNotification(req *SendNotificationRequest) error {
	if req.Title == "" || req.Content == "" {
		return fmt.Errorf("标题和内容不能为空")
	}
	if req.Type == "" {
		req.Type = "system"
	}
	if req.TargetType == "" {
		req.TargetType = "all"
	}

	now := time.Now().Unix()

	switch req.TargetType {
	case "all":
		// 广播消息：user_id = 0
		notification := &model.Notification{
			UserId:      0,
			Title:       req.Title,
			Content:     req.Content,
			Type:        req.Type,
			IsRead:      false,
			ActionUrl:   req.ActionUrl,
			CreatedTime: now,
		}
		return model.CreateNotification(notification)

	case "users":
		if len(req.TargetUsers) == 0 {
			return fmt.Errorf("目标用户列表不能为空")
		}
		notifications := make([]*model.Notification, len(req.TargetUsers))
		for i, userId := range req.TargetUsers {
			notifications[i] = &model.Notification{
				UserId:      userId,
				Title:       req.Title,
				Content:     req.Content,
				Type:        req.Type,
				IsRead:      false,
				ActionUrl:   req.ActionUrl,
				CreatedTime: now,
			}
		}
		return model.CreateNotificationsBatch(notifications)

	case "group":
		if req.TargetGroup == "" {
			return fmt.Errorf("目标用户组不能为空")
		}
		// 查询该组所有用户
		users, _, err := model.SearchUsers("", req.TargetGroup, 0, 10000)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			return fmt.Errorf("该用户组没有用户")
		}
		notifications := make([]*model.Notification, len(users))
		for i, user := range users {
			notifications[i] = &model.Notification{
				UserId:      user.Id,
				Title:       req.Title,
				Content:     req.Content,
				Type:        req.Type,
				IsRead:      false,
				ActionUrl:   req.ActionUrl,
				CreatedTime: now,
			}
		}
		return model.CreateNotificationsBatch(notifications)

	default:
		return fmt.Errorf("不支持的目标类型: %s", req.TargetType)
	}
}
