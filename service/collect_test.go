package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

// 规范 §4.1 验收：脏文本提取第一条链接 + 尾部标点剥离 + 大小写不敏感
func TestExtractCollectURL(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   string
		wantOk bool
	}{
		{name: "纯链接", raw: "https://vm.tiktok.com/ZGdCHpVmW/", want: "https://vm.tiktok.com/ZGdCHpVmW/", wantOk: true},
		{name: "脏文本包裹", raw: "看这个 https://vm.tiktok.com/ABC/ 不错", want: "https://vm.tiktok.com/ABC/", wantOk: true},
		// 规范 §4.1 剥离清单仅 ASCII：.,;:!?)]}"' —— 全角标点不在其内，保持原样
		{name: "尾部标点剥离-中文句号", raw: "https://vm.tiktok.com/ABC/。", want: "https://vm.tiktok.com/ABC/。", wantOk: true},
		{name: "尾部标点剥离-英文句号", raw: "https://vm.tiktok.com/ABC/.", want: "https://vm.tiktok.com/ABC/", wantOk: true},
		{name: "尾部标点剥离-英文括号", raw: "(https://vm.tiktok.com/ABC/)", want: "https://vm.tiktok.com/ABC/", wantOk: true},
		{name: "大写协议", raw: "HTTPS://VM.TIKTOK.COM/ABC/", want: "HTTPS://VM.TIKTOK.COM/ABC/", wantOk: true},
		{name: "多条取第一条", raw: "https://vm.tiktok.com/AAA/ 和 https://www.douyin.com/video/123", want: "https://vm.tiktok.com/AAA/", wantOk: true},
		{name: "douyin 长链", raw: "https://www.douyin.com/video/7283123456789?x=1", want: "https://www.douyin.com/video/7283123456789?x=1", wantOk: true},
		{name: "tiktok.com 主域", raw: "https://www.tiktok.com/@user/video/123456", want: "https://www.tiktok.com/@user/video/123456", wantOk: true},
		{name: "无链接", raw: "今天天气不错", want: "", wantOk: false},
		{name: "其他域名不算", raw: "https://www.youtube.com/shorts/abc", want: "", wantOk: false},
		{name: "伪 tiktok 域名不算", raw: "https://tiktok.com.evil.com/x", want: "", wantOk: false},
		{name: "换行脏文本", raw: "第一段\nhttps://v.douyin.com/iR8abc/\n第二段", want: "https://v.douyin.com/iR8abc/", wantOk: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractCollectURL(tt.raw)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// 规范 §4.2 平台识别
func TestDetectPlatform(t *testing.T) {
	assert.Equal(t, "tiktok", DetectPlatform("https://vm.tiktok.com/AAA/"))
	assert.Equal(t, "tiktok", DetectPlatform("HTTPS://WWW.TIKTOK.COM/@user/video/1"))
	assert.Equal(t, "douyin", DetectPlatform("https://v.douyin.com/ABC/"))
	assert.Equal(t, "unknown", DetectPlatform("https://example.com/"))
}

// 规范 §4.3：超长输入先截断 4096 再提取
func TestExtractCollectURLAfterTruncation(t *testing.T) {
	// 链接在 4096 之后会被截掉 → 提取失败（截断由 SubmitCollectLink 调用方做，规范明确规定先截断再提取）
	longPrefix := strings.Repeat("a", 4200)
	truncated := (longPrefix + " https://vm.tiktok.com/LATE/")[:CollectMaxURLLen]
	_, ok := ExtractCollectURL(truncated)
	assert.False(t, ok)

	// 链接在前部 → 截断不影响能提取到（尾部垃圾 b 属于 URL 字符，由 resolved_url 512 上限兜底）
	raw := "https://vm.tiktok.com/FRONT/" + strings.Repeat("b", 4200)
	got, ok := ExtractCollectURL(raw[:CollectMaxURLLen])
	assert.True(t, ok)
	assert.True(t, strings.HasPrefix(got, "https://vm.tiktok.com/FRONT/"))
}

// url_key 规范化：大小写/首尾空白不敏感（幂等命中依赖此规则）
func TestCollectLinkURLKeyNormalization(t *testing.T) {
	// 与 model.CollectLinkURLKey 的行为对齐（这里只验证大小写归一语义由 model 层保证）
	k1 := model.CollectLinkURLKey("https://VM.Tiktok.com/ABC/")
	k2 := model.CollectLinkURLKey("https://vm.tiktok.com/abc/")
	assert.Equal(t, k1, k2)
}

// 设备名清洗：控制字符剥离、超长截断、空白回退默认名
func TestSanitizeCollectDeviceName(t *testing.T) {
	assert.Equal(t, "未命名设备", sanitizeCollectDeviceName(""))
	assert.Equal(t, "未命名设备", sanitizeCollectDeviceName("   "))
	assert.Equal(t, "Windows-PC", sanitizeCollectDeviceName("Windows-PC"))
	// 控制字符（\n \x00）被剥离
	assert.Equal(t, "abc", sanitizeCollectDeviceName("a\nb\x00c"))
	long := strings.Repeat("x", 100)
	assert.Len(t, sanitizeCollectDeviceName(long), 64)
}
