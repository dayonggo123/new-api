package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// SEOMonitorData SEO 监控数据
type SEOMonitorData struct {
	Date            string                   `json:"date"`
	OrganicTraffic  int                      `json:"organic_traffic"`
	IndexedPages    int                      `json:"indexed_pages"`
	RankingKeywords int                      `json:"ranking_keywords"`
	AvgPosition     float64                  `json:"avg_position"`
	TopKeywords     []KeywordRanking         `json:"top_keywords"`
	HealthScore     int                      `json:"health_score"`
	Issues          []MonitorIssue           `json:"issues"`
	UpdatedAt       int64                    `json:"updated_at"`
}

// KeywordRanking 关键词排名
type KeywordRanking struct {
	Keyword   string  `json:"keyword"`
	Position  float64 `json:"position"`
	Clicks    int     `json:"clicks"`
	Impressions int   `json:"impressions"`
	CTR       float64 `json:"ctr"`
	Change    float64 `json:"change"` // 环比变化
}

// MonitorIssue 监控问题
type MonitorIssue struct {
	Type        string `json:"type"`        // error / warning / info
	Category    string `json:"category"`    // indexing / performance / content / technical
	Message     string `json:"message"`
	Count       int    `json:"count"`
	AutoFixable bool   `json:"auto_fixable"`
}

var (
	monitorData     *SEOMonitorData
	monitorDataLock sync.RWMutex
	monitorHistory  []SEOMonitorData // 最近 30 天历史
)

func init() {
	monitorData = &SEOMonitorData{
		Date:            time.Now().Format("2006-01-02"),
		OrganicTraffic:  0,
		IndexedPages:    0,
		RankingKeywords: 0,
		AvgPosition:     0,
		TopKeywords:     []KeywordRanking{},
		HealthScore:     0,
		Issues:          []MonitorIssue{},
		UpdatedAt:       time.Now().Unix(),
	}
	monitorHistory = []SEOMonitorData{}
}

// GetSEOMonitorData 获取当前监控数据
func GetSEOMonitorData() *SEOMonitorData {
	monitorDataLock.RLock()
	defer monitorDataLock.RUnlock()

	// 返回副本
	data := *monitorData
	return &data
}

// GetSEOMonitorHistory 获取监控历史（最近 N 天）
func GetSEOMonitorHistory(days int) []SEOMonitorData {
	monitorDataLock.RLock()
	defer monitorDataLock.RUnlock()

	if days <= 0 {
		days = 30
	}

	if len(monitorHistory) <= days {
		result := make([]SEOMonitorData, len(monitorHistory))
		copy(result, monitorHistory)
		return result
	}

	return monitorHistory[len(monitorHistory)-days:]
}

// UpdateSEOMonitorData 更新监控数据（支持手动更新或 GSC API 回调）
func UpdateSEOMonitorData(data *SEOMonitorData) {
	monitorDataLock.Lock()
	defer monitorDataLock.Unlock()

	// 保存旧数据到历史
	if monitorData.OrganicTraffic > 0 {
		monitorHistory = append(monitorHistory, *monitorData)
		// 只保留最近 90 天
		if len(monitorHistory) > 90 {
			monitorHistory = monitorHistory[len(monitorHistory)-90:]
		}
	}

	data.UpdatedAt = time.Now().Unix()
	monitorData = data
}

// UpdateMonitorFromGSC 从 Google Search Console API 更新数据
func UpdateMonitorFromGSC(siteURL string, startDate, endDate string) error {
	cfg := operation_setting.GetSEOSetting()
	if cfg.GoogleIndexingAPIKey == "" {
		return fmt.Errorf("google api not configured")
	}

	// TODO: 实现 GSC API 调用
	// 需要 OAuth 2.0 认证 + searchanalytics/query API
	// https://developers.google.com/webmaster-tools/search-console-api-original/v3/searchanalytics/query

	return fmt.Errorf("GSC API integration not yet implemented - please use manual update")
}

// SimulateMonitorData 模拟监控数据（用于演示和测试）
func SimulateMonitorData() *SEOMonitorData {
	now := time.Now()

	data := &SEOMonitorData{
		Date:            now.Format("2006-01-02"),
		OrganicTraffic:  1250,
		IndexedPages:    45,
		RankingKeywords: 128,
		AvgPosition:     12.5,
		TopKeywords: []KeywordRanking{
			{Keyword: "AI video prompt library", Position: 3.2, Clicks: 320, Impressions: 1500, CTR: 21.3, Change: 1.5},
			{Keyword: "Sora video prompts", Position: 5.1, Clicks: 180, Impressions: 920, CTR: 19.6, Change: -0.3},
			{Keyword: "node canvas video tool", Position: 2.8, Clicks: 210, Impressions: 680, CTR: 30.9, Change: 2.1},
			{Keyword: "best AI video generators 2026", Position: 8.5, Clicks: 95, Impressions: 2100, CTR: 4.5, Change: 0.8},
			{Keyword: "Kling AI prompts", Position: 4.6, Clicks: 145, Impressions: 580, CTR: 25.0, Change: -1.2},
		},
		HealthScore: 72,
		Issues: []MonitorIssue{
			{Type: "warning", Category: "content", Message: "12 篇文章缺少 GEO 结构化内容", Count: 12, AutoFixable: true},
			{Type: "warning", Category: "technical", Message: "8 个页面未设置多语言 hreflang", Count: 8, AutoFixable: true},
			{Type: "info", Category: "indexing", Message: "3 个页面未被 Google 索引", Count: 3, AutoFixable: false},
			{Type: "info", Category: "performance", Message: "建议增加内容更新频率", Count: 1, AutoFixable: false},
		},
		UpdatedAt: now.Unix(),
	}

	UpdateSEOMonitorData(data)
	return data
}

// CalculateTrafficChange 计算流量环比变化
func CalculateTrafficChange() float64 {
	monitorDataLock.RLock()
	defer monitorDataLock.RUnlock()

	if len(monitorHistory) == 0 {
		return 0
	}

	lastData := monitorHistory[len(monitorHistory)-1]
	if lastData.OrganicTraffic == 0 {
		return 0
	}

	change := float64(monitorData.OrganicTraffic-lastData.OrganicTraffic) * 100 / float64(lastData.OrganicTraffic)
	return change
}

// GetSEOHealthSummary 获取 SEO 健康摘要
func GetSEOHealthSummary() map[string]interface{} {
	monitorDataLock.RLock()
	defer monitorDataLock.RUnlock()

	trafficChange := CalculateTrafficChange()

	return map[string]interface{}{
		"health_score":     monitorData.HealthScore,
		"organic_traffic":  monitorData.OrganicTraffic,
		"traffic_change":   trafficChange,
		"indexed_pages":    monitorData.IndexedPages,
		"ranking_keywords": monitorData.RankingKeywords,
		"avg_position":     monitorData.AvgPosition,
		"issues_count":     len(monitorData.Issues),
		"last_updated":     monitorData.UpdatedAt,
	}
}
