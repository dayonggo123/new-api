package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ---- User API: Validate discount code ----

type ValidateDiscountRequest struct {
	Code   string  `json:"code"`
	PlanId int     `json:"plan_id"`
	Amount float64 `json:"amount"` // optional, if not provided, will fetch from plan
}

func ValidateSubscriptionDiscount(c *gin.Context) {
	var req ValidateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if req.PlanId <= 0 {
		common.ApiErrorMsg(c, "套餐ID无效")
		return
	}

	originalAmount := req.Amount
	if originalAmount <= 0 {
		plan, err := model.GetSubscriptionPlanById(req.PlanId)
		if err != nil {
			common.ApiErrorMsg(c, "套餐不存在")
			return
		}
		originalAmount = plan.PriceAmount
	}

	result, err := model.ValidateDiscountCode(req.Code, req.PlanId, originalAmount)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, result)
}

// ---- Admin APIs: CRUD discount codes ----

func AdminListSubscriptionDiscounts(c *gin.Context) {
	codes, err := model.ListSubscriptionDiscountCodes()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, codes)
}

type AdminCreateDiscountRequest struct {
	Code            string  `json:"code"`
	DiscountType    string  `json:"discount_type"`
	DiscountValue   float64 `json:"discount_value"`
	Currency        string  `json:"currency"`
	MaxUses         int     `json:"max_uses"`
	ValidFrom       int64   `json:"valid_from"`
	ValidUntil      int64   `json:"valid_until"`
	ApplicablePlans string  `json:"applicable_plans"`
	MinOrderAmount  float64 `json:"min_order_amount"`
	Enabled         bool    `json:"enabled"`
	Description     string  `json:"description"`
}

func AdminCreateSubscriptionDiscount(c *gin.Context) {
	var req AdminCreateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	req.Code = strings.TrimSpace(strings.ToUpper(req.Code))
	if req.Code == "" {
		common.ApiErrorMsg(c, "折扣码不能为空")
		return
	}
	if len(req.Code) < 3 {
		common.ApiErrorMsg(c, "折扣码长度至少3位")
		return
	}
	if len(req.Code) > 64 {
		common.ApiErrorMsg(c, "折扣码长度不能超过64位")
		return
	}

	if req.DiscountType == "" {
		req.DiscountType = model.DiscountTypePercentage
	}
	if req.DiscountType != model.DiscountTypePercentage && req.DiscountType != model.DiscountTypeFixed {
		common.ApiErrorMsg(c, "折扣类型无效")
		return
	}

	if req.DiscountValue <= 0 {
		common.ApiErrorMsg(c, "折扣值必须大于0")
		return
	}
	if req.DiscountType == model.DiscountTypePercentage && req.DiscountValue > 100 {
		common.ApiErrorMsg(c, "百分比折扣不能超过100%")
		return
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.MaxUses < 0 {
		req.MaxUses = 0
	}
	if req.MinOrderAmount < 0 {
		req.MinOrderAmount = 0
	}

	// Check duplicate code
	existing, _ := model.GetSubscriptionDiscountCodeByCode(req.Code)
	if existing != nil {
		common.ApiErrorMsg(c, "折扣码已存在")
		return
	}

	dc := &model.SubscriptionDiscountCode{
		Code:            req.Code,
		DiscountType:    req.DiscountType,
		DiscountValue:   req.DiscountValue,
		Currency:        req.Currency,
		MaxUses:         req.MaxUses,
		ValidFrom:       req.ValidFrom,
		ValidUntil:      req.ValidUntil,
		ApplicablePlans: req.ApplicablePlans,
		MinOrderAmount:  req.MinOrderAmount,
		Enabled:         req.Enabled,
		Description:     req.Description,
	}
	if err := model.DB.Create(dc).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, dc)
}

type AdminUpdateDiscountRequest struct {
	Code            string  `json:"code"`
	DiscountType    string  `json:"discount_type"`
	DiscountValue   float64 `json:"discount_value"`
	Currency        string  `json:"currency"`
	MaxUses         int     `json:"max_uses"`
	ValidFrom       int64   `json:"valid_from"`
	ValidUntil      int64   `json:"valid_until"`
	ApplicablePlans string  `json:"applicable_plans"`
	MinOrderAmount  float64 `json:"min_order_amount"`
	Enabled         bool    `json:"enabled"`
	Description     string  `json:"description"`
}

func AdminUpdateSubscriptionDiscount(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}

	var req AdminUpdateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	req.Code = strings.TrimSpace(strings.ToUpper(req.Code))
	if req.Code == "" {
		common.ApiErrorMsg(c, "折扣码不能为空")
		return
	}
	if len(req.Code) < 3 {
		common.ApiErrorMsg(c, "折扣码长度至少3位")
		return
	}
	if len(req.Code) > 64 {
		common.ApiErrorMsg(c, "折扣码长度不能超过64位")
		return
	}

	if req.DiscountType == "" {
		req.DiscountType = model.DiscountTypePercentage
	}
	if req.DiscountType != model.DiscountTypePercentage && req.DiscountType != model.DiscountTypeFixed {
		common.ApiErrorMsg(c, "折扣类型无效")
		return
	}

	if req.DiscountValue <= 0 {
		common.ApiErrorMsg(c, "折扣值必须大于0")
		return
	}
	if req.DiscountType == model.DiscountTypePercentage && req.DiscountValue > 100 {
		common.ApiErrorMsg(c, "百分比折扣不能超过100%")
		return
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.MaxUses < 0 {
		req.MaxUses = 0
	}
	if req.MinOrderAmount < 0 {
		req.MinOrderAmount = 0
	}

	// Check if another discount has the same code
	existing, err := model.GetSubscriptionDiscountCodeByCode(req.Code)
	if err == nil && existing != nil && existing.Id != id {
		common.ApiErrorMsg(c, "折扣码已存在")
		return
	}

	updateMap := map[string]interface{}{
		"code":             req.Code,
		"discount_type":    req.DiscountType,
		"discount_value":   req.DiscountValue,
		"currency":         req.Currency,
		"max_uses":         req.MaxUses,
		"valid_from":       req.ValidFrom,
		"valid_until":      req.ValidUntil,
		"applicable_plans": req.ApplicablePlans,
		"min_order_amount": req.MinOrderAmount,
		"enabled":          req.Enabled,
		"description":      req.Description,
		"updated_at":       common.GetTimestamp(),
	}

	if err := model.DB.Model(&model.SubscriptionDiscountCode{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, nil)
}

func AdminDeleteSubscriptionDiscount(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	if err := model.DeleteSubscriptionDiscountCode(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

type AdminUpdateDiscountStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func AdminUpdateSubscriptionDiscountStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpdateDiscountStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DB.Model(&model.SubscriptionDiscountCode{}).Where("id = ?", id).Update("enabled", *req.Enabled).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
