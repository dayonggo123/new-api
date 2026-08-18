package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// AlipaySetting 支付宝开放平台（官方直连）支付配置。
// 需要企业支付宝账号：开放平台创建应用（电脑网站支付/手机网站支付能力），
// 获取 AppID、应用私钥（PKCS8/RSA2）、支付宝公钥、AES 密钥（内容加密）。
type AlipaySetting struct {
	AlipayEnabled   bool   `json:"alipay_enabled"`     // 是否启用支付宝支付
	AppID           string `json:"alipay_app_id"`      // 支付宝开放平台应用 AppID
	AppPrivateKey   string `json:"alipay_app_private_key"`  // 应用私钥（PKCS8，RSA2）
	AlipayPublicKey string `json:"alipay_public_key"`  // 支付宝公钥（用于验签）
	AESKey          string `json:"alipay_aes_key"`     // 内容加密 AES 密钥（32 位，新版接口必填）
	IsSandbox       bool   `json:"alipay_is_sandbox"`  // 是否使用沙箱环境（联调用）
}

var alipaySetting = AlipaySetting{
	AlipayEnabled:   false,
	AppID:           "",
	AppPrivateKey:   "",
	AlipayPublicKey: "",
	AESKey:          "",
	IsSandbox:       false,
}

func init() {
	config.GlobalConfig.Register("alipay_setting", &alipaySetting)
}

func GetAlipaySetting() *AlipaySetting {
	return &alipaySetting
}

// AlipayConfigured 判断支付宝配置是否完整可用
func (s *AlipaySetting) Configured() bool {
	return s.AlipayEnabled &&
		s.AppID != "" &&
		s.AppPrivateKey != "" &&
		s.AlipayPublicKey != "" &&
		s.AESKey != ""
}
