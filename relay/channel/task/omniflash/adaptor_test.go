package omniflash

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestContext(t *testing.T, req relaycommon.TaskSubmitReq) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("task_request", req)
	return c, w
}

func relayInfoWithTask() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
}

func TestBuildRequestURL(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://generativelanguage.googleapis.com"}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://generativelanguage.googleapis.com",
			UpstreamModelName: "gemini-omni-flash-preview",
		},
	}

	url, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.True(t, strings.Contains(url, "/models/gemini-omni-flash-preview:interactions"))
}

func TestBuildRequestBody(t *testing.T) {
	t.Run("text only request", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt: "a cat dancing",
			Model:  "gemini-omni-flash-preview",
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.NoError(t, err)
		require.NotNil(t, body)

		data, err := io.ReadAll(body)
		require.NoError(t, err)

		var req OmniFlashRequest
		require.NoError(t, common.Unmarshal(data, &req))
		require.Len(t, req.Input, 1)
		assert.Equal(t, "a cat dancing", req.Input[0].Text)
	})

	t.Run("text and image input", func(t *testing.T) {
		imgB64 := base64.StdEncoding.EncodeToString([]byte("fake-image-bytes"))
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt: "animate this",
			Image:  imgB64,
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.NoError(t, err)

		data, err := io.ReadAll(body)
		require.NoError(t, err)

		var req OmniFlashRequest
		require.NoError(t, common.Unmarshal(data, &req))
		require.Len(t, req.Input, 2)
		assert.Equal(t, "animate this", req.Input[0].Text)
		require.NotNil(t, req.Input[1].Image)
		assert.Equal(t, imgB64, req.Input[1].Image.BytesBase64Encoded)
	})

	t.Run("data URI image input", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt: "animate this",
			Images: []string{"data:image/png;base64,iVBORw0KGgo="},
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.NoError(t, err)

		data, err := io.ReadAll(body)
		require.NoError(t, err)

		var req OmniFlashRequest
		require.NoError(t, common.Unmarshal(data, &req))
		require.Len(t, req.Input, 2)
		require.NotNil(t, req.Input[1].Image)
		assert.Equal(t, "iVBORw0KGgo=", req.Input[1].Image.BytesBase64Encoded)
		assert.Equal(t, "image/png", req.Input[1].Image.MimeType)
	})

	t.Run("reference video input", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt:         "extend this video",
			ReferenceVideo: []string{"https://generativelanguage.googleapis.com/files/abc123"},
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.NoError(t, err)

		data, err := io.ReadAll(body)
		require.NoError(t, err)

		var req OmniFlashRequest
		require.NoError(t, common.Unmarshal(data, &req))
		require.Len(t, req.Input, 2)
		require.NotNil(t, req.Input[1].Reference)
		require.NotNil(t, req.Input[1].Reference.Video)
		assert.Equal(t, "https://generativelanguage.googleapis.com/files/abc123", req.Input[1].Reference.Video.URI)
	})

	t.Run("config from size and duration", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt:  "a wide video",
			Size:    "1920x1080",
			Seconds: "8",
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.NoError(t, err)

		data, err := io.ReadAll(body)
		require.NoError(t, err)

		var req OmniFlashRequest
		require.NoError(t, common.Unmarshal(data, &req))
		assert.Equal(t, "16:9", req.Config.AspectRatio)
		assert.Equal(t, "1080p", req.Config.Resolution)
		assert.Equal(t, 8, req.Config.DurationSeconds)
	})

	t.Run("config from metadata", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt: "metadata config",
			Metadata: map[string]interface{}{
				"aspect_ratio":     "9:16",
				"resolution":       "4k",
				"duration_seconds": 6,
				"session_id":       "session-abc",
			},
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.NoError(t, err)

		data, err := io.ReadAll(body)
		require.NoError(t, err)

		var req OmniFlashRequest
		require.NoError(t, common.Unmarshal(data, &req))
		assert.Equal(t, "9:16", req.Config.AspectRatio)
		assert.Equal(t, "4k", req.Config.Resolution)
		assert.Equal(t, 6, req.Config.DurationSeconds)
		assert.Equal(t, "session-abc", req.Config.SessionID)
	})

	t.Run("explicit fields override metadata", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt:      "explicit wins",
			AspectRatio: "1:1",
			Resolution:  "720p",
			Duration:    4,
			Metadata: map[string]interface{}{
				"aspect_ratio":     "9:16",
				"resolution":       "4k",
				"duration_seconds": 6,
			},
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.NoError(t, err)

		data, err := io.ReadAll(body)
		require.NoError(t, err)

		var req OmniFlashRequest
		require.NoError(t, common.Unmarshal(data, &req))
		assert.Equal(t, "1:1", req.Config.AspectRatio)
		assert.Equal(t, "720p", req.Config.Resolution)
		assert.Equal(t, 4, req.Config.DurationSeconds)
	})

	t.Run("missing task_request", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		body, err := adaptor.BuildRequestBody(c, info)
		require.Error(t, err)
		assert.Nil(t, body)
	})
}

func TestParseTaskResult(t *testing.T) {
	t.Run("in progress", func(t *testing.T) {
		resp := OmniFlashOperationResponse{
			Name: "interactions/abc123",
			Done: false,
		}
		data, err := common.Marshal(resp)
		require.NoError(t, err)

		adaptor := &TaskAdaptor{}
		ti, err := adaptor.ParseTaskResult(data)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatusInProgress, ti.Status)
		assert.Equal(t, "50%", ti.Progress)
	})

	t.Run("success with video", func(t *testing.T) {
		resp := OmniFlashOperationResponse{
			Name: "interactions/abc123",
			Done: true,
		}
		resp.Response.GeneratedVideos = []OmniFlashResultVideo{
			{},
		}
		resp.Response.GeneratedVideos[0].Video.URI = "https://generativelanguage.googleapis.com/files/video123"
		data, err := common.Marshal(resp)
		require.NoError(t, err)

		adaptor := &TaskAdaptor{}
		ti, err := adaptor.ParseTaskResult(data)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatusSuccess, ti.Status)
		assert.Equal(t, "100%", ti.Progress)
		assert.Equal(t, "https://generativelanguage.googleapis.com/files/video123", ti.RemoteUrl)
		assert.NotEmpty(t, ti.TaskID)
	})

	t.Run("success without video", func(t *testing.T) {
		resp := OmniFlashOperationResponse{
			Name: "interactions/abc123",
			Done: true,
		}
		data, err := common.Marshal(resp)
		require.NoError(t, err)

		adaptor := &TaskAdaptor{}
		ti, err := adaptor.ParseTaskResult(data)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatusSuccess, ti.Status)
		assert.Empty(t, ti.RemoteUrl)
	})

	t.Run("error response", func(t *testing.T) {
		resp := OmniFlashOperationResponse{
			Name: "interactions/abc123",
			Error: struct {
				Message string `json:"message"`
			}{Message: "content policy violation"},
		}
		data, err := common.Marshal(resp)
		require.NoError(t, err)

		adaptor := &TaskAdaptor{}
		ti, err := adaptor.ParseTaskResult(data)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatusFailure, ti.Status)
		assert.Equal(t, "content policy violation", ti.Reason)
		assert.Equal(t, "100%", ti.Progress)
	})
}

func TestEstimateBilling(t *testing.T) {
	t.Run("duration from metadata", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt: "test",
			Metadata: map[string]interface{}{
				"duration_seconds": 8,
				"resolution":       "4k",
			},
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		billing := adaptor.EstimateBilling(c, info)
		assert.Equal(t, float64(8), billing["seconds"])
		assert.Equal(t, 1.5, billing["resolution"])
	})

	t.Run("defaults to 4 seconds when no duration", func(t *testing.T) {
		c, _ := setupTestContext(t, relaycommon.TaskSubmitReq{
			Prompt: "test",
		})
		adaptor := &TaskAdaptor{}
		info := relayInfoWithTask()

		billing := adaptor.EstimateBilling(c, info)
		assert.Equal(t, float64(4), billing["seconds"])
		assert.Equal(t, 1.0, billing["resolution"])
	})
}

func TestParseImageInput(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		assert.Nil(t, parseImageInput(""))
		assert.Nil(t, parseImageInput("   "))
	})

	t.Run("data URI", func(t *testing.T) {
		img := parseImageInput("data:image/png;base64,iVBORw0KGgo=")
		require.NotNil(t, img)
		assert.Equal(t, "image/png", img.MimeType)
		assert.Equal(t, "iVBORw0KGgo=", img.BytesBase64Encoded)
	})

	t.Run("raw base64", func(t *testing.T) {
		b64 := base64.StdEncoding.EncodeToString([]byte("fake-image"))
		img := parseImageInput(b64)
		require.NotNil(t, img)
		assert.Equal(t, b64, img.BytesBase64Encoded)
		assert.NotEmpty(t, img.MimeType)
	})

	t.Run("invalid data URI returns nil", func(t *testing.T) {
		assert.Nil(t, parseImageInput("data:image/png"))
	})
}

func TestSizeToOmniAspectRatio(t *testing.T) {
	assert.Equal(t, "16:9", SizeToOmniAspectRatio("1024x1024"))
	assert.Equal(t, "16:9", SizeToOmniAspectRatio("1920x1080"))
	assert.Equal(t, "9:16", SizeToOmniAspectRatio("1080x1920"))
	assert.Equal(t, "16:9", SizeToOmniAspectRatio("garbage"))
}

func TestSizeToOmniResolution(t *testing.T) {
	assert.Equal(t, "720p", SizeToOmniResolution("1280x720"))
	assert.Equal(t, "1080p", SizeToOmniResolution("1920x1080"))
	assert.Equal(t, "4k", SizeToOmniResolution("3840x2160"))
	assert.Equal(t, "720p", SizeToOmniResolution("garbage"))
}

func TestResolveOmniResolution(t *testing.T) {
	assert.Equal(t, "1080p", ResolveOmniResolution(nil, "1080p", ""))
	assert.Equal(t, "4k", ResolveOmniResolution(map[string]any{"resolution": "4k"}, "", ""))
	assert.Equal(t, "1080p", ResolveOmniResolution(nil, "", "1920x1080"))
	assert.Equal(t, "720p", ResolveOmniResolution(nil, "", ""))
}

func TestOmniResolutionRatio(t *testing.T) {
	assert.Equal(t, 1.5, OmniResolutionRatio("4k"))
	assert.Equal(t, 1.0, OmniResolutionRatio("1080p"))
}

func TestDoResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/v1/videos/generations", nil)

	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_abc123",
		},
		OriginModelName: "gemini-omni-flash-preview",
	}

	respBody, _ := json.Marshal(OmniFlashSubmitResponse{Name: "interactions/xyz"})
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     make(http.Header),
	}

	adaptor := &TaskAdaptor{}
	taskID, taskData, taskErr := adaptor.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.NotEmpty(t, taskID)
	assert.NotEmpty(t, taskData)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "task_abc123")
}
