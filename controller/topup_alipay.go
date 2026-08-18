package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/smartwalle/alipay/v3"
	"github.com/thanhpk/randstr"
)

const (
	PaymentMethodAlipay = "alipay"
)

type AlipayPayRequest struct {
	// Amount 充值数量（与 Stripe 一致：展示币种或 tokens）
	Amount int64 `json:"amount"`
	// PaymentMethod 支付方式，须为 "alipay"
	PaymentMethod string `json:"payment_method"`
}

// getAlipayClient 根据配置构建支付宝 SDK 客户端。
// 每次调用构建以保证配置热更新即时生效（SDK 无共享可变状态，开销可忽略）。
func getAlipayClient() (*alipay.Client, error) {
	cfg := operation_setting.GetAlipaySetting()
	if !cfg.Configured() {
		return nil, fmt.Errorf("支付宝未配置或配置不完整")
	}
	client, err := alipay.New(cfg.AppID, cfg.AppPrivateKey, !cfg.IsSandbox)
	if err != nil {
		return nil, fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}
	if err := client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
		return nil, fmt.Errorf("加载支付宝公钥失败: %v", err)
	}
	if err := client.SetEncryptKey(cfg.AESKey); err != nil {
		return nil, fmt.Errorf("设置支付宝 AES 密钥失败: %v", err)
	}
	return client, nil
}

func getAlipayMinTopup() float64 {
	min := operation_setting.MinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		min = min * int(common.QuotaPerUnit)
	}
	return float64(min)
}

// RequestAlipayPay 支付宝电脑网站支付下单（充值）
func RequestAlipayPay(c *gin.Context) {
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.PaymentMethod != PaymentMethodAlipay {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "不支持的支付渠道"})
		return
	}
	if float64(req.Amount) < getAlipayMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %.0f", getAlipayMinTopup())})
		return
	}
	if req.Amount > 10000 {
		c.JSON(http.StatusOK, gin.H{"message": "充值数量不能大于 10000", "data": 10})
		return
	}

	client, err := getAlipayClient()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}

	id := c.GetInt("id")
	user, _ := model.GetUserById(id, false)
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	// 支付宝按人民币计算，金额计算与 epay 一致（展示币种/分组倍率/折扣）
	chargedMoney := getPayMoney(req.Amount, group)
	if chargedMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	reference := fmt.Sprintf("alipay-ref-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "ali_" + common.Sha1([]byte(reference))

	topUp := &model.TopUp{
		UserId:        id,
		Amount:        req.Amount,
		Money:         chargedMoney,
		TradeNo:       referenceId,
		PaymentMethod: PaymentMethodAlipay,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	notifyURL := strings.TrimRight(service.GetCallbackAddress(), "/") + "/api/user/alipay/notify"
	returnURL := strings.TrimRight(system_setting.ServerAddress, "/") + "/console/log"

	pay := alipay.TradePagePay{}
	pay.NotifyURL = notifyURL
	pay.ReturnURL = returnURL
	pay.Subject = fmt.Sprintf("充值 %d (%s)", req.Amount, user.Username)
	pay.OutTradeNo = referenceId
	pay.TotalAmount = fmt.Sprintf("%.2f", chargedMoney)
	pay.ProductCode = "FAST_INSTANT_TRADE_PAY"
	pay.TimeoutExpress = "30m"

	payURL, err := client.TradePagePay(pay)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payURL.String(),
		},
	})
}

// AlipayNotify 支付宝异步通知回调（充值与订阅共用）
// 验签通过后：优先完成订阅订单，否则完成充值订单。
// 必须返回 "success"（纯文本）告知支付宝已处理。
func AlipayNotify(c *gin.Context) {
	client, err := getAlipayClient()
	if err != nil {
		c.String(http.StatusOK, "fail")
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}

	noti, err := client.DecodeNotification(c.Request.Context(), c.Request.Form)
	if err != nil {
		common.SysLog(fmt.Sprintf("支付宝回调验签失败: %v", err))
		c.String(http.StatusOK, "fail")
		return
	}

	status := string(noti.TradeStatus)
	if status != "TRADE_SUCCESS" && status != "TRADE_FINISHED" {
		common.SysLog(fmt.Sprintf("支付宝回调非成功状态: %s, tradeNo=%s", status, noti.OutTradeNo))
		c.String(http.StatusOK, "fail")
		return
	}
	if noti.OutTradeNo == "" {
		common.SysLog("支付宝回调缺少 out_trade_no")
		c.String(http.StatusOK, "fail")
		return
	}

	// 优先尝试完成订阅订单（与 Stripe 回调逻辑一致）
	LockOrder(noti.OutTradeNo)
	defer UnlockOrder(noti.OutTradeNo)
	payload := map[string]any{
		"trade_no":   noti.TradeNo,
		"buyer_id":   noti.BuyerId,
		"total_amount": noti.TotalAmount,
		"trade_status": status,
	}
	if err := model.CompleteSubscriptionOrder(noti.OutTradeNo, common.GetJsonString(payload)); err == nil {
		common.SysLog(fmt.Sprintf("支付宝订阅订单完成: %s", noti.OutTradeNo))
		c.String(http.StatusOK, "success")
		return
	}

	if err := model.Recharge(noti.OutTradeNo, ""); err != nil {
		common.SysLog(fmt.Sprintf("支付宝充值到账失败 %s: %v", noti.OutTradeNo, err))
		c.String(http.StatusOK, "fail")
		return
	}

	common.SysLog(fmt.Sprintf("支付宝充值到账成功: %s, %s元", noti.OutTradeNo, noti.TotalAmount))
	c.String(http.StatusOK, "success")
}
