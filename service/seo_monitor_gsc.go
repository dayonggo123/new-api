package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/searchconsole/v1"
)

// GSCConfig 解析后的 Google Service Account 配置
type GSCConfig struct {
	Type        string `json:"type"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
	PrivateKey  string `json:"private_key"`
}

// ValidateGSCConfig 检查 GSC 配置是否完整
func ValidateGSCConfig() error {
	cfg := operation_setting.GetSEOSetting()
	if cfg.GoogleServiceAccountJSON == "" {
		return fmt.Errorf("未配置 Google Service Account JSON，请先在 SEO 设置中填写")
	}
	if cfg.GSCSiteURL == "" && cfg.SiteDomain == "" {
		return fmt.Errorf("未配置 GSC 站点 URL 或网站域名")
	}
	return nil
}

// GetGSCSiteURL 获取最终使用的 GSC 站点 URL
func GetGSCSiteURL() string {
	cfg := operation_setting.GetSEOSetting()
	if cfg.GSCSiteURL != "" {
		return cfg.GSCSiteURL
	}
	if cfg.SiteDomain != "" {
		// 默认尝试 https 协议；GSC 也支持 sc-domain: 前缀表示域名级属性
		return "https://" + cfg.SiteDomain + "/"
	}
	return ""
}

// createGSCClient 使用 Service Account JSON 创建 OAuth2 HTTP 客户端
func createGSCClient(saJSON string) (*http.Client, error) {
	ctx := context.Background()

	// 解析 JSON 以验证字段
	var gscCfg GSCConfig
	if err := json.Unmarshal([]byte(saJSON), &gscCfg); err != nil {
		return nil, fmt.Errorf("Service Account JSON 格式错误: %v", err)
	}
	if gscCfg.ClientEmail == "" || gscCfg.PrivateKey == "" {
		return nil, fmt.Errorf("Service Account JSON 缺少 client_email 或 private_key")
	}

	// Google 的 searchconsole/v1 只需要 WebmastersReadonly 或 Webmasters 范围
	conf, err := google.JWTConfigFromJSON(
		[]byte(saJSON),
		searchconsole.WebmastersReadonlyScope,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 JWT 配置失败: %v", err)
	}

	return conf.Client(ctx), nil
}

// FetchGSCSearchAnalytics 从 Google Search Console 拉取搜索分析数据
// siteURL: GSC 属性 URL，如 https://harse.tv/ 或 sc-domain:harse.tv
// startDate/endDate: 格式 2006-01-02
func FetchGSCSearchAnalytics(siteURL, startDate, endDate string) (*SEOMonitorData, error) {
	if err := ValidateGSCConfig(); err != nil {
		return nil, err
	}

	cfg := operation_setting.GetSEOSetting()
	if siteURL == "" {
		siteURL = GetGSCSiteURL()
	}

	client, err := createGSCClient(cfg.GoogleServiceAccountJSON)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := searchconsole.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("创建 Search Console 服务失败: %v", err)
	}

	// 1. 拉取按 query 分组的数据
	queryReq := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"query"},
		RowLimit:   100,
	}

	queryResp, err := svc.Searchanalytics.Query(siteURL, queryReq).Do()
	if err != nil {
		return nil, fmt.Errorf("查询 GSC search analytics 失败: %v", err)
	}

	// 2. 拉取按 page 分组的数据，用于估算已索引/有展现的页面数
	pageReq := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"page"},
		RowLimit:   5000,
	}

	pageResp, err := svc.Searchanalytics.Query(siteURL, pageReq).Do()
	if err != nil {
		return nil, fmt.Errorf("查询 GSC page analytics 失败: %v", err)
	}

	return buildSEOMonitorData(queryResp, pageResp, startDate, endDate)
}

// buildSEOMonitorData 把 GSC 响应转换为 SEOMonitorData
func buildSEOMonitorData(queryResp, pageResp *searchconsole.SearchAnalyticsQueryResponse, startDate, endDate string) (*SEOMonitorData, error) {
	data := &SEOMonitorData{
		Date:        time.Now().Format("2006-01-02"),
		TopKeywords: []KeywordRanking{},
		Issues:      []MonitorIssue{},
		IsSimulated: false,
	}

	if queryResp == nil {
		return data, nil
	}

	var totalClicks, totalImpressions int64
	var weightedPositionSum float64
	var lowCTRCount, top3Count, top10Count int

	for _, row := range queryResp.Rows {
		if len(row.Keys) == 0 {
			continue
		}
		keyword := row.Keys[0]
		clicks := int64(row.Clicks)
		impressions := int64(row.Impressions)
		position := row.Position
		ctr := row.Ctr * 100 // GSC 返回的是 0-1 小数，转成百分比

		totalClicks += clicks
		totalImpressions += impressions
		weightedPositionSum += position * float64(impressions)

		if position <= 3 {
			top3Count++
		}
		if position <= 10 {
			top10Count++
		}
		if ctr < 3.0 && impressions >= 100 {
			lowCTRCount++
		}

		data.TopKeywords = append(data.TopKeywords, KeywordRanking{
			Keyword:     keyword,
			Position:    position,
			Clicks:      int(clicks),
			Impressions: int(impressions),
			CTR:         ctr,
			Change:      0, // 环比在后续统一计算
		})
	}

	// 按点击量排序取 Top
	sort.Slice(data.TopKeywords, func(i, j int) bool {
		return data.TopKeywords[i].Clicks > data.TopKeywords[j].Clicks
	})
	if len(data.TopKeywords) > 50 {
		data.TopKeywords = data.TopKeywords[:50]
	}

	data.OrganicTraffic = int(totalClicks)
	data.RankingKeywords = len(queryResp.Rows)
	if totalImpressions > 0 {
		data.AvgPosition = weightedPositionSum / float64(totalImpressions)
	}

	// 已索引/有展现的页面数
	if pageResp != nil {
		data.IndexedPages = len(pageResp.Rows)
	}

	// 计算环比变化（与历史数据对比）
	calculateKeywordChanges(data)

	// 计算健康评分与问题
	data.HealthScore = calculateHealthScore(data, lowCTRCount, top3Count, top10Count)
	data.Issues = buildMonitorIssues(data, lowCTRCount)

	return data, nil
}

// calculateKeywordChanges 根据 monitorHistory 计算每个关键词的环比变化
func calculateKeywordChanges(data *SEOMonitorData) {
	monitorDataLock.RLock()
	defer monitorDataLock.RUnlock()

	if len(monitorHistory) == 0 {
		return
	}

	lastData := monitorHistory[len(monitorHistory)-1]
	lastMap := make(map[string]KeywordRanking)
	for _, kw := range lastData.TopKeywords {
		lastMap[kw.Keyword] = kw
	}

	for i := range data.TopKeywords {
		if lastKw, ok := lastMap[data.TopKeywords[i].Keyword]; ok {
			data.TopKeywords[i].Change = lastKw.Position - data.TopKeywords[i].Position
		}
	}
}

// calculateHealthScore 基于 GSC 指标计算健康评分
func calculateHealthScore(data *SEOMonitorData, lowCTRCount, top3Count, top10Count int) int {
	score := 60

	// 平均排名越靠前分数越高
	if data.AvgPosition > 0 && data.AvgPosition <= 10 {
		score += 15
	} else if data.AvgPosition > 0 && data.AvgPosition <= 20 {
		score += 5
	}

	// 首页关键词占比
	if data.RankingKeywords > 0 {
		top10Ratio := float64(top10Count) / float64(data.RankingKeywords)
		if top10Ratio >= 0.3 {
			score += 10
		} else if top10Ratio >= 0.1 {
			score += 5
		}
	}

	// 低 CTR 问题扣分
	if lowCTRCount >= 10 {
		score -= 10
	} else if lowCTRCount >= 5 {
		score -= 5
	}

	// 索引页面数加分
	if data.IndexedPages >= 50 {
		score += 5
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// buildMonitorIssues 基于 GSC 数据生成监控问题
func buildMonitorIssues(data *SEOMonitorData, lowCTRCount int) []MonitorIssue {
	var issues []MonitorIssue

	if lowCTRCount > 0 {
		issues = append(issues, MonitorIssue{
			Type:        "warning",
			Category:    "performance",
			Message:     fmt.Sprintf("%d 个关键词 CTR 低于 3%%，建议优化标题和描述", lowCTRCount),
			Count:       lowCTRCount,
			AutoFixable: false,
		})
	}

	if data.AvgPosition > 10 {
		issues = append(issues, MonitorIssue{
			Type:        "warning",
			Category:    "performance",
			Message:     fmt.Sprintf("平均排名 %.1f 未进首页，建议加强内容和外链", data.AvgPosition),
			Count:       1,
			AutoFixable: false,
		})
	}

	if data.IndexedPages == 0 {
		issues = append(issues, MonitorIssue{
			Type:        "error",
			Category:    "indexing",
			Message:     "GSC 未发现已索引页面，请检查站点地图和索引状态",
			Count:       1,
			AutoFixable: false,
		})
	}

	return issues
}

// GetDefaultGSCDateRange 获取默认的 GSC 查询日期范围（最近 7 天）
func GetDefaultGSCDateRange() (startDate, endDate string) {
	end := time.Now().AddDate(0, 0, -2) // GSC 数据通常有 2 天延迟
	start := end.AddDate(0, 0, -6)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

// SyncMonitorFromGSC 一键从 GSC 同步监控数据
func SyncMonitorFromGSC() (*SEOMonitorData, error) {
	if err := ValidateGSCConfig(); err != nil {
		return nil, err
	}

	startDate, endDate := GetDefaultGSCDateRange()
	data, err := FetchGSCSearchAnalytics(GetGSCSiteURL(), startDate, endDate)
	if err != nil {
		return nil, err
	}

	UpdateSEOMonitorData(data)
	return data, nil
}
