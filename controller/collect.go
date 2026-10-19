package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// 采集队列接口（Storyboard-Copilot 桌面端 + iOS 快捷指令）
// 规范：docs/spec-collect-api-for-newapi-2026-10-19.md（v1.1）
// 错误码表：400 INVALID_URL / 401 INVALID_TOKEN / 403 DEVICE_REVOKED
//
//	409 QUEUE_FULL / 429 RATE_LIMITED / 400 BATCH_TOO_LARGE / 500 INTERNAL
func collectFail(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"code":    code,
		"message": message,
	})
}

// validateCollectDevice 校验设备令牌（link/ack 从 body 取，pending 从 header 取）
// 校验失败按 IP 限流（5 次/分钟），防爆破
func validateCollectDevice(c *gin.Context, token string) (*model.CollectDevice, bool) {
	ip := c.ClientIP()
	if !service.CollectTokenLimiterAllow(ip) {
		collectFail(c, http.StatusTooManyRequests, "RATE_LIMITED", "令牌校验尝试过于频繁，请稍后再试")
		return nil, false
	}
	device, err := service.ValidateCollectDevice(token)
	if err != nil {
		service.CollectTokenLimiterRecord(ip)
		if errors.Is(err, service.ErrCollectDeviceRevoked) {
			collectFail(c, http.StatusForbidden, "DEVICE_REVOKED", "设备令牌已吊销，请重新生成")
		} else {
			collectFail(c, http.StatusUnauthorized, "INVALID_TOKEN", "设备令牌无效")
		}
		return nil, false
	}
	return device, true
}

// CollectSubmitLink POST /api/collect/link
// iOS 快捷指令调用，token 放 body（配置 header 交互成本高）
func CollectSubmitLink(c *gin.Context) {
	var req dto.CollectLinkSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		collectFail(c, http.StatusBadRequest, "INVALID_URL", "请求格式错误：需要 token 和 url 字段")
		return
	}
	device, ok := validateCollectDevice(c, req.Token)
	if !ok {
		return
	}
	id, duplicated, err := service.SubmitCollectLink(device, req.URL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCollectInvalidURL):
			collectFail(c, http.StatusBadRequest, "INVALID_URL", "未从输入中找到 tiktok/douyin 链接")
		case errors.Is(err, service.ErrCollectQueueFull):
			collectFail(c, http.StatusConflict, "QUEUE_FULL", "待采集积压过多，请先打开桌面端")
		case errors.Is(err, service.ErrCollectRateLimited):
			collectFail(c, http.StatusTooManyRequests, "RATE_LIMITED", "超过单日提交配额（200 条/24h），请明天再来")
		default:
			common.SysError("collect submit error: " + err.Error())
			collectFail(c, http.StatusInternalServerError, "INTERNAL", "服务端异常，请稍后重试")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":         id,
			"duplicated": duplicated,
		},
	})
}

// CollectGetPending GET /api/collect/pending?limit=20
// 桌面端轮询；token 走 X-Collect-Token header；无任务必须 204 空 body
func CollectGetPending(c *gin.Context) {
	token := c.GetHeader("X-Collect-Token")
	if token == "" {
		// 兼容 query 参数（部分 HTTP 客户端配 header 不便）
		token = c.Query("token")
	}
	device, ok := validateCollectDevice(c, token)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := service.GetPendingCollectItems(device, limit)
	if err != nil {
		common.SysError("collect pending error: " + err.Error())
		collectFail(c, http.StatusInternalServerError, "INTERNAL", "服务端异常，请稍后重试")
		return
	}
	if len(items) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	type pendingItem struct {
		Id       int64  `json:"id"`
		Url      string `json:"url"`
		Platform string `json:"platform"`
	}
	pending := make([]pendingItem, 0, len(items))
	for _, item := range items {
		pending = append(pending, pendingItem{
			Id:       item.Id,
			Url:      item.ResolvedUrl,
			Platform: item.Platform,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"items": pending},
	})
}

// CollectAck POST /api/collect/ack
// 桌面端上报下载结果；幂等（终态 id 重复 ack 静默忽略）
func CollectAck(c *gin.Context) {
	var req dto.CollectAckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		collectFail(c, http.StatusBadRequest, "INVALID_URL", "请求格式错误：需要 token 和 results 字段")
		return
	}
	device, ok := validateCollectDevice(c, req.Token)
	if !ok {
		return
	}
	if len(req.Results) == 0 {
		collectFail(c, http.StatusBadRequest, "INVALID_URL", "results 不能为空")
		return
	}
	updated, err := service.AckCollectResults(device, req.Results)
	if err != nil {
		if errors.Is(err, service.ErrCollectBatchTooLarge) {
			collectFail(c, http.StatusBadRequest, "BATCH_TOO_LARGE", "ack 批次过大（上限 100 条）")
		} else {
			common.SysError("collect ack error: " + err.Error())
			collectFail(c, http.StatusInternalServerError, "INTERNAL", "服务端异常，请稍后重试")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"updated": updated},
	})
}

// CollectCreateDevice POST /api/collect/device（网页端，登录态鉴权）
// 明文令牌仅此一次返回
func CollectCreateDevice(c *gin.Context) {
	userId := c.GetInt("id")
	var req dto.CollectDeviceCreateRequest
	// name 可选，允许空 body
	_ = c.ShouldBindJSON(&req)
	device, token, err := service.CreateCollectDevice(userId, req.Name)
	if err != nil {
		common.SysError("collect create device error: " + err.Error())
		collectFail(c, http.StatusInternalServerError, "INTERNAL", "服务端异常，请稍后重试")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":    device.Id,
			"token": token,
			"name":  device.Name,
		},
	})
}

// CollectListDevices GET /api/collect/devices（网页端，登录态鉴权）
// 只返回 id/name/token_hint/last_seen/revoked，绝不返回 token 或 hash
func CollectListDevices(c *gin.Context) {
	userId := c.GetInt("id")
	devices, err := model.GetCollectDevicesByUser(userId)
	if err != nil {
		common.SysError("collect list devices error: " + err.Error())
		collectFail(c, http.StatusInternalServerError, "INTERNAL", "服务端异常，请稍后重试")
		return
	}
	type deviceItem struct {
		Id        int64  `json:"id"`
		Name      string `json:"name"`
		TokenHint string `json:"token_hint"`
		LastSeen  int64  `json:"last_seen"`
		Revoked   bool   `json:"revoked"`
	}
	list := make([]deviceItem, 0, len(devices))
	for _, d := range devices {
		list = append(list, deviceItem{
			Id:        d.Id,
			Name:      d.Name,
			TokenHint: d.TokenHint,
			LastSeen:  d.LastSeen,
			Revoked:   d.Revoked,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"devices": list},
	})
}

// CollectRevokeDevice DELETE /api/collect/device/:id（网页端，登录态鉴权）
// 幂等：重复吊销返回成功
func CollectRevokeDevice(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		collectFail(c, http.StatusBadRequest, "INVALID_URL", "设备 id 格式错误")
		return
	}
	err = model.RevokeCollectDevice(id, userId)
	if err != nil {
		collectFail(c, http.StatusNotFound, "NOT_FOUND", "设备不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"id": id},
	})
}
