package taskcommon

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

// UnmarshalMetadata converts a map[string]any metadata to a typed struct via JSON round-trip.
// This replaces the repeated pattern: json.Marshal(metadata) → json.Unmarshal(bytes, &target).
func UnmarshalMetadata(metadata map[string]any, target any) error {
	if metadata == nil {
		return nil
	}
	// Prevent metadata from overriding model fields to avoid billing bypass.
	delete(metadata, "model")
	metaBytes, err := common.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata failed: %w", err)
	}
	if err := common.Unmarshal(metaBytes, target); err != nil {
		return fmt.Errorf("unmarshal metadata failed: %w", err)
	}
	return nil
}

// DefaultString returns val if non-empty, otherwise fallback.
func DefaultString(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

// DefaultInt returns val if non-zero, otherwise fallback.
func DefaultInt(val, fallback int) int {
	if val == 0 {
		return fallback
	}
	return val
}

// EncodeLocalTaskID encodes an upstream operation name to a URL-safe base64 string.
// Used by Gemini/Vertex to store upstream names as task IDs.
func EncodeLocalTaskID(name string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(name))
}

// DecodeLocalTaskID decodes a base64-encoded upstream operation name.
func DecodeLocalTaskID(id string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// BuildProxyURL constructs the video proxy URL using the public task ID.
// e.g., "https://your-server.com/v1/videos/task_xxxx/content"
func BuildProxyURL(taskID string) string {
	return fmt.Sprintf("%s/v1/videos/%s/content", system_setting.ServerAddress, taskID)
}

// Status-to-progress mapping constants for polling updates.
const (
	ProgressSubmitted  = "10%"
	ProgressQueued     = "20%"
	ProgressInProgress = "30%"
	ProgressComplete   = "100%"
)

// ---------------------------------------------------------------------------
// BaseBilling — embeddable no-op implementations for TaskAdaptor billing methods.
// Adaptors that do not need custom billing can embed this struct directly.
// ---------------------------------------------------------------------------

type BaseBilling struct{}

// EstimateBilling returns nil (no extra ratios; use base model price).
func (BaseBilling) EstimateBilling(_ *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	return nil
}

// AdjustBillingOnSubmit returns nil (no submit-time adjustment).
func (BaseBilling) AdjustBillingOnSubmit(_ *relaycommon.RelayInfo, _ []byte) map[string]float64 {
	return nil
}

// AdjustBillingOnComplete returns 0 (keep pre-charged amount).
func (BaseBilling) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}

// ---------------------------------------------------------------------------
// Video mapping helpers shared by task adaptors (e.g. Doubao, VolcEngine).
// ---------------------------------------------------------------------------

// ParseSizeToRatio maps OpenAI-style "widthxheight" strings to aspect ratios.
// If the input already contains a colon, it is returned as-is. Well-known sizes
// are normalised (e.g. 1920x1080 -> 16:9). Unknown numeric sizes are reduced by
// their greatest common divisor; if parsing fails, the original value is
// returned so the upstream can decide whether to reject it.
func ParseSizeToRatio(size string) (string, float64) {
	size = strings.TrimSpace(size)
	if size == "" {
		return "", 1.0
	}
	// Already a ratio string.
	if strings.Contains(size, ":") {
		return size, 1.0
	}
	// Well-known presets used by new-api video channels.
	switch strings.ToLower(size) {
	case "1920x1080":
		return "16:9", 1.0
	case "1080x1920":
		return "9:16", 1.0
	case "512x512", "1024x1024":
		return "1:1", 1.0
	case "1280x720":
		return "16:9", 1.0
	case "720x1280":
		return "9:16", 1.0
	}
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return size, 1.0
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return size, 1.0
	}
	g := gcd(w, h)
	return fmt.Sprintf("%d:%d", w/g, h/g), 1.0
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// EstimateDurationRatio returns a billing multiplier relative to the default
// duration of 5 seconds. A 10-second video is billed 2x the base price.
func EstimateDurationRatio(duration int) float64 {
	if duration <= 0 {
		return 1.0
	}
	return float64(duration) / 5.0
}

// EstimateResolutionRatio returns a billing multiplier based on the requested
// resolution. Supported values: 480p, 720p, 1080p, 1440p, 2160p/4k. Unknown
// resolutions return 1.0 so the base price is unchanged.
func EstimateResolutionRatio(resolution string) float64 {
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "480p":
		return 0.5
	case "720p":
		return 1.0
	case "1080p":
		return 1.5
	case "1440p":
		return 2.0
	case "2160p", "4k", "uhd":
		return 3.0
	default:
		return 1.0
	}
}

// HasVideoReference reports whether the request carries any video input that
// should trigger a video-input billing discount/ratio.
func HasVideoReference(req relaycommon.TaskSubmitReq) bool {
	if req.HasVideoReference() {
		return true
	}
	if req.Metadata == nil {
		return false
	}
	contentRaw, ok := req.Metadata["content"]
	if !ok {
		return false
	}
	contentSlice, ok := contentRaw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range contentSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if itemMap["type"] == "video_url" {
			return true
		}
		if _, has := itemMap["video_url"]; has {
			return true
		}
	}
	return false
}
