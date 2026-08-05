package service

import (
	"os"
	"strings"
	"testing"
)

// 模拟一条真实的 R2 presigned URL（513 字符），用于验证解析与压缩
const sampleR2SignedURL = "https://87c129bea46e5e69d2d92f9b9ef83ca8.r2.cloudflarestorage.com/geminigen-prd-upload-bucket/496972/generated_result/video/7d921388-8ca7-11f1-aac3-262631b5d4f2/20260731_061733_0_UTC_0.mp4?response-content-type=application%2Foctet-stream&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=08469da09882d0a162826c69312656fe%2F20260731%2Fauto%2Fs3%2Faws4_request&X-Amz-Date=20260731T062009Z&X-Amz-Expires=604800&X-Amz-SignedHeaders=host&X-Amz-Signature=ba58fcc8f272f52ed7b104a9b0cdc902db2ee9d51e954b9f33a423ce6358bcbf"

func TestNormalizeTemplateThumbnailUrl(t *testing.T) {
	// 设置公网 URL 环境变量
	os.Setenv("UPLOADS_PUBLIC_URL", "https://heharse.cloud/uploads/")
	defer os.Unsetenv("UPLOADS_PUBLIC_URL")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"absolute public url unchanged", "https://heharse.cloud/uploads/permanent/a.jpg", "https://heharse.cloud/uploads/permanent/a.jpg"},
		{"cdn url unchanged", "https://cdn.example.com/x.jpg", "https://cdn.example.com/x.jpg"},
		{"relative path", "/uploads/permanent/b.jpg", "https://heharse.cloud/uploads/permanent/b.jpg"},
		{"localhost absolute", "http://localhost:3000/uploads/permanent/c.jpg", "https://heharse.cloud/uploads/permanent/c.jpg"},
		{"127.0.0.1 absolute", "http://127.0.0.1:3000/uploads/permanent/d.jpg", "https://heharse.cloud/uploads/permanent/d.jpg"},
		{"private ip absolute", "http://192.168.1.100:3000/uploads/permanent/e.jpg", "https://heharse.cloud/uploads/permanent/e.jpg"},
		{"with whitespace", "  /uploads/permanent/f.jpg  ", "https://heharse.cloud/uploads/permanent/f.jpg"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := normalizeTemplateThumbnailUrl(c.in)
			if got != c.want {
				t.Errorf("normalizeTemplateThumbnailUrl(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestIsLocalOrPrivateURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"http://localhost:3000/x", true},
		{"http://127.0.0.1:3000/x", true},
		{"http://192.168.1.10/x", true},
		{"http://10.0.0.5/x", true},
		{"http://172.16.0.1/x", true},
		{"http://172.31.255.255/x", true},
		{"http://172.15.0.1/x", false},
		{"http://172.32.0.1/x", false},
		{"https://heharse.cloud/x", false},
		{"https://cdn.example.com/x", false},
		{"https://www.google.com/x", false},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := isLocalOrPrivateURL(c.in)
			if got != c.want {
				t.Errorf("isLocalOrPrivateURL(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// normalizeR2SignedURL 依赖 storage.R2Enabled()（全局状态，单测时未初始化 -> false），
// 因此这里直接验证纯解析逻辑：压缩后的短路径不超长、无签名参数。
func TestNormalizeR2SignedURLParsing(t *testing.T) {
	if len(sampleR2SignedURL) <= 500 {
		t.Fatalf("test fixture should exceed 500 chars, got %d", len(sampleR2SignedURL))
	}

	m := r2SignedURLRe.FindStringSubmatch(sampleR2SignedURL)
	if len(m) != 3 {
		t.Fatalf("r2SignedURLRe failed to parse: %q", sampleR2SignedURL)
	}
	bucket, keyWithQuery := m[1], m[2]
	if idx := strings.IndexAny(keyWithQuery, "?#"); idx >= 0 {
		keyWithQuery = keyWithQuery[:idx]
	}
	short := r2ShortPrefix + bucket + "/" + keyWithQuery

	if bucket != "geminigen-prd-upload-bucket" {
		t.Errorf("bucket = %q, want geminigen-prd-upload-bucket", bucket)
	}
	if keyWithQuery != "496972/generated_result/video/7d921388-8ca7-11f1-aac3-262631b5d4f2/20260731_061733_0_UTC_0.mp4" {
		t.Errorf("key = %q", keyWithQuery)
	}
	if strings.Contains(short, "X-Amz-") {
		t.Errorf("short path should drop signature params, got %q", short)
	}
	if len(short) > 500 {
		t.Errorf("short path too long: %d chars", len(short))
	}
}

// resolveTemplateThumbnailUrl：单测时 R2 未配置（R2Enabled()=false），
// 验证 r2:// 短路径/完整 R2 URL 在 R2 不可用时的兜底行为，以及非 R2 URL 原样返回。
func TestResolveTemplateThumbnailUrl(t *testing.T) {
	os.Setenv("UPLOADS_PUBLIC_URL", "https://heharse.cloud/uploads/")
	defer os.Unsetenv("UPLOADS_PUBLIC_URL")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"cdn url unchanged", "https://cdn.example.com/x.jpg", "https://cdn.example.com/x.jpg"},
		{"relative path unchanged", "/uploads/permanent/b.jpg", "/uploads/permanent/b.jpg"},
		// R2 未启用时：r2:// 短路径无法解析为可访问 URL，返回空（前端显示占位图，不报错）
		{"r2 short when r2 disabled", "r2://geminigen-prd-upload-bucket/496972/a.mp4", ""},
		// R2 未启用时：完整 R2 URL 原样返回（历史数据兜底）
		{"full r2 url when r2 disabled", sampleR2SignedURL, sampleR2SignedURL},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveTemplateThumbnailUrl(c.in)
			if got != c.want {
				t.Errorf("resolveTemplateThumbnailUrl(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
