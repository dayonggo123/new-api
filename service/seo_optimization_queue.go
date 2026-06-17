package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// OptimizationQueueAddRequest 添加优化队列请求
type OptimizationQueueAddRequest struct {
	RecordID    int    `json:"record_id"`
	ContentType string `json:"content_type" binding:"required"` // article / prompt / keyword
	Title       string `json:"title"`
	Keyword     string `json:"keyword"`
	Reason      string `json:"reason"` // low_ctr / low_score / manual
	ScoreBefore int    `json:"score_before"`
	Extra       string `json:"extra"`
}

// AddToOptimizationQueue 添加优化队列项
func AddToOptimizationQueue(req *OptimizationQueueAddRequest) (*model.SEOOptimizationQueueItem, error) {
	item := &model.SEOOptimizationQueueItem{
		RecordID:    req.RecordID,
		ContentType: req.ContentType,
		Title:       req.Title,
		Keyword:     req.Keyword,
		Reason:      req.Reason,
		ScoreBefore: req.ScoreBefore,
		Status:      "pending",
		Extra:       req.Extra,
		CreatedTime: common.GetTimestamp(),
		UpdatedTime: common.GetTimestamp(),
	}
	if item.Reason == "" {
		item.Reason = "manual"
	}
	if err := model.AddSEOOptimizationQueueItem(item); err != nil {
		return nil, err
	}
	return item, nil
}

// ListOptimizationQueue 查询优化队列
func ListOptimizationQueue(status string, limit, offset int) ([]*model.SEOOptimizationQueueItem, int64, error) {
	return model.ListSEOOptimizationQueueItems(status, limit, offset)
}

// UpdateOptimizationQueueStatus 更新队列项状态
func UpdateOptimizationQueueStatus(id int, status string, scoreAfter int) error {
	if status == "" {
		return fmt.Errorf("status is required")
	}
	validStatuses := map[string]bool{"pending": true, "processing": true, "optimized": true, "dismissed": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}
	return model.UpdateSEOOptimizationQueueItemStatus(id, status, scoreAfter)
}

// LowCTROpportunity 低 CTR 机会项
type LowCTROpportunity struct {
	Keyword     string  `json:"keyword"`
	Position    float64 `json:"position"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	CTR         float64 `json:"ctr"`
}

// GetLowCTROpportunities 从监控数据中提取低 CTR 机会
// ctrThreshold：CTR 阈值（百分比，如 5.0），minImpressions：最小展现量
func GetLowCTROpportunities(ctrThreshold float64, minImpressions int) []LowCTROpportunity {
	monitorData := GetSEOMonitorData()
	if monitorData == nil || len(monitorData.TopKeywords) == 0 {
		return []LowCTROpportunity{}
	}

	var result []LowCTROpportunity
	for _, kw := range monitorData.TopKeywords {
		if kw.Impressions >= minImpressions && kw.CTR < ctrThreshold {
			result = append(result, LowCTROpportunity{
				Keyword:     kw.Keyword,
				Position:    kw.Position,
				Impressions: kw.Impressions,
				Clicks:      kw.Clicks,
				CTR:         kw.CTR,
			})
		}
	}
	return result
}

// RankingDropOpportunity 排名下降/低位机会项
type RankingDropOpportunity struct {
	Keyword     string  `json:"keyword"`
	Position    float64 `json:"position"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	CTR         float64 `json:"ctr"`
	Change      float64 `json:"change"` // 环比变化，负数表示下降
	Reason      string  `json:"reason"` // 原因标签
}

// GetRankingDropOpportunities 从监控数据中提取排名下降或排名低位的机会
// positionThreshold：认为需要优化的位置阈值（如 10.0 表示未进首页）
// changeThreshold：排名变化阈值（负数，如 -1.0 表示下降超过 1 位）
func GetRankingDropOpportunities(positionThreshold float64, changeThreshold float64) []RankingDropOpportunity {
	monitorData := GetSEOMonitorData()
	if monitorData == nil || len(monitorData.TopKeywords) == 0 {
		return []RankingDropOpportunity{}
	}

	var result []RankingDropOpportunity
	for _, kw := range monitorData.TopKeywords {
		reason := ""
		if kw.Position >= positionThreshold {
			reason = "排名低位"
		}
		if kw.Change <= changeThreshold {
			if reason != "" {
				reason += " + 排名下降"
			} else {
				reason = "排名下降"
			}
		}
		if reason == "" {
			continue
		}

		result = append(result, RankingDropOpportunity{
			Keyword:     kw.Keyword,
			Position:    kw.Position,
			Impressions: kw.Impressions,
			Clicks:      kw.Clicks,
			CTR:         kw.CTR,
			Change:      kw.Change,
			Reason:      reason,
		})
	}
	return result
}

// GuessContentTypeByKeyword 根据关键词推断对应的内容类型（简化实现）
func GuessContentTypeByKeyword(keyword string) string {
	lower := strings.ToLower(keyword)
	if strings.Contains(lower, "prompt") {
		return "prompt"
	}
	return "article"
}
