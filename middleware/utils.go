package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// parseAcceptLanguage 解析 Accept-Language Header，返回 "zh" 或 "en"
func parseAcceptLanguage(header string) string {
	if header == "" {
		return "zh"
	}
	// 取第一个语言标签
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return "zh"
	}
	lang := strings.TrimSpace(parts[0])
	// 只取主语言标签，去掉区域后缀
	lang = strings.Split(lang, ";")[0]
	lang = strings.TrimSpace(lang)
	// 取主语言代码
	if idx := strings.Index(lang, "-"); idx > 0 {
		lang = lang[:idx]
	}
	if strings.EqualFold(lang, "zh") || strings.EqualFold(lang, "zh-hans") || strings.EqualFold(lang, "zh-hant") || strings.EqualFold(lang, "zh-cn") || strings.EqualFold(lang, "zh-tw") {
		return "zh"
	}
	// 默认英文（任何非中文语言）
	return "en"
}

func abortWithOpenAiMessage(c *gin.Context, statusCode int, message string, code ...types.ErrorCode) {
	codeStr := ""
	errType := "new_api_error"
	var helpURL string
	lang := parseAcceptLanguage(c.GetHeader("Accept-Language"))
	if len(code) > 0 {
		codeStr = string(code[0])
		// 尝试用模板替换 message 和 type
		tpl := types.GetErrorTemplate(code[0])
		if tpl != nil {
			rendered := types.RenderTemplate(tpl, nil, lang)
			if rendered != "" {
				message = rendered
			}
			errType = tpl.Type
			helpURL = tpl.HelpURL
		}
	}
	userId := c.GetInt("id")
	requestId := c.GetString(common.RequestIdKey)

	// 构造结构化响应
	errorObj := gin.H{
		"message": common.MessageWithRequestId(message, requestId),
		"type":    errType,
		"code":    codeStr,
	}
	if helpURL != "" {
		errorObj["help"] = helpURL
	}

	resp := gin.H{
		"error":      errorObj,
		"request_id": requestId,
	}

	c.JSON(statusCode, resp)
	c.Abort()
	logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | request_id=%s | %s", userId, requestId, message))
}

// abortWithOpenAiError 接收完整的 NewAPIError，输出结构化错误响应
func abortWithOpenAiError(c *gin.Context, err *types.NewAPIError) {
	if err == nil {
		abortWithOpenAiMessage(c, http.StatusInternalServerError, "internal server error")
		return
	}

	requestId := c.GetString(common.RequestIdKey)
	if err.RequestID == "" {
		err.SetRequestID(requestId)
	}
	// 设置语言
	lang := parseAcceptLanguage(c.GetHeader("Accept-Language"))
	err.SetLang(lang)

	openaiErr := err.ToOpenAIError()
	userId := c.GetInt("id")

	resp := gin.H{
		"error":      openaiErr,
		"request_id": requestId,
	}

	c.JSON(err.StatusCode, resp)
	c.Abort()
	logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | request_id=%s | %s", userId, requestId, err.Error()))
}

func abortWithMidjourneyMessage(c *gin.Context, statusCode int, code int, description string) {
	c.JSON(statusCode, gin.H{
		"description": description,
		"type":        "new_api_error",
		"code":        code,
	})
	c.Abort()
	logger.LogError(c.Request.Context(), description)
}
