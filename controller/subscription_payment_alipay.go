package controller

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/smartwalle/alipay/v3"
	"github.com/thanhpk/randstr"
)

type SubscriptionAlipayPayRequest struct {
	PlanId int `json:"plan_id"`
}

// SubscriptionRequestAlipayPay 支付宝电脑网站支付下单（订阅套餐）
func SubscriptionRequestAlipayPay(c *gin.Context) {
	var req SubscriptionAlipayPayRequest
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

	client, err := getAlipayClient()
	if err != nil {
		common.ApiErrorMsg(c, "支付宝未配置或配置不完整")
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
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

	reference := fmt.Sprintf("sub-alipay-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "sub_ali_" + common.Sha1([]byte(reference))

	originalAmount := plan.PriceAmount
	finalAmount := math.Round(originalAmount*100) / 100
	if finalAmount <= 0 {
		common.ApiErrorMsg(c, "套餐金额无效")
		return
	}

	order := &model.SubscriptionOrder{
		UserId:         userId,
		PlanId:         plan.Id,
		Money:          finalAmount,
		OriginalAmount: finalAmount,
		DiscountAmount: 0,
		TradeNo:        referenceId,
		PaymentMethod:  PaymentMethodAlipay,
		CreateTime:     time.Now().Unix(),
		Status:         common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	notifyURL := strings.TrimRight(service.GetCallbackAddress(), "/") + "/api/user/alipay/notify"
	returnURL := strings.TrimRight(system_setting.ServerAddress, "/") + "/console/log"

	pay := alipay.TradePagePay{}
	pay.NotifyURL = notifyURL
	pay.ReturnURL = returnURL
	pay.Subject = fmt.Sprintf("订阅套餐: %s", plan.Title)
	pay.OutTradeNo = referenceId
	pay.TotalAmount = fmt.Sprintf("%.2f", finalAmount)
	pay.ProductCode = "FAST_INSTANT_TRADE_PAY"
	pay.TimeoutExpress = "30m"

	payURL, err := client.TradePagePay(pay)
	if err != nil {
		common.SysLog(fmt.Sprintf("支付宝订阅下单失败: %v", err))
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payURL.String(),
		},
	})
}
