package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

// SigninResult 签到结果
type SigninResult struct {
	PointsEarned    int `json:"points_earned"`
	BonusPoints     int `json:"bonus_points"`
	TotalPoints     int `json:"total_points"`
	ConsecutiveDays int `json:"consecutive_days"`
}

// UserSignin 用户签到
func UserSignin(userId int) (*SigninResult, error) {
	if !operation_setting.IsSigninPointsEnabled() {
		return nil, fmt.Errorf("积分签到功能已关闭")
	}

	// 获取今天 00:00:00 的时间戳
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()

	// 检查今天是否已签到
	hasSignin, err := model.HasSigninToday(userId, todayStart)
	if err != nil {
		return nil, err
	}
	if hasSignin {
		return nil, fmt.Errorf("今日已签到")
	}

	// 获取或创建用户积分记录
	up, err := model.GetOrCreateUserPoints(userId)
	if err != nil {
		return nil, err
	}

	// 计算连续签到天数
	consecutiveDays := 1
	if up.LastSigninDate > 0 {
		yesterdayStart := todayStart - 86400
		if up.LastSigninDate >= yesterdayStart && up.LastSigninDate < todayStart {
			// 昨天签到了，连续天数+1
			consecutiveDays = up.ConsecutiveDays + 1
		}
	}

	// 计算积分
	basePoints := operation_setting.GetSigninBasePoints()
	bonusPoints := operation_setting.GetConsecutiveBonus(consecutiveDays)
	totalEarned := basePoints + bonusPoints
	newTotalPoints := up.TotalPoints + totalEarned

	// 开启事务
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 插入签到历史
	history := &model.UserSigninHistory{
		UserId:           userId,
		SigninDate:       todayStart,
		Points:           basePoints,
		BonusPoints:      bonusPoints,
		TotalPointsAfter: newTotalPoints,
		CreatedTime:      time.Now().Unix(),
	}
	if err := tx.Create(history).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 2. 插入基础积分流水
	if err := model.CreatePointsTransaction(tx, userId, basePoints, newTotalPoints-bonusPoints,
		"signin", "", fmt.Sprintf("第%d天签到", consecutiveDays)); err != nil {
		tx.Rollback()
		return nil, err
	}

	// 3. 如果有奖励积分，插入奖励流水
	if bonusPoints > 0 {
		if err := model.CreatePointsTransaction(tx, userId, bonusPoints, newTotalPoints,
			"signin_bonus", "", fmt.Sprintf("连续签到%d天奖励", consecutiveDays)); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 4. 更新用户积分总表
	if err := model.UpdateUserPoints(tx, userId, newTotalPoints, consecutiveDays, todayStart); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &SigninResult{
		PointsEarned:    totalEarned,
		BonusPoints:     bonusPoints,
		TotalPoints:     newTotalPoints,
		ConsecutiveDays: consecutiveDays,
	}, nil
}

// UnlockPromptResult 解锁提示词结果
type UnlockPromptResult struct {
	PromptId        int `json:"prompt_id"`
	Cost            int `json:"cost"`
	RemainingPoints int `json:"remaining_points"`
}

// UnlockPrompt 解锁提示词
func UnlockPrompt(userId int, promptId int) (*UnlockPromptResult, error) {
	// 1. 检查提示词是否存在且为付费
	prompt, err := model.GetPromptById(promptId)
	if err != nil {
		return nil, fmt.Errorf("提示词不存在")
	}
	if prompt.Status != 1 {
		return nil, fmt.Errorf("提示词已下架")
	}
	if !prompt.IsPremium {
		return nil, fmt.Errorf("该提示词免费，无需解锁")
	}

	// 2. 检查是否已解锁
	unlocked, err := model.IsPromptUnlocked(userId, promptId)
	if err != nil {
		return nil, err
	}
	if unlocked {
		return nil, fmt.Errorf("已解锁该提示词")
	}

	// 3. 检查积分是否足够
	up, err := model.GetUserPointsById(userId)
	if err != nil {
		// 如果用户没有积分记录，则积分为0
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("积分不足，需要 %d 积分", prompt.UnlockCost)
		}
		return nil, err
	}
	if up.TotalPoints < prompt.UnlockCost {
		return nil, fmt.Errorf("积分不足，需要 %d 积分", prompt.UnlockCost)
	}

	// 4. 扣减积分并解锁（事务）
	newTotal := up.TotalPoints - prompt.UnlockCost
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 插入解锁记录
	if err := model.UnlockPrompt(tx, userId, promptId, prompt.UnlockCost); err != nil {
		tx.Rollback()
		return nil, err
	}

	// 插入积分流水
	if err := model.CreatePointsTransaction(tx, userId, -prompt.UnlockCost, newTotal,
		"unlock_prompt", fmt.Sprintf("%d", promptId), fmt.Sprintf("解锁提示词: %s", prompt.Title)); err != nil {
		tx.Rollback()
		return nil, err
	}

	// 更新积分总表
	if err := model.UpdateUserPoints(tx, userId, newTotal, up.ConsecutiveDays, up.LastSigninDate); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &UnlockPromptResult{
		PromptId:        promptId,
		Cost:            prompt.UnlockCost,
		RemainingPoints: newTotal,
	}, nil
}

// AdjustUserPoints 管理员调整用户积分
func AdjustUserPoints(userId int, amount int, description string) (int, error) {
	up, err := model.GetOrCreateUserPoints(userId)
	if err != nil {
		return 0, err
	}

	newTotal := up.TotalPoints + amount
	if newTotal < 0 {
		return 0, fmt.Errorf("调整后积分不能为负数")
	}

	tx := model.DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := model.CreatePointsTransaction(tx, userId, amount, newTotal,
		"admin_adjust", "", description); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := model.UpdateUserPoints(tx, userId, newTotal, up.ConsecutiveDays, up.LastSigninDate); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return newTotal, nil
}
