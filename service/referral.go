package service

import (
	"fmt"
	"net/url"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

// referralServerAddress returns the configured public server address.
func referralServerAddress() string {
	base := system_setting.ServerAddress
	if base == "" {
		base = "http://localhost:3000"
	}
	return base
}

// GenerateQRCodeURL builds a public QR code URL for a short link.
func GenerateQRCodeURL(shortLink string) string {
	base := referralServerAddress()
	params := url.Values{}
	params.Set("data", shortLink)
	return fmt.Sprintf("%s/api/public/qr?%s", base, params.Encode())
}

// GetOrCreateUserReferralCode returns the current user's referral code, creating it if necessary.
func GetOrCreateUserReferralCode(userId int) (*model.ReferralCode, error) {
	baseUrl := referralServerAddress()

	code, err := model.CreateOrGetReferralCode(userId, baseUrl, "")
	if err != nil {
		return nil, err
	}
	if code.QrUrl == "" {
		code.QrUrl = GenerateQRCodeURL(code.ShortLink)
		_ = model.DB.Model(code).Update("qr_url", code.QrUrl)
	}
	return code, nil
}

// RecordReferralRelationship records the referral relationship and increments usage count.
func RecordReferralRelationship(code string, inviteeId int, source, contentId, ip, deviceFingerprint string) error {
	rc, err := model.GetReferralCodeByCode(code)
	if err != nil {
		return err
	}
	if rc == nil {
		return fmt.Errorf("referral code not found: %s", code)
	}
	if !rc.IsValid() {
		return fmt.Errorf("referral code is not valid: %s", code)
	}
	if rc.UserId == inviteeId {
		return fmt.Errorf("self-invitation is not allowed")
	}

	if err := model.CreateReferralRelationship(rc.Id, rc.UserId, inviteeId, code, source, contentId, ip, deviceFingerprint); err != nil {
		return err
	}
	return model.IncrementReferralUsedCount(rc.Id)
}

// ResolveInviterIdByCode resolves the inviter user id from a referral code.
func ResolveInviterIdByCode(code string) (int, error) {
	rc, err := model.GetReferralCodeByCode(code)
	if err != nil {
		return 0, err
	}
	if rc == nil || !rc.IsValid() {
		return 0, nil
	}
	return rc.UserId, nil
}

// ValidateReferralCode checks whether a referral code is usable and returns inviter info.
func ValidateReferralCode(code string) (map[string]interface{}, error) {
	rc, err := model.GetReferralCodeByCode(code)
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return map[string]interface{}{
			"valid": false,
		}, nil
	}

	inviter, err := model.GetUserById(rc.UserId, false)
	if err != nil {
		return nil, err
	}
	if inviter == nil || inviter.Status != common.UserStatusEnabled {
		return map[string]interface{}{
			"valid": false,
		}, nil
	}

	valid := rc.IsValid()
	rewardPreview := map[string]interface{}{
		"register_bonus":     common.QuotaForInvitee,
		"topup_rebate_ratio": 0.0,
	}

	return map[string]interface{}{
		"valid":          valid,
		"inviter_id":     inviter.Id,
		"inviter_name":   inviter.Username,
		"reward_preview": rewardPreview,
	}, nil
}
