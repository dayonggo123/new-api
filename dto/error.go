package dto

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

//type OpenAIError struct {
//	Message string `json:"message"`
//	Type    string `json:"type"`
//	Param   string `json:"param"`
//	Code    any    `json:"code"`
//}

type OpenAIErrorWithStatusCode struct {
	Error      types.OpenAIError `json:"error"`
	StatusCode int               `json:"status_code"`
	LocalError bool
}

type GeneralErrorResponse struct {
	Error    json.RawMessage `json:"error"`
	Message  string          `json:"message"`
	Msg      string          `json:"msg"`
	Err      string          `json:"err"`
	ErrorMsg string          `json:"error_msg"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Detail   string          `json:"detail,omitempty"`
	Header   struct {
		Message string `json:"message"`
	} `json:"header"`
	Response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
}

func (e GeneralErrorResponse) TryToOpenAIError() *types.OpenAIError {
	var openAIError types.OpenAIError
	if len(e.Error) > 0 {
		err := common.Unmarshal(e.Error, &openAIError)
		if err == nil && openAIError.Message != "" {
			return &openAIError
		}
	}
	return nil
}

// GeminiGenErrorDetail 匹配 GeminiGen 上游的错误格式:
// {"detail": {"error_code": "...", "message": "..."}}
type GeminiGenErrorDetail struct {
	ErrorCode    string `json:"error_code"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error_message"`
}

type GeminiGenErrorResponse struct {
	Detail GeminiGenErrorDetail `json:"detail"`
}

// TryParseGeminiGenError 尝试解析 GeminiGen 格式的错误响应，返回 error_code 和 message。
func TryParseGeminiGenError(body []byte) (code, message string) {
	var resp GeminiGenErrorResponse
	if err := common.Unmarshal(body, &resp); err == nil {
		msg := resp.Detail.Message
		if msg == "" {
			msg = resp.Detail.ErrorMessage
		}
		if msg != "" {
			return resp.Detail.ErrorCode, msg
		}
	}
	return "", ""
}

func (e GeneralErrorResponse) ToMessage() string {
	if len(e.Error) > 0 {
		switch common.GetJsonType(e.Error) {
		case "object":
			var openAIError types.OpenAIError
			err := common.Unmarshal(e.Error, &openAIError)
			if err == nil && openAIError.Message != "" {
				return openAIError.Message
			}
		case "string":
			var msg string
			err := common.Unmarshal(e.Error, &msg)
			if err == nil && msg != "" {
				return msg
			}
		default:
			return string(e.Error)
		}
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != "" {
		return e.Err
	}
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	if e.Detail != "" {
		return e.Detail
	}
	if e.Header.Message != "" {
		return e.Header.Message
	}
	if e.Response.Error.Message != "" {
		return e.Response.Error.Message
	}
	return ""
}
