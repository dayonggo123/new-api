package model

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

const (
	referralCodeLength = 6
	referralCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // excludes 0/O/I/1
)

// ReferralCode represents a user's social sharing referral code.
type ReferralCode struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId        int    `json:"user_id" gorm:"uniqueIndex;not null"`
	Code          string `json:"code" gorm:"type:varchar(16);uniqueIndex;not null"`
	ShortLink     string `json:"short_link" gorm:"type:varchar(255);column:short_link"`
	QrUrl         string `json:"qr_url" gorm:"type:varchar(255);column:qr_url"`
	ExpiresAt     *int64 `json:"expires_at" gorm:"column:expires_at"`
	MaxUses       int    `json:"max_uses" gorm:"column:max_uses;default:0"`
	UsedCount     int    `json:"used_count" gorm:"column:used_count;default:0"`
	CreatedTime   int64  `json:"created_time" gorm:"column:created_time;index"`
	UpdatedTime   int64  `json:"updated_time" gorm:"column:updated_time"`
}

func (ReferralCode) TableName() string {
	return "referral_codes"
}

// IsExpired reports whether the referral code has expired.
func (rc *ReferralCode) IsExpired() bool {
	if rc.ExpiresAt == nil || *rc.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > *rc.ExpiresAt
}

// IsExhausted reports whether the code has reached its max usage limit.
func (rc *ReferralCode) IsExhausted() bool {
	if rc.MaxUses <= 0 {
		return false
	}
	return rc.UsedCount >= rc.MaxUses
}

// IsValid reports whether the code can still be used for registration.
func (rc *ReferralCode) IsValid() bool {
	return !rc.IsExpired() && !rc.IsExhausted()
}

// generateReferralCode returns a random 6-character code from the alphabet.
func generateReferralCode() string {
	b := make([]byte, referralCodeLength)
	for i := range b {
		b[i] = referralCodeAlphabet[rand.Intn(len(referralCodeAlphabet))]
	}
	return string(b)
}

// GetReferralCodeByUserId fetches the referral code for a given user.
func GetReferralCodeByUserId(userId int) (*ReferralCode, error) {
	var code ReferralCode
	err := DB.Where("user_id = ?", userId).First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

// GetReferralCodeByCode fetches a referral code by its short code string.
func GetReferralCodeByCode(code string) (*ReferralCode, error) {
	var rc ReferralCode
	err := DB.Where("code = ?", code).First(&rc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rc, nil
}

// CreateOrGetReferralCode returns the user's existing referral code or creates one.
func CreateOrGetReferralCode(userId int, baseUrl string, qrUrl string) (*ReferralCode, error) {
	existing, err := GetReferralCodeByUserId(userId)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now().Unix()
	rc := generateReferralCode()
	code := &ReferralCode{
		UserId:      userId,
		Code:        rc,
		ShortLink:   fmt.Sprintf("%s/r/%s", baseUrl, rc),
		QrUrl:       qrUrl,
		CreatedTime: now,
		UpdatedTime: now,
	}

	// Retry on conflict a few times.
	for i := 0; i < 5; i++ {
		err = DB.Create(code).Error
		if err == nil {
			return code, nil
		}
		rc = generateReferralCode()
		code.Code = rc
		code.ShortLink = fmt.Sprintf("%s/r/%s", baseUrl, rc)
	}
	return nil, err
}

// IncrementReferralUsedCount increments the used_count atomically.
func IncrementReferralUsedCount(codeId int) error {
	return DB.Model(&ReferralCode{}).Where("id = ?", codeId).Update("used_count", gorm.Expr("used_count + ?", 1)).Error
}

// ReferralRelationship records the binding between inviter and invitee.
type ReferralRelationship struct {
	Id                 int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ReferralCodeId     int    `json:"referral_code_id" gorm:"column:referral_code_id;index;not null"`
	InviterId          int    `json:"inviter_id" gorm:"column:inviter_id;index;not null"`
	InviteeId          int    `json:"invitee_id" gorm:"column:invitee_id;index;not null"`
	Code               string `json:"code" gorm:"type:varchar(16);index"`
	Source             string `json:"source" gorm:"type:varchar(32);column:source"`
	ContentId          string `json:"content_id" gorm:"type:varchar(64);column:content_id"`
	Ip                 string `json:"ip" gorm:"type:varchar(64);column:ip"`
	DeviceFingerprint  string `json:"device_fingerprint" gorm:"type:varchar(128);column:device_fingerprint"`
	CreatedTime        int64  `json:"created_time" gorm:"column:created_time;index"`
}

func (ReferralRelationship) TableName() string {
	return "referral_relationships"
}

// CreateReferralRelationship inserts a new relationship record.
func CreateReferralRelationship(codeId, inviterId, inviteeId int, code, source, contentId, ip, deviceFingerprint string) error {
	rel := &ReferralRelationship{
		ReferralCodeId:    codeId,
		InviterId:         inviterId,
		InviteeId:         inviteeId,
		Code:              code,
		Source:            source,
		ContentId:         contentId,
		Ip:                ip,
		DeviceFingerprint: deviceFingerprint,
		CreatedTime:       time.Now().Unix(),
	}
	return DB.Create(rel).Error
}
