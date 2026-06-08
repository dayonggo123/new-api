package controller

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math"
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
// 文档：https://api.zhunfu.cn/ V1 协议
// API 接口支付地址：{PayAddress}/mapi.php
// 签名规则：参数按 ASCII 排序，sign/sign_type/空值不参与，拼接成 a=b&c=d（不 URL 编码），+ KEY，MD5 小写

type SubscriptionYizhifuV1PayRequest struct {
	PlanId        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

// yizhifuV1MD5Sign 按 V1 文档计算 MD5 签名
func yizhifuV1MD5Sign(params map[string]string, key string) string {
	// 1. 过滤 sign、sign_type 和空值
	filtered := make(map[string]string)
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		filtered[k] = v
	}

	// 2. 按 ASCII 码从小到大排序
	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 拼接成 URL 键值对格式（参数值不要进行 URL 编码）
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, filtered[k]))
	}
	signString := strings.Join(parts, "&")

	// 4. 拼接密钥
	signString = signString + key

	// 5. MD5 小写
	return fmt.Sprintf("%x", md5.Sum([]byte(signString)))
}

// yizhifuV1BuildForm 构建支付表单参数（含签名）
func yizhifuV1BuildForm(params map[string]string, key string) map[string]string {
	params["sign_type"] = "MD5"
	params["sign"] = yizhifuV1MD5Sign(params, key)
	return params
}

// SubscriptionRequestYizhifuV1 发起易支付 V1 订阅支付
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

	// 订阅价格直接使用套餐配置价格，不根据用户当前分组的模型调用倍率进行折扣
	originalAmount := plan.PriceAmount
	finalAmount := originalAmount

	// 检查配置
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
		UserId:         userId,
		PlanId:         plan.Id,
		Money:          math.Round(finalAmount*100) / 100,
		OriginalAmount: math.Round(originalAmount*100) / 100,
		DiscountAmount: 0,
		TradeNo:        tradeNo,
		PaymentMethod:  req.PaymentMethod,
		CreateTime:     time.Now().Unix(),
		Status:         common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	// 获取用户 IP
	clientIP := c.ClientIP()

	// 构造 V1 支付参数
	// 套餐价格存的是 USD，易支付收人民币，需要乘以汇率
	cnyAmount := finalAmount * operation_setting.USDExchangeRate
	moneyStr := strconv.FormatFloat(cnyAmount, 'f', 2, 64)
	common.SysLog(fmt.Sprintf("[YizhifuV1] plan=%s, original=%.2f USD, final=%.2f, cny=%s", plan.Title, originalAmount, finalAmount, moneyStr))
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

	// 计算签名
	signedParams := yizhifuV1BuildForm(params, operation_setting.EpayKey)

	// 调试日志
	common.SysLog(fmt.Sprintf("[YizhifuV1] pay params: %+v", signedParams))

	// 调用 mapi.php 获取支付链接
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

	// 返回支付参数给前端（前端可用 payurl 或 qrcode）
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    signedParams,
		"url":     payUrl,
		"result":  result,
	})
}

// SubscriptionYizhifuV1Notify 易支付 V1 异步通知
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

	// 验证签名
	sign := params["sign"]
	calculatedSign := yizhifuV1MD5Sign(params, operation_setting.EpayKey)
	if sign != calculatedSign {
		common.SysLog(fmt.Sprintf("[YizhifuV1] notify sign mismatch: got=%s, calc=%s", sign, calculatedSign))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	// 验证商户 ID
	if params["pid"] != operation_setting.EpayId {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	// 验证支付状态
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

// SubscriptionYizhifuV1Return 易支付 V1 页面跳转通知
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

	// 验证签名
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

// extractParams 从请求中提取所有参数
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

// mapToValues 将 map 转为 url.Values
func mapToValues(m map[string]string) url.Values {
	values := url.Values{}
	for k, v := range m {
		values.Set(k, v)
	}
	return values
}
