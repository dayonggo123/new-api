package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Discount types
const (
	DiscountTypePercentage = "percentage"
	DiscountTypeFixed      = "fixed"
)

// SubscriptionDiscountCode represents a discount code for subscription plans.
type SubscriptionDiscountCode struct {
	Id int `json:"id"`

	Code           string  `json:"code" gorm:"type:varchar(64);uniqueIndex;not null"`
	DiscountType   string  `json:"discount_type" gorm:"type:varchar(16);not null;default:'percentage'"` // percentage / fixed
	DiscountValue  float64 `json:"discount_value" gorm:"type:decimal(10,6);not null;default:0"`
	Currency       string  `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`
	MaxUses        int     `json:"max_uses" gorm:"type:int;default:0"`          // 0 = unlimited
	UsedCount      int     `json:"used_count" gorm:"type:int;default:0"`        // auto-increment on successful order
	ValidFrom      int64   `json:"valid_from" gorm:"bigint;default:0"`          // 0 = no start limit
	ValidUntil     int64   `json:"valid_until" gorm:"bigint;default:0"`         // 0 = no end limit
	ApplicablePlans string `json:"applicable_plans" gorm:"type:varchar(512);default:''"` // comma-separated plan IDs, empty = all
	MinOrderAmount float64 `json:"min_order_amount" gorm:"type:decimal(10,6);default:0"`
	Enabled        bool    `json:"enabled" gorm:"default:true"`
	Description    string  `json:"description" gorm:"type:varchar(255);default:''"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (d *SubscriptionDiscountCode) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	d.CreatedAt = now
	d.UpdatedAt = now
	return nil
}

func (d *SubscriptionDiscountCode) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = common.GetTimestamp()
	return nil
}

// ---- Validation ----

type DiscountValidationResult struct {
	Valid            bool    `json:"valid"`
	Code             string  `json:"code"`
	DiscountType     string  `json:"discount_type"`
	DiscountValue    float64 `json:"discount_value"`
	OriginalAmount   float64 `json:"original_amount"`
	DiscountedAmount float64 `json:"discounted_amount"`
	DiscountAmount   float64 `json:"discount_amount"`
	Message          string  `json:"message,omitempty"`
}

// ValidateDiscountCode checks if a discount code is valid for the given plan and amount.
func ValidateDiscountCode(code string, planId int, originalAmount float64) (*DiscountValidationResult, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	result := &DiscountValidationResult{
		Valid:          false,
		Code:           code,
		OriginalAmount: originalAmount,
	}

	if code == "" {
		result.Message = "折扣码不能为空"
		return result, nil
	}

	var dc SubscriptionDiscountCode
	if err := DB.Where("code = ?", code).First(&dc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Message = "折扣码不存在"
			return result, nil
		}
		return nil, err
	}

	if !dc.Enabled {
		result.Message = "折扣码已禁用"
		return result, nil
	}

	now := common.GetTimestamp()
	if dc.ValidFrom > 0 && now < dc.ValidFrom {
		result.Message = "折扣码尚未生效"
		return result, nil
	}
	if dc.ValidUntil > 0 && now > dc.ValidUntil {
		result.Message = "折扣码已过期"
		return result, nil
	}

	if dc.MaxUses > 0 && dc.UsedCount >= dc.MaxUses {
		result.Message = "折扣码已达使用上限"
		return result, nil
	}

	if dc.MinOrderAmount > 0 && originalAmount < dc.MinOrderAmount {
		result.Message = fmt.Sprintf("订单金额需达到 %.2f 才能使用该折扣码", dc.MinOrderAmount)
		return result, nil
	}

	// Check applicable plans
	if strings.TrimSpace(dc.ApplicablePlans) != "" {
		planIds := strings.Split(dc.ApplicablePlans, ",")
		found := false
		for _, pidStr := range planIds {
			pidStr = strings.TrimSpace(pidStr)
			if pidStr == "" {
				continue
			}
			var pid int
			if _, err := fmt.Sscanf(pidStr, "%d", &pid); err != nil {
				continue
			}
			if pid == planId {
				found = true
				break
			}
		}
		if !found {
			result.Message = "该折扣码不适用于当前套餐"
			return result, nil
		}
	}

	// Calculate discounted amount
	discountedAmount := originalAmount
	discountAmount := 0.0

	switch dc.DiscountType {
	case DiscountTypePercentage:
		if dc.DiscountValue <= 0 || dc.DiscountValue > 100 {
			result.Message = "折扣百分比无效"
			return result, nil
		}
		discountAmount = originalAmount * dc.DiscountValue / 100.0
		discountedAmount = originalAmount - discountAmount
	case DiscountTypeFixed:
		if dc.DiscountValue <= 0 {
			result.Message = "固定折扣金额无效"
			return result, nil
		}
		discountAmount = dc.DiscountValue
		discountedAmount = originalAmount - discountAmount
	default:
		result.Message = "未知的折扣类型"
		return result, nil
	}

	if discountedAmount < 0 {
		discountedAmount = 0
		discountAmount = originalAmount
	}

	result.Valid = true
	result.DiscountType = dc.DiscountType
	result.DiscountValue = dc.DiscountValue
	result.DiscountedAmount = math.Round(discountedAmount*100) / 100
	result.DiscountAmount = math.Round(discountAmount*100) / 100
	return result, nil
}

// IncrementDiscountUsage atomically increments the used_count of a discount code.
func IncrementDiscountUsage(code string) error {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil
	}
	return DB.Model(&SubscriptionDiscountCode{}).
		Where("code = ?", code).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

// GetSubscriptionDiscountCodeByCode fetches a discount code by its code string.
func GetSubscriptionDiscountCodeByCode(code string) (*SubscriptionDiscountCode, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, errors.New("code is empty")
	}
	var dc SubscriptionDiscountCode
	if err := DB.Where("code = ?", code).First(&dc).Error; err != nil {
		return nil, err
	}
	return &dc, nil
}

// ---- Admin helpers ----

func ListSubscriptionDiscountCodes() ([]SubscriptionDiscountCode, error) {
	var codes []SubscriptionDiscountCode
	if err := DB.Order("created_at desc").Find(&codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

func GetSubscriptionDiscountCodeById(id int) (*SubscriptionDiscountCode, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	var dc SubscriptionDiscountCode
	if err := DB.Where("id = ?", id).First(&dc).Error; err != nil {
		return nil, err
	}
	return &dc, nil
}

func DeleteSubscriptionDiscountCode(id int) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	return DB.Where("id = ?", id).Delete(&SubscriptionDiscountCode{}).Error
}
