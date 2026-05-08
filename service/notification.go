package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// SendNotificationRequest 发送通知请求
type SendNotificationRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	Type        string `json:"type" binding:"required"` // system / promotion / announcement / task_status
	TargetType  string `json:"target_type"`             // all / users / group / tier / tag
	TargetUsers []int  `json:"target_users"`            // target_type=users 时有效
	TargetGroup string `json:"target_group"`            // target_type=group 时有效
	TargetTiers []int  `json:"target_tiers"`            // target_type=tier 时有效
	TargetTags  []int  `json:"target_tags"`             // target_type=tag 时有效
	ActionUrl   string `json:"action_url"`
	UseTemplate bool   `json:"use_template"`            // 是否启用模板变量替换
}

// UserContext 用户上下文，用于模板变量替换
type UserContext struct {
	UserId         int
	Username       string
	DisplayName    string
	TierName       string
	TierLevel      int
	TotalPoints    int
	ConsecutiveDays int
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

	// 1. 解析目标用户列表
	userIds, err := resolveTargetUsers(req)
	if err != nil {
		return err
	}
	if len(userIds) == 0 {
		return fmt.Errorf("没有符合条件的用户")
	}

	// 2. 广播消息直接插入一条 user_id=0 的记录
	if req.TargetType == "all" {
		notification := &model.Notification{
			UserId:      0,
			Title:       req.Title,
			Content:     req.Content,
			Type:        req.Type,
			IsRead:      false,
			ActionUrl:   req.ActionUrl,
			CreatedTime: time.Now().Unix(),
		}
		return model.CreateNotification(notification)
	}

	// 3. 个性化消息：逐个渲染模板并批量插入
	now := time.Now().Unix()
	useTemplate := req.UseTemplate && (strings.Contains(req.Title, "{{") || strings.Contains(req.Content, "{{"))

	// 批量查询用户信息（避免 N+1）
	userContexts, err := buildUserContexts(userIds)
	if err != nil {
		return err
	}

	notifications := make([]*model.Notification, 0, len(userIds))
	for _, uc := range userContexts {
		title := req.Title
		content := req.Content
		if useTemplate {
			title = renderTemplate(title, uc)
			content = renderTemplate(content, uc)
		}
		notifications = append(notifications, &model.Notification{
			UserId:      uc.UserId,
			Title:       title,
			Content:     content,
			Type:        req.Type,
			IsRead:      false,
			ActionUrl:   req.ActionUrl,
			CreatedTime: now,
		})
	}

	return model.CreateNotificationsBatch(notifications)
}

// resolveTargetUsers 根据 target_type 解析目标用户 ID 列表
func resolveTargetUsers(req *SendNotificationRequest) ([]int, error) {
	switch req.TargetType {
	case "all":
		// 广播，返回空列表，由上层处理
		return []int{}, nil

	case "users":
		if len(req.TargetUsers) == 0 {
			return nil, fmt.Errorf("目标用户列表不能为空")
		}
		return req.TargetUsers, nil

	case "group":
		if req.TargetGroup == "" {
			return nil, fmt.Errorf("目标用户组不能为空")
		}
		users, _, err := model.SearchUsers("", req.TargetGroup, 0, 10000)
		if err != nil {
			return nil, err
		}
		if len(users) == 0 {
			return nil, fmt.Errorf("该用户组没有用户")
		}
		userIds := make([]int, len(users))
		for i, u := range users {
			userIds[i] = u.Id
		}
		return userIds, nil

	case "tier":
		if len(req.TargetTiers) == 0 {
			return nil, fmt.Errorf("目标层级不能为空")
		}
		return model.GetUsersByTierLevels(req.TargetTiers)

	case "tag":
		if len(req.TargetTags) == 0 {
			return nil, fmt.Errorf("目标标签不能为空")
		}
		return model.GetUsersByTagIds(req.TargetTags)

	default:
		return nil, fmt.Errorf("不支持的目标类型: %s", req.TargetType)
	}
}

// buildUserContexts 批量构建用户上下文
func buildUserContexts(userIds []int) ([]*UserContext, error) {
	if len(userIds) == 0 {
		return []*UserContext{}, nil
	}

	// 查询用户信息
	var users []model.User
	err := model.DB.Where("id IN ?", userIds).Find(&users).Error
	if err != nil {
		return nil, err
	}

	// 查询用户积分和层级
	var userPoints []model.UserPoints
	err = model.DB.Where("user_id IN ?", userIds).Find(&userPoints).Error
	if err != nil {
		return nil, err
	}
	upMap := make(map[int]*model.UserPoints, len(userPoints))
	for i := range userPoints {
		upMap[userPoints[i].UserId] = &userPoints[i]
	}

	// 查询所有层级定义（用于 tier_name）
	tiers, err := model.GetAllTiers()
	if err != nil {
		return nil, err
	}
	tierMap := make(map[int]string, len(tiers))
	for _, t := range tiers {
		tierMap[t.Level] = t.Name
	}

	contexts := make([]*UserContext, 0, len(users))
	for _, u := range users {
		uc := &UserContext{
			UserId:      u.Id,
			Username:    u.Username,
			DisplayName: u.DisplayName,
		}
		if up, ok := upMap[u.Id]; ok {
			uc.TierLevel = up.TierLevel
			if uc.TierLevel == 0 {
				uc.TierLevel = 1
			}
			uc.TotalPoints = up.TotalPoints
			uc.ConsecutiveDays = up.ConsecutiveDays
			uc.TierName = tierMap[uc.TierLevel]
			if uc.TierName == "" {
				uc.TierName = fmt.Sprintf("L%d", uc.TierLevel)
			}
		}
		if uc.DisplayName == "" {
			uc.DisplayName = uc.Username
		}
		contexts = append(contexts, uc)
	}

	return contexts, nil
}

// renderTemplate 渲染模板变量
func renderTemplate(template string, uc *UserContext) string {
	result := template
	result = strings.ReplaceAll(result, "{{username}}", uc.Username)
	result = strings.ReplaceAll(result, "{{display_name}}", uc.DisplayName)
	result = strings.ReplaceAll(result, "{{tier_name}}", uc.TierName)
	result = strings.ReplaceAll(result, "{{tier_level}}", fmt.Sprintf("%d", uc.TierLevel))
	result = strings.ReplaceAll(result, "{{total_points}}", fmt.Sprintf("%d", uc.TotalPoints))
	result = strings.ReplaceAll(result, "{{consecutive_days}}", fmt.Sprintf("%d", uc.ConsecutiveDays))
	return result
}
