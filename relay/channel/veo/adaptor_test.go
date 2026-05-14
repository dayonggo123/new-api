package veo

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestGetRequestURL(t *testing.T) {
	a := &Adaptor{}

	tests := []struct {
		baseURL  string
		model    string
		reqPath  string
		expected string
	}{
		{"https://api.geminigen.ai", "kling-video-2-6", "", "https://api.geminigen.ai/uapi/v1/video-gen/kling"},
		{"https://api.geminigen.ai", "veo-3.1", "", "https://api.geminigen.ai/uapi/v1/video-gen/veo"},
		{"https://api.geminigen.ai", "nano-banana-pro", "", "https://api.geminigen.ai/uapi/v1/generate_image"},
		// GET query: empty model, extract from request path
		{"https://api.geminigen.ai", "", "/uapi/v1/video-gen/kling", "https://api.geminigen.ai/uapi/v1/video-gen/kling"},
		{"https://api.geminigen.ai", "", "/uapi/v1/video-gen/veo", "https://api.geminigen.ai/uapi/v1/video-gen/veo"},
		{"https://api.geminigen.ai", "", "/uapi/v1/generate_image", "https://api.geminigen.ai/uapi/v1/generate_image"},
		{"https://api.geminigen.ai", "", "/uapi/v1/imagen/grok", "https://api.geminigen.ai/uapi/v1/imagen/grok"},
		{"https://api.geminigen.ai", "", "/uapi/v1/meta_ai/generate", "https://api.geminigen.ai/uapi/v1/meta_ai/generate"},
		// unknown model fallback
		{"https://api.geminigen.ai", "unknown-model", "", "https://api.geminigen.ai/uapi/v1/video-gen/unknown-model"},
	}

	for _, tc := range tests {
		info := &relaycommon.RelayInfo{}
		info.ChannelMeta = &relaycommon.ChannelMeta{
			ChannelBaseUrl:    tc.baseURL,
			UpstreamModelName: tc.model,
		}
		info.RequestURLPath = tc.reqPath
		got, err := a.GetRequestURL(info)
		if err != nil {
			t.Fatalf("GetRequestURL error: %v", err)
		}
		if got != tc.expected {
			t.Errorf("GetRequestURL(base=%q, model=%q, path=%q) = %q, want %q",
				tc.baseURL, tc.model, tc.reqPath, got, tc.expected)
		}
	}
}

func TestValidateKlingInput_Mode(t *testing.T) {
	tests := []struct {
		model string
		mode  string
		want  string // empty means no error
	}{
		{"kling-video-2-6", "standard", ""},
		{"kling-video-2-6", "professional", ""},
		{"kling-video-2-6", "professional_audio", ""},
		{"kling-video-2-6", "invalid_mode", "INVALID_INPUT"},
		{"kling-video-2-5", "relax", ""},
		{"kling-video-2-5", "standard", ""},
		{"kling-video-2-5", "professional", ""},
		{"kling-video-2-5", "invalid_mode", "INVALID_INPUT"},
		{"kling-video-3-0", "standard", ""},
		{"kling-video-3-0", "relax", "INVALID_INPUT"},
		{"veo-3.1", "standard", ""}, // non-kling, no validation
	}
	for _, tc := range tests {
		err := validateKlingInput(tc.model, nil, tc.mode)
		if tc.want == "" {
			if err != nil {
				t.Errorf("validateKlingInput(%q, nil, %q) unexpected error: %v", tc.model, tc.mode, err)
			}
		} else {
			if err == nil {
				t.Errorf("validateKlingInput(%q, nil, %q) expected error containing %q, got nil", tc.model, tc.mode, tc.want)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("validateKlingInput(%q, nil, %q) error = %q, want containing %q", tc.model, tc.mode, err.Error(), tc.want)
			}
		}
	}
}

func TestValidateKlingInput_RequireVideo(t *testing.T) {
	// Motion models require ref_video
	err := validateKlingInput("kling-video-motion-3", nil, "")
	if err == nil || !strings.Contains(err.Error(), "INVALID_VIDEO_FILE") {
		t.Errorf("motion-3 without video: expected INVALID_VIDEO_FILE error, got %v", err)
	}

	err = validateKlingInput("kling-video-motion-3", []filePart{{fieldName: "ref_videos", contentType: "video/mp4"}}, "")
	if err != nil {
		t.Errorf("motion-3 with video: unexpected error: %v", err)
	}

	// Text-to-video models do not require ref_video
	err = validateKlingInput("kling-video-2-6", nil, "")
	if err != nil {
		t.Errorf("2-6 without video: unexpected error: %v", err)
	}
}

func TestValidateKlingInput_FileCount(t *testing.T) {
	// Too many images
	files := make([]filePart, 5)
	for i := range files {
		files[i] = filePart{fieldName: "ref_images"}
	}
	err := validateKlingInput("kling-video-2-6", files, "")
	if err == nil || !strings.Contains(err.Error(), "Too many reference images") {
		t.Errorf("expected too many images error, got %v", err)
	}

	// Too many videos
	files = []filePart{{fieldName: "ref_videos"}, {fieldName: "ref_videos"}}
	err = validateKlingInput("kling-video-2-6", files, "")
	if err == nil || !strings.Contains(err.Error(), "Too many reference videos") {
		t.Errorf("expected too many videos error, got %v", err)
	}
}

func TestValidateKlingInput_FileFormat(t *testing.T) {
	// Invalid image format
	files := []filePart{{fieldName: "ref_images", contentType: "image/gif"}}
	err := validateKlingInput("kling-video-2-6", files, "")
	if err == nil || !strings.Contains(err.Error(), "INVALID_VIDEO_FILE") {
		t.Errorf("expected invalid image format error, got %v", err)
	}

	// Invalid video format
	files = []filePart{{fieldName: "ref_videos", contentType: "video/avi"}}
	err = validateKlingInput("kling-video-2-6", files, "")
	if err == nil || !strings.Contains(err.Error(), "INVALID_VIDEO_FILE") {
		t.Errorf("expected invalid video format error, got %v", err)
	}
}

func TestValidateKlingInput_FileSize(t *testing.T) {
	// Image too large
	files := []filePart{{fieldName: "ref_images", contentType: "image/jpeg", content: make([]byte, maxRefImageSize+1)}}
	err := validateKlingInput("kling-video-2-6", files, "")
	if err == nil || !strings.Contains(err.Error(), "FILE_TOO_LARGE") {
		t.Errorf("expected FILE_TOO_LARGE error for image, got %v", err)
	}

	// Video too large
	files = []filePart{{fieldName: "ref_videos", contentType: "video/mp4", content: make([]byte, maxRefVideoSize+1)}}
	err = validateKlingInput("kling-video-2-6", files, "")
	if err == nil || !strings.Contains(err.Error(), "FILE_TOO_LARGE") {
		t.Errorf("expected FILE_TOO_LARGE error for video, got %v", err)
	}
}

func TestBuildMultipartBody(t *testing.T) {
	var _ *bytes.Buffer = nil // ensure bytes package is used
	files := []filePart{
		{fieldName: "ref_images", fileName: "cat.jpg", content: []byte("fake-image"), contentType: "image/jpeg"},
		{fieldName: "ref_videos", fileName: "motion.mp4", content: []byte("fake-video"), contentType: "video/mp4"},
	}
	buf, ct, err := buildMultipartBody("kling-video-2-6", "a cat dancing", "720p", "standard", "16:9", "5", files)
	if err != nil {
		t.Fatalf("buildMultipartBody error: %v", err)
	}
	if ct == "" || !strings.Contains(ct, "multipart/form-data") {
		t.Errorf("unexpected content type: %q", ct)
	}

	// Parse the multipart body to verify fields
	boundary := strings.TrimPrefix(ct, "multipart/form-data; boundary=")
	reader := multipart.NewReader(buf, boundary)
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("ReadForm error: %v", err)
	}
	defer form.RemoveAll()

	if v := form.Value["model"]; len(v) != 1 || v[0] != "kling-video-2-6" {
		t.Errorf("model field = %v", v)
	}
	if v := form.Value["prompt"]; len(v) != 1 || v[0] != "a cat dancing" {
		t.Errorf("prompt field = %v", v)
	}
	if v := form.Value["mode"]; len(v) != 1 || v[0] != "standard" {
		t.Errorf("mode field = %v", v)
	}
	if v := form.Value["aspect_ratio"]; len(v) != 1 || v[0] != "16:9" {
		t.Errorf("aspect_ratio field = %v", v)
	}
	if v := form.Value["duration"]; len(v) != 1 || v[0] != "5" {
		t.Errorf("duration field = %v", v)
	}
	if v := form.Value["resolution"]; len(v) != 1 || v[0] != "720p" {
		t.Errorf("resolution field = %v", v)
	}

	if len(form.File["ref_images"]) != 1 {
		t.Errorf("ref_images files = %d", len(form.File["ref_images"]))
	}
	if len(form.File["ref_videos"]) != 1 {
		t.Errorf("ref_videos files = %d", len(form.File["ref_videos"]))
	}
}

func TestParseMP4Duration(t *testing.T) {
	// Build a minimal MP4 with moov > mvhd box (version 0)
	// mvhd v0 layout: version(1) + flags(3) + creation_time(4) + modification_time(4) + timescale(4) + duration(4) + ...

	// Let's build a minimal valid mvhd box
	timescale := uint32(30000)
	duration := uint32(600) // 600/30000 = 0.02s
	mvhd := make([]byte, 100)
	mvhd[0] = 0 // version 0
	// version 0: timescale at offset 12, duration at offset 16
	mvhd[12] = byte(timescale >> 24)
	mvhd[13] = byte(timescale >> 16)
	mvhd[14] = byte(timescale >> 8)
	mvhd[15] = byte(timescale)
	mvhd[16] = byte(duration >> 24)
	mvhd[17] = byte(duration >> 16)
	mvhd[18] = byte(duration >> 8)
	mvhd[19] = byte(duration)

	mvhdSize := 8 + len(mvhd)
	mvhdBox := make([]byte, mvhdSize)
	mvhdBox[0] = byte(mvhdSize >> 24)
	mvhdBox[1] = byte(mvhdSize >> 16)
	mvhdBox[2] = byte(mvhdSize >> 8)
	mvhdBox[3] = byte(mvhdSize)
	copy(mvhdBox[4:8], []byte("mvhd"))
	copy(mvhdBox[8:], mvhd)

	moovSize := 8 + len(mvhdBox)
	moovBox := make([]byte, moovSize)
	moovBox[0] = byte(moovSize >> 24)
	moovBox[1] = byte(moovSize >> 16)
	moovBox[2] = byte(moovSize >> 8)
	moovBox[3] = byte(moovSize)
	copy(moovBox[4:8], []byte("moov"))
	copy(moovBox[8:], mvhdBox)

	// ftyp box (required for valid MP4, but our parser only needs moov)
	ftyp := []byte{
		0, 0, 0, 0x18, 'f', 't', 'y', 'p',
		'i', 's', 'o', 'm', 0, 0, 0, 0,
		'i', 's', 'o', 'm', 'm', 'p', '4', '1',
	}

	data := append(ftyp, moovBox...)

	d, err := parseMP4Duration(data)
	if err != nil {
		t.Fatalf("parseMP4Duration error: %v", err)
	}
	expected := float64(duration) / float64(timescale)
	if d != expected {
		t.Errorf("parseMP4Duration = %f, want %f", d, expected)
	}
}

func TestDownloadFile(t *testing.T) {
	// Test with a reliable public URL (skip if network is unavailable)
	data, ct, err := downloadFile("https://www.google.com/favicon.ico")
	if err != nil {
		t.Skipf("Network unavailable, skipping downloadFile test: %v", err)
	}
	if len(data) == 0 {
		t.Error("downloadFile returned empty data")
	}
	if ct == "" {
		t.Error("downloadFile returned empty content type")
	}
}

// Benchmark for buildMultipartBody
func BenchmarkBuildMultipartBody(b *testing.B) {
	files := []filePart{
		{fieldName: "ref_images", fileName: "cat.jpg", content: make([]byte, 1024*1024), contentType: "image/jpeg"},
	}
	for i := 0; i < b.N; i++ {
		_, _, _ = buildMultipartBody("kling-video-2-6", "prompt", "720p", "standard", "16:9", "5", files)
	}
}
