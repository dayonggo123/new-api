package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const echotikRanklistPath = "/api/v3/echotik/video/ranklist"

// EchotikClient 封装对 EchoTik 上游的 HTTP 调用。
// 客户端本身不缓存配置，每次请求都会读取当前 operation_setting，支持热更新。
type EchotikClient struct{}

// NewEchotikClient 创建一个新的 EchoTik 客户端实例。
func NewEchotikClient() *EchotikClient {
	return &EchotikClient{}
}

// Fetch 请求 EchoTik 上游视频榜单接口，返回原始响应字节。
func (c *EchotikClient) Fetch(ctx context.Context, params *dto.EchotikRanklistParams) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("echotik client is nil")
	}
	if params == nil {
		return nil, fmt.Errorf("echotik ranklist params is nil")
	}

	setting := operation_setting.GetEchotikSetting()
	baseURL := setting.EchotikBaseURL
	if baseURL == "" {
		baseURL = "https://open.echotik.live"
	}

	query := params.ToQuery()
	upstreamURL := baseURL + echotikRanklistPath
	if len(query) > 0 {
		upstreamURL = upstreamURL + "?" + query.Encode()
	}

	parsedURL, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream url: %w", err)
	}

	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(
		parsedURL.String(),
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	); err != nil {
		return nil, fmt.Errorf("request blocked by ssrf protection: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create upstream request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(setting.EchotikUsername + ":" + setting.EchotikPassword))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := GetHttpClient()
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upstream: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upstream response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogWarn(ctx, fmt.Sprintf("Echotik upstream returned non-200 status %d: %s", resp.StatusCode, string(body)))
		return body, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	return body, nil
}
