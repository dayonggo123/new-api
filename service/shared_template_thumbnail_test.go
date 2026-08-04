package service

import (
	"os"
	"testing"
)

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
