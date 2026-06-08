package controller

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/thanhpk/randstr"
)

type SubscriptionStripePayRequest struct {
	PlanId int `json:"plan_id"`
}

func SubscriptionRequestStripePay(c *gin.Context) {
	var req SubscriptionStripePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.StripePriceId == "" {
		common.ApiErrorMsg(c, "该套餐未配置 StripePriceId")
		return
	}
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		common.ApiErrorMsg(c, "Stripe 未配置或密钥无效")
		return
	}
	if setting.StripeWebhookSecret == "" {
		common.ApiErrorMsg(c, "Stripe Webhook 未配置")
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user == nil {
		common.ApiErrorMsg(c, "用户不存在")
		return
	}

	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	reference := fmt.Sprintf("sub-stripe-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "sub_ref_" + common.Sha1([]byte(reference))

	// 订阅价格直接使用套餐配置价格，不根据用户当前分组的模型调用倍率进行折扣
	originalAmount := plan.PriceAmount
	finalAmount := originalAmount

	// Note: Stripe subscription mode uses the configured price ID, which has a fixed price.
	// Group discount for Stripe is recorded in the order but the actual Stripe checkout
	// will still charge the original price. Consider using Stripe coupons for real discounts.
	// For now, we record the discounted amount in the order for tracking purposes.
	payLink, err := genStripeSubscriptionLink(referenceId, user.StripeCustomer, user.Email, plan.StripePriceId)
	if err != nil {
		common.SysLog(fmt.Sprintf("获取Stripe Checkout支付链接失败: %v", err))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	order := &model.SubscriptionOrder{
		UserId:         userId,
		PlanId:         plan.Id,
		Money:          math.Round(finalAmount*100) / 100,
		OriginalAmount: math.Round(originalAmount*100) / 100,
		DiscountAmount: 0,
		TradeNo:        referenceId,
		PaymentMethod:  PaymentMethodStripe,
		CreateTime:     time.Now().Unix(),
		Status:         common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
		},
	})
}

func genStripeSubscriptionLink(referenceId string, customerId string, email string, priceId string) (string, error) {
	stripe.Key = setting.StripeApiSecret

	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(referenceId),
		SuccessURL:        stripe.String(system_setting.ServerAddress + "/console/topup"),
		CancelURL:         stripe.String(system_setting.ServerAddress + "/console/topup"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceId),
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
	}

	if "" == customerId {
		if "" != email {
			params.CustomerEmail = stripe.String(email)
		}
		// 订阅模式下 Stripe 自动创建 customer，不需要也不能传 CustomerCreation
	} else {
		params.Customer = stripe.String(customerId)
	}

	result, err := session.New(params)
	if err != nil {
		// 如果使用了已有的 customerId 但报错（比如换了新 Stripe 账户，旧 customerId 失效），
		// 则回退到不使用 customer，改用 email 让 Stripe 创建新 customer
		if customerId != "" {
			common.SysLog(fmt.Sprintf("Stripe Subscription 使用已有 customer 失败: %v，尝试回退创建新 customer", err))
			fallbackParams := &stripe.CheckoutSessionParams{
				ClientReferenceID: stripe.String(referenceId),
				SuccessURL:        stripe.String(system_setting.ServerAddress + "/console/topup"),
				CancelURL:         stripe.String(system_setting.ServerAddress + "/console/topup"),
				LineItems: []*stripe.CheckoutSessionLineItemParams{
					{
						Price:    stripe.String(priceId),
						Quantity: stripe.Int64(1),
					},
				},
				Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			}
			if email != "" {
				fallbackParams.CustomerEmail = stripe.String(email)
			}
			result, err = session.New(fallbackParams)
			if err != nil {
				return "", err
			}
			return result.URL, nil
		}
		return "", err
	}
	return result.URL, nil
}
