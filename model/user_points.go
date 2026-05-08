package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// UserPoints 用户积分总表
type UserPoints struct {
	UserId          int   `json:"user_id" gorm:"primaryKey"`
	TotalPoints     int   `json:"total_points" gorm:"default:0"`
	ConsecutiveDays int   `json:"consecutive_days" gorm:"default:0"`
	LastSigninDate  int64 `json:"last_signin_date" gorm:"bigint"`
	TierLevel       int   `json:"tier_level" gorm:"default:1"`
	CreatedTime     int64 `json:"created_time" gorm:"bigint"`
	UpdatedTime     int64 `json:"updated_time" gorm:"bigint"`
}

func (up *UserPoints) TableName() string {
	return "user_points"
}

// GetUserPointsById 获取用户积分
func GetUserPointsById(userId int) (*UserPoints, error) {
	if userId == 0 {
		return nil, errors.New("user id is empty")
	}
	var up UserPoints
	err := DB.First(&up, userId).Error
	if err != nil {
		return nil, err
	}
	return &up, nil
}

// GetOrCreateUserPoints 获取或创建用户积分记录
func GetOrCreateUserPoints(userId int) (*UserPoints, error) {
	up, err := GetUserPointsById(userId)
	if err == nil {
		return up, nil
	}
	now := time.Now().Unix()
	up = &UserPoints{
		UserId:      userId,
		TotalPoints: 0,
		CreatedTime: now,
		UpdatedTime: now,
	}
	err = DB.Create(up).Error
	return up, err
}

// UpdateUserPoints 更新用户积分（需外部事务控制）
func UpdateUserPoints(tx *gorm.DB, userId int, totalPoints int, consecutiveDays int, lastSigninDate int64) error {
	updates := map[string]interface{}{
		"total_points":      totalPoints,
		"consecutive_days":  consecutiveDays,
		"last_signin_date":  lastSigninDate,
		"updated_time":      time.Now().Unix(),
	}
	return tx.Model(&UserPoints{}).Where("user_id = ?", userId).Updates(updates).Error
}

// UserSigninHistory 签到历史记录
type UserSigninHistory struct {
	UserId           int   `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	SigninDate       int64 `json:"signin_date" gorm:"primaryKey;autoIncrement:false;bigint"`
	Points           int   `json:"points" gorm:"default:0"`
	BonusPoints      int   `json:"bonus_points" gorm:"default:0"`
	TotalPointsAfter int   `json:"total_points_after"`
	CreatedTime      int64 `json:"created_time" gorm:"bigint"`
}

func (ush *UserSigninHistory) TableName() string {
	return "user_signin_history"
}

// HasSigninToday 检查用户今天是否已签到
func HasSigninToday(userId int, todayStart int64) (bool, error) {
	var count int64
	err := DB.Model(&UserSigninHistory{}).
		Where("user_id = ? AND signin_date = ?", userId, todayStart).
		Count(&count).Error
	return count > 0, err
}

// GetSigninHistoryByUserId 获取用户签到历史
func GetSigninHistoryByUserId(userId int, startIdx int, pageSize int) ([]*UserSigninHistory, int64, error) {
	var histories []*UserSigninHistory
	var total int64
	err := DB.Model(&UserSigninHistory{}).Where("user_id = ?", userId).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DB.Where("user_id = ?", userId).Order("signin_date DESC").Limit(pageSize).Offset(startIdx).Find(&histories).Error
	return histories, total, err
}

// UserPointsTransaction 积分流水表
type UserPointsTransaction struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	UserId       int    `json:"user_id" gorm:"index:idx_user_created,priority:1"`
	ChangeAmount int    `json:"change_amount"`
	BalanceAfter int    `json:"balance_after"`
	Type         string `json:"type" gorm:"type:varchar(32)"`
	RelatedId    string `json:"related_id" gorm:"type:varchar(64)"`
	Description  string `json:"description" gorm:"type:varchar(255)"`
	CreatedTime  int64  `json:"created_time" gorm:"bigint;index:idx_user_created,priority:2"`
}

func (upt *UserPointsTransaction) TableName() string {
	return "user_points_transactions"
}

// CreatePointsTransaction 创建积分流水记录（需外部事务控制）
func CreatePointsTransaction(tx *gorm.DB, userId int, changeAmount int, balanceAfter int, txType string, relatedId string, description string) error {
	record := &UserPointsTransaction{
		UserId:       userId,
		ChangeAmount: changeAmount,
		BalanceAfter: balanceAfter,
		Type:         txType,
		RelatedId:    relatedId,
		Description:  description,
		CreatedTime:  time.Now().Unix(),
	}
	return tx.Create(record).Error
}

// GetPointsTransactionsByUserId 获取用户积分流水
func GetPointsTransactionsByUserId(userId int, startIdx int, pageSize int) ([]*UserPointsTransaction, int64, error) {
	var transactions []*UserPointsTransaction
	var total int64
	err := DB.Model(&UserPointsTransaction{}).Where("user_id = ?", userId).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DB.Where("user_id = ?", userId).Order("created_time DESC").Limit(pageSize).Offset(startIdx).Find(&transactions).Error
	return transactions, total, err
}
