#!/bin/bash
# 部署脚本：新增易支付 V1 渠道并恢复标准 Epay
# 用法：在服务器 /root/new-api 目录下执行：bash deploy_yizhifu_v1.sh

set -e

cd /root/new-api

echo "=== 1. 备份原文件 ==="
cp controller/subscription_payment_epay.go controller/subscription_payment_epay.go.bak.$(date +%s)
cp router/api-router.go router/api-router.go.bak.$(date +%s)

echo "=== 2. 恢复 subscription_payment_epay.go 为标准实现 ==="
cat > controller/subscription_payment_epay.go << 'EOF'
package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type SubscriptionEpayPayRequest struct {
	PlanId        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

func SubscriptionRequestEpay(c *gin.Context) {
	var req SubscriptionEpayPayRequest
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
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}
	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}

	userId := c.GetInt("id")
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

	callBackAddress := service.GetCallbackAddress()
	returnUrl, err := url.Parse(callBackAddress + "/api/subscription/epay/return")
	if err != nil {
		common.ApiErrorMsg(c, "回调地址配置错误")
		return
	}
	notifyUrl, err := url.Parse(callBackAddress + "/api/subscription/epay/notify")
	if err != nil {
		common.ApiErrorMsg(c, "回调地址配置错误")
		return
	}

	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("SUBUSR%dNO%s", userId, tradeNo)

	client := GetEpayClient()
	if client == nil {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}

	order := &model.SubscriptionOrder{
		UserId:        userId,
		PlanId:        plan.Id,
		Money:         plan.PriceAmount,
		TradeNo:       tradeNo,
		PaymentMethod: req.PaymentMethod,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	uri, params, err := client.Purchase(&epay.PurchaseArgs{
		Type:           req.PaymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("SUB:%s", plan.Title),
		Money:          strconv.FormatFloat(plan.PriceAmount, 'f', 2, 64),
		Device:         epay.PC,
		NotifyUrl:      notifyUrl,
		ReturnUrl:      returnUrl,
	})
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success", "data": params, "url": uri})
}

func SubscriptionEpayNotify(c *gin.Context) {
	var params map[string]string

	if c.Request.Method == "POST" {
		if err := c.Request.ParseForm(); err != nil {
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	client := GetEpayClient()
	if client == nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	verifyInfo, err := client.Verify(params)
	if err != nil || !verifyInfo.VerifyStatus {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if verifyInfo.TradeStatus != epay.StatusTradeSuccess {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	LockOrder(verifyInfo.ServiceTradeNo)
	defer UnlockOrder(verifyInfo.ServiceTradeNo)

	if err := model.CompleteSubscriptionOrder(verifyInfo.ServiceTradeNo, common.GetJsonString(verifyInfo)); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	_, _ = c.Writer.Write([]byte("success"))
}

func SubscriptionEpayReturn(c *gin.Context) {
	var params map[string]string

	if c.Request.Method == "POST" {
		if err := c.Request.ParseForm(); err != nil {
			c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
		return
	}

	client := GetEpayClient()
	if client == nil {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
		return
	}
	verifyInfo, err := client.Verify(params)
	if err != nil || !verifyInfo.VerifyStatus {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
		return
	}
	if verifyInfo.TradeStatus == epay.StatusTradeSuccess {
		LockOrder(verifyInfo.ServiceTradeNo)
		defer UnlockOrder(verifyInfo.ServiceTradeNo)
		if err := model.CompleteSubscriptionOrder(verifyInfo.ServiceTradeNo, common.GetJsonString(verifyInfo)); err != nil {
			c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
			return
		}
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=success")
		return
	}
	c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=pending")
}
EOF

echo "=== 3. 创建 subscription_payment_yizhifu.go ==="
cat > controller/subscription_payment_yizhifu.go << 'EOF'
package controller

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// YizhifuV1（易支付 V1 旧版协议）订阅支付
// API 接口支付地址：{PayAddress}/mapi.php
// 签名规则：参数按 ASCII 排序，sign/sign_type/空值不参与，拼接成 a=b&c=d（不 URL 编码），+ KEY，MD5 小写

type SubscriptionYizhifuV1PayRequest struct {
	PlanId        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

func yizhifuV1MD5Sign(params map[string]string, key string) string {
	filtered := make(map[string]string)
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		filtered[k] = v
	}
	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, filtered[k]))
	}
	signString := strings.Join(parts, "&")
	signString = signString + key
	return fmt.Sprintf("%x", md5.Sum([]byte(signString)))
}

func yizhifuV1BuildForm(params map[string]string, key string) map[string]string {
	params["sign_type"] = "MD5"
	params["sign"] = yizhifuV1MD5Sign(params, key)
	return params
}

func SubscriptionRequestYizhifuV1(c *gin.Context) {
	var req SubscriptionYizhifuV1PayRequest
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
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}
	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}

	userId := c.GetInt("id")
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

	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl := callBackAddress + "/api/subscription/yizhifu/return"
	notifyUrl := callBackAddress + "/api/subscription/yizhifu/notify"

	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("SUBUSR%dNO%s", userId, tradeNo)

	order := &model.SubscriptionOrder{
		UserId:        userId,
		PlanId:        plan.Id,
		Money:         plan.PriceAmount,
		TradeNo:       tradeNo,
		PaymentMethod: req.PaymentMethod,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	clientIP := c.ClientIP()
	moneyStr := strconv.FormatFloat(plan.PriceAmount, 'f', 2, 64)
	params := map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         req.PaymentMethod,
		"out_trade_no": tradeNo,
		"notify_url":   notifyUrl,
		"return_url":   returnUrl,
		"name":         fmt.Sprintf("SUB:%s", plan.Title),
		"money":        moneyStr,
		"clientip":     clientIP,
		"param":        "",
	}

	signedParams := yizhifuV1BuildForm(params, operation_setting.EpayKey)
	common.SysLog(fmt.Sprintf("[YizhifuV1] pay params: %+v", signedParams))

	payUrl := strings.TrimSuffix(operation_setting.PayAddress, "/") + "/mapi.php"
	resp, err := http.PostForm(payUrl, mapToValues(signedParams))
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo)
		common.ApiErrorMsg(c, "请求支付接口失败")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo)
		common.ApiErrorMsg(c, "读取支付响应失败")
		return
	}

	common.SysLog(fmt.Sprintf("[YizhifuV1] mapi response: %s", string(body)))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo)
		common.ApiErrorMsg(c, "解析支付响应失败")
		return
	}

	code, _ := result["code"].(float64)
	if code != 1 {
		msg, _ := result["msg"].(string)
		_ = model.ExpireSubscriptionOrder(tradeNo)
		common.ApiErrorMsg(c, fmt.Sprintf("支付接口返回错误: %s", msg))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    signedParams,
		"url":     payUrl,
		"result":  result,
	})
}

func SubscriptionYizhifuV1Notify(c *gin.Context) {
	params := extractParams(c)
	if len(params) == 0 {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	sign := params["sign"]
	calculatedSign := yizhifuV1MD5Sign(params, operation_setting.EpayKey)
	if sign != calculatedSign {
		common.SysLog(fmt.Sprintf("[YizhifuV1] notify sign mismatch: got=%s, calc=%s", sign, calculatedSign))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if params["pid"] != operation_setting.EpayId {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if params["trade_status"] != "TRADE_SUCCESS" {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	tradeNo := params["out_trade_no"]
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	if err := model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(params)); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	_, _ = c.Writer.Write([]byte("success"))
}

func SubscriptionYizhifuV1Return(c *gin.Context) {
	params := extractParams(c)
	if len(params) == 0 {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
		return
	}

	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
		return
	}

	sign := params["sign"]
	calculatedSign := yizhifuV1MD5Sign(params, operation_setting.EpayKey)
	if sign != calculatedSign {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
		return
	}

	if params["pid"] != operation_setting.EpayId {
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
		return
	}

	if params["trade_status"] == "TRADE_SUCCESS" {
		tradeNo := params["out_trade_no"]
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		if err := model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(params)); err != nil {
			c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=fail")
			return
		}
		c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=success")
		return
	}

	c.Redirect(http.StatusFound, system_setting.ServerAddress+"/console/topup?pay=pending")
}

func extractParams(c *gin.Context) map[string]string {
	var params map[string]string
	if c.Request.Method == "POST" {
		if err := c.Request.ParseForm(); err != nil {
			return map[string]string{}
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}
	return params
}

func mapToValues(m map[string]string) url.Values {
	values := url.Values{}
	for k, v := range m {
		values.Set(k, v)
	}
	return values
}
EOF

echo "=== 4. 修改 api-router.go 添加路由 ==="
python3 << 'PYEOF'
import re

with open('router/api-router.go', 'r', encoding='utf-8') as f:
    content = f.read()

# 添加 yizhifu/pay 路由（在 epay/pay 之后）
if 'SubscriptionRequestYizhifuV1' not in content:
    content = content.replace(
        'subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)',
        'subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)\n\t\tsubscriptionRoute.POST("/yizhifu/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestYizhifuV1)'
    )

# 添加 yizhifu notify/return 路由（在 epay return 之后）
if 'SubscriptionYizhifuV1Notify' not in content:
    content = content.replace(
        'apiRouter.POST("/subscription/epay/return", controller.SubscriptionEpayReturn)',
        'apiRouter.POST("/subscription/epay/return", controller.SubscriptionEpayReturn)\n\n\t\tapiRouter.POST("/subscription/yizhifu/notify", controller.SubscriptionYizhifuV1Notify)\n\t\tapiRouter.GET("/subscription/yizhifu/notify", controller.SubscriptionYizhifuV1Notify)\n\t\tapiRouter.GET("/subscription/yizhifu/return", controller.SubscriptionYizhifuV1Return)\n\t\tapiRouter.POST("/subscription/yizhifu/return", controller.SubscriptionYizhifuV1Return)'
    )

with open('router/api-router.go', 'w', encoding='utf-8') as f:
    f.write(content)

print('api-router.go updated')
PYEOF

echo "=== 5. 编译检查 ==="
go build ./...

echo "=== 6. 构建并重启容器 ==="
docker compose down && docker compose up -d --build

echo "=== 部署完成 ==="
echo "新接口："
echo "  POST /api/subscription/yizhifu/pay"
echo "  GET/POST /api/subscription/yizhifu/notify"
echo "  GET/POST /api/subscription/yizhifu/return"
