package volcenginevideo

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func newTestContext(req relaycommon.TaskSubmitReq) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/videos/generations", nil)
	c.Set("task_request", req)
	return c, w
}

func newTestRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeVolcEngine,
			ChannelBaseUrl: "https://ark.cn-beijing.volces.com",
			ApiKey:         "test-key",
			UpstreamModelName: "doubao-seedance-2-0-260128",
		},
		OriginModelName: "doubao-seedance-2-0-260128",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_abc123",
		},
	}
}

func TestParseSizeToRatio(t *testing.T) {
	cases := []struct {
		size     string
		expected string
	}{
		{"1920x1080", "16:9"},
		{"1080x1920", "9:16"},
		{"512x512", "1:1"},
		{"16:9", "16:9"},
		{"9:16", "9:16"},
		{"1280x720", "16:9"},
		{"", ""},
	}
	for _, tc := range cases {
		ratio, _ := taskcommon.ParseSizeToRatio(tc.size)
		if ratio != tc.expected {
			t.Errorf("ParseSizeToRatio(%q) = %q, want %q", tc.size, ratio, tc.expected)
		}
	}
}

func TestBuildRequestBodyMapping(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Prompt:          "a cat running on a roof",
		Model:           "doubao-seedance-2-0-260128",
		Images:          []string{"https://example.com/cat.jpg"},
		ReferenceImages: []string{"https://example.com/ref1.jpg"},
		ReferenceVideo:  []string{"https://example.com/ref.mp4"},
		ReferenceAudio:  []string{"https://example.com/ref.mp3"},
		Size:            "1920x1080",
		Duration:        10,
		Resolution:      "1080p",
	}

	c, _ := newTestContext(req)
	a := &TaskAdaptor{}
	a.Init(newTestRelayInfo())

	body, err := a.BuildRequestBody(c, newTestRelayInfo())
	if err != nil {
		t.Fatalf("BuildRequestBody failed: %v", err)
	}

	var payload VolcengineVideoRequest
	if err := common.Unmarshal(readAll(body), &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}

	if payload.Model != "doubao-seedance-2-0-260128" {
		t.Errorf("model = %q, want %q", payload.Model, "doubao-seedance-2-0-260128")
	}
	if payload.Duration != 10 {
		t.Errorf("duration = %d, want 10", payload.Duration)
	}
	if payload.Resolution != "1080p" {
		t.Errorf("resolution = %q, want 1080p", payload.Resolution)
	}
	if payload.Ratio != "16:9" {
		t.Errorf("ratio = %q, want 16:9", payload.Ratio)
	}
	if !payload.GenerateAudio {
		t.Errorf("generate_audio should default to true")
	}
	if payload.Watermark {
		t.Errorf("watermark should default to false")
	}

	if len(payload.Content) != 5 {
		t.Fatalf("content length = %d, want 5", len(payload.Content))
	}
	assertContentItem(t, payload.Content[0], "text", "user", "a cat running on a roof", "", "", "")
	assertContentItem(t, payload.Content[1], "image_url", "first_frame", "", "https://example.com/cat.jpg", "", "")
	assertContentItem(t, payload.Content[2], "image_url", "reference_image", "", "https://example.com/ref1.jpg", "", "")
	assertContentItem(t, payload.Content[3], "video_url", "reference_video", "", "", "https://example.com/ref.mp4", "")
	assertContentItem(t, payload.Content[4], "audio_url", "reference_audio", "", "", "", "https://example.com/ref.mp3")
}

func TestBuildRequestBodyDefaults(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Prompt: "a cat running on a roof",
		Model:  "doubao-seedance-2-0-260128",
	}
	c, _ := newTestContext(req)
	a := &TaskAdaptor{}
	a.Init(newTestRelayInfo())

	body, err := a.BuildRequestBody(c, newTestRelayInfo())
	if err != nil {
		t.Fatalf("BuildRequestBody failed: %v", err)
	}

	var payload VolcengineVideoRequest
	if err := common.Unmarshal(readAll(body), &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}

	if payload.Duration != 5 {
		t.Errorf("duration = %d, want 5", payload.Duration)
	}
	if payload.Resolution != "720p" {
		t.Errorf("resolution = %q, want 720p", payload.Resolution)
	}
	if payload.Ratio != "" {
		t.Errorf("ratio should be empty when not provided, got %q", payload.Ratio)
	}
	if !payload.GenerateAudio {
		t.Errorf("generate_audio should default to true")
	}
}

func assertContentItem(t *testing.T, item VolcengineContentItem, typ, role, text, imageURL, videoURL, audioURL string) {
	t.Helper()
	if item.Type != typ {
		t.Errorf("content type = %q, want %q", item.Type, typ)
	}
	if item.Role != role {
		t.Errorf("content role = %q, want %q", item.Role, role)
	}
	if text != "" && item.Text != text {
		t.Errorf("content text = %q, want %q", item.Text, text)
	}
	if imageURL != "" && (item.ImageURL == nil || item.ImageURL.URL != imageURL) {
		t.Errorf("content image_url = %v, want %q", item.ImageURL, imageURL)
	}
	if videoURL != "" && (item.VideoURL == nil || item.VideoURL.URL != videoURL) {
		t.Errorf("content video_url = %v, want %q", item.VideoURL, videoURL)
	}
	if audioURL != "" && (item.AudioURL == nil || item.AudioURL.URL != audioURL) {
		t.Errorf("content audio_url = %v, want %q", item.AudioURL, audioURL)
	}
}

func TestParseTaskResult(t *testing.T) {
	a := &TaskAdaptor{}

	cases := []struct {
		status       string
		expectedTask model.TaskStatus
		expectedURL  string
	}{
		{"queued", model.TaskStatusQueued, ""},
		{"running", model.TaskStatusInProgress, ""},
		{"succeeded", model.TaskStatusSuccess, "https://cdn.example.com/output.mp4"},
		{"failed", model.TaskStatusFailure, ""},
		{"expired", model.TaskStatusFailure, ""},
		{"cancelled", model.TaskStatusFailure, ""},
	}

	for _, tc := range cases {
		body := []byte(`{
			"id": "task-xxx",
			"status": "` + tc.status + `",
			"content": {
				"video_url": "https://cdn.example.com/output.mp4",
				"last_frame_url": "https://cdn.example.com/last_frame.jpg"
			},
			"usage": {
				"completion_tokens": 1,
				"total_tokens": 2
			}
		}`)
		info, err := a.ParseTaskResult(body)
		if err != nil {
			t.Fatalf("ParseTaskResult failed for %s: %v", tc.status, err)
		}
		if info.Status != string(tc.expectedTask) {
			t.Errorf("status %s: expected %s, got %s", tc.status, tc.expectedTask, info.Status)
		}
		if info.Url != tc.expectedURL {
			t.Errorf("status %s: expected url %q, got %q", tc.status, tc.expectedURL, info.Url)
		}
	}
}

func TestConvertToOpenAIVideo(t *testing.T) {
	result := VolcengineVideoTaskResult{
		ID:     "task-xxx",
		Status: "succeeded",
		Content: VolcengineVideoContent{
			VideoURL:     "https://cdn.example.com/output.mp4",
			LastFrameURL: "https://cdn.example.com/last_frame.jpg",
		},
		Usage: VolcengineUsage{
			CompletionTokens: 1,
			TotalTokens:      2,
		},
		CreatedAt: time.Now().Unix(),
	}
	data, _ := common.Marshal(result)

	originTask := &model.Task{
		TaskID:    "task_abc123",
		CreatedAt: result.CreatedAt,
		Data:      data,
		Properties: model.Properties{
			Input: "a cat running on a roof",
		},
	}

	a := &TaskAdaptor{}
	respBytes, err := a.ConvertToOpenAIVideo(originTask)
	if err != nil {
		t.Fatalf("ConvertToOpenAIVideo failed: %v", err)
	}

	var resp dto.OpenAIVideoGenerationResponse
	if err := common.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if resp.Created != result.CreatedAt {
		t.Errorf("created = %d, want %d", resp.Created, result.CreatedAt)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data length = %d, want 1", len(resp.Data))
	}
	item := resp.Data[0]
	if item.URL != "https://cdn.example.com/output.mp4" {
		t.Errorf("url = %q, want output.mp4", item.URL)
	}
	if item.LastFrameURL != "https://cdn.example.com/last_frame.jpg" {
		t.Errorf("last_frame_url = %q, want last_frame.jpg", item.LastFrameURL)
	}
	if item.RevisedPrompt != "a cat running on a roof" {
		t.Errorf("revised_prompt = %q, want prompt", item.RevisedPrompt)
	}
	if resp.Usage == nil || resp.Usage.CompletionTokens != 1 {
		t.Errorf("usage.completion_tokens = %d, want 1", resp.Usage.CompletionTokens)
	}
}

func readAll(r io.Reader) []byte {
	b, _ := io.ReadAll(r)
	return b
}
