package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// echotikClientInstance 复用同一个 EchoTik 客户端实例。
var echotikClientInstance = NewEchotikClient()

// GetRanklist 获取 EchoTik 视频榜单结果，优先使用本地缓存，缓存缺失或过期时回源。
func GetRanklist(ctx context.Context, params *dto.EchotikRanklistParams, forceRefresh bool) (*dto.EchotikRanklistResult, error) {
	setting := operation_setting.GetEchotikSetting()
	if !setting.EchotikEnabled {
		return nil, errors.New("EchoTik 接口未启用")
	}
	if setting.EchotikUsername == "" || setting.EchotikPassword == "" {
		return nil, errors.New("EchoTik 认证信息未配置")
	}
	if params == nil {
		return nil, errors.New("请求参数为空")
	}

	// 未开启缓存时直接回源。
	if !setting.EchotikCacheEnabled {
		return FetchAndSave(ctx, params)
	}

	key := ranklistParamsToKey(params)

	// 1. 命中未过期缓存且非强制刷新时直接返回。
	if !forceRefresh {
		fresh, err := model.GetFreshEchotikRanklistSnapshot(key)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("Failed to query fresh Echotik ranklist snapshot: %s", err.Error()))
		}
		if fresh != nil {
			return snapshotToResult(fresh), nil
		}
	}

	// 2. 缓存缺失、已过期或强制刷新时回源。
	result, err := FetchAndSave(ctx, params)
	if err == nil {
		return result, nil
	}

	// 3. 回源失败时尝试返回过期缓存（stale-while-error）。
	stale, staleErr := model.GetEchotikRanklistSnapshot(key)
	if staleErr != nil {
		logger.LogError(ctx, fmt.Sprintf("Failed to query stale Echotik ranklist snapshot: %s", staleErr.Error()))
	}
	if stale != nil {
		logger.LogWarn(ctx, fmt.Sprintf("Upstream fetch failed, returning stale cache for %v: %s", params, err.Error()))
		return snapshotToResult(stale), nil
	}

	return nil, err
}

// FetchAndSave 直接回源 EchoTik，解析响应后保存到本地缓存并返回结果。
func FetchAndSave(ctx context.Context, params *dto.EchotikRanklistParams) (*dto.EchotikRanklistResult, error) {
	setting := operation_setting.GetEchotikSetting()
	if !setting.EchotikEnabled {
		return nil, errors.New("EchoTik 接口未启用")
	}
	if setting.EchotikUsername == "" || setting.EchotikPassword == "" {
		return nil, errors.New("EchoTik 认证信息未配置")
	}
	if params == nil {
		return nil, errors.New("请求参数为空")
	}

	body, err := echotikClientInstance.Fetch(ctx, params)
	if err != nil {
		return nil, err
	}

	result, err := parseUpstreamResponse(body)
	if err != nil {
		return nil, err
	}

	// 仅当上游返回 code == 200 且 data 非空时持久化缓存。
	if result.Code != 200 {
		return nil, fmt.Errorf("upstream returned code %d: %s", result.Code, result.Message)
	}

	if err := saveSnapshot(ctx, params, body, result); err != nil {
		logger.LogError(ctx, fmt.Sprintf("Failed to save Echotik ranklist snapshot: %s", err.Error()))
		// 保存失败不影响本次响应返回。
	}

	return result, nil
}

// ComputeExpiresAt 根据数据日期计算缓存过期时间戳（秒）。
func ComputeExpiresAt(date string) int64 {
	setting := operation_setting.GetEchotikSetting()
	now := time.Now()

	ttlSeconds := setting.EchotikCacheOlderTTLSeconds
	dateT, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err == nil {
		dateStart := time.Date(dateT.Year(), dateT.Month(), dateT.Day(), 0, 0, 0, 0, time.Local)
		nowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		daysAgo := int(nowStart.Sub(dateStart).Hours() / 24)
		if daysAgo <= 0 {
			ttlSeconds = setting.EchotikCacheTodayTTLSeconds
		} else if daysAgo <= 7 {
			ttlSeconds = setting.EchotikCacheLast7DaysTTLSeconds
		} else {
			ttlSeconds = setting.EchotikCacheOlderTTLSeconds
		}
	}

	if ttlSeconds <= 0 {
		ttlSeconds = 3600
	}

	return now.Unix() + int64(ttlSeconds)
}

// parseUpstreamResponse 解析上游原始响应，提取 code/message/requestId/data 及条数。
func parseUpstreamResponse(body []byte) (*dto.EchotikRanklistResult, error) {
	var upstream dto.EchotikUpstreamResponse
	if err := common.Unmarshal(body, &upstream); err != nil {
		return nil, fmt.Errorf("failed to parse upstream response: %w", err)
	}

	result := &dto.EchotikRanklistResult{
		RawResponse: string(body),
		Code:        upstream.Code,
		Message:     upstream.Message,
		RequestID:   upstream.RequestID,
	}

	if upstream.Data != nil {
		switch v := upstream.Data.(type) {
		case []any:
			result.ItemCount = len(v)
		case []map[string]any:
			result.ItemCount = len(v)
		default:
			// 其他类型视为空数组
			result.ItemCount = 0
		}
	}

	return result, nil
}

// saveSnapshot 将上游响应解析并保存到数据库。
func saveSnapshot(ctx context.Context, params *dto.EchotikRanklistParams, body []byte, result *dto.EchotikRanklistResult) error {
	if result == nil {
		return errors.New("result is nil")
	}
	if result.ItemCount == 0 {
		// data 为空时不缓存，避免空结果污染缓存。
		return nil
	}

	var upstream dto.EchotikUpstreamResponse
	if err := common.Unmarshal(body, &upstream); err != nil {
		return fmt.Errorf("failed to unmarshal upstream response for items: %w", err)
	}

	itemsJSON, err := common.Marshal(upstream.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	now := time.Now().Unix()
	snapshot := &model.EchotikVideoRanklistSnapshot{
		Date:              params.Date,
		Region:            params.Region,
		VideoRankField:    params.VideoRankField,
		RankType:          params.RankType,
		ProductCategoryID: params.ProductCategoryID,
		CreatedByAI:       params.CreatedByAI,
		PageNum:           params.PageNum,
		PageSize:          params.PageSize,
		RawResponse:       string(body),
		Items:             string(itemsJSON),
		ItemCount:         result.ItemCount,
		UpstreamCode:      result.Code,
		UpstreamMessage:   result.Message,
		UpstreamRequestID: result.RequestID,
		FetchedAt:         now,
		ExpiresAt:         ComputeExpiresAt(params.Date),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := model.UpsertEchotikRanklistSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to upsert snapshot: %w", err)
	}

	return nil
}

// ranklistParamsToKey 将 dto 参数转换为 model 查询键。
func ranklistParamsToKey(params *dto.EchotikRanklistParams) *model.EchotikRanklistKey {
	if params == nil {
		return nil
	}
	return &model.EchotikRanklistKey{
		Date:              params.Date,
		Region:            params.Region,
		VideoRankField:    params.VideoRankField,
		RankType:          params.RankType,
		ProductCategoryID: params.ProductCategoryID,
		CreatedByAI:       params.CreatedByAI,
		PageNum:           params.PageNum,
		PageSize:          params.PageSize,
	}
}

// snapshotToResult 将数据库快照转换为内部返回结果。
func snapshotToResult(snapshot *model.EchotikVideoRanklistSnapshot) *dto.EchotikRanklistResult {
	if snapshot == nil {
		return nil
	}
	return &dto.EchotikRanklistResult{
		RawResponse: snapshot.RawResponse,
		Code:        snapshot.UpstreamCode,
		Message:     snapshot.UpstreamMessage,
		RequestID:   snapshot.UpstreamRequestID,
		ItemCount:   snapshot.ItemCount,
	}
}
