package dto

import (
	"net/url"
	"strconv"
)

// EchotikRanklistParams EchoTik 视频榜单请求参数，作为缓存键的 8 个维度。
type EchotikRanklistParams struct {
	Date              string `json:"date"`
	Region            string `json:"region"`
	VideoRankField    int    `json:"video_rank_field"`
	RankType          int    `json:"rank_type"`
	ProductCategoryID string `json:"product_category_id"`
	CreatedByAI       string `json:"created_by_ai"`
	PageNum           int    `json:"page_num"`
	PageSize          int    `json:"page_size"`
}

// ToQuery 将参数转换为 url.Values，用于构造上游请求 URL。
func (p *EchotikRanklistParams) ToQuery() url.Values {
	if p == nil {
		return url.Values{}
	}

	q := url.Values{}
	q.Set("date", p.Date)
	q.Set("region", p.Region)
	q.Set("video_rank_field", strconv.Itoa(p.VideoRankField))
	q.Set("rank_type", strconv.Itoa(p.RankType))
	q.Set("page_num", strconv.Itoa(p.PageNum))
	q.Set("page_size", strconv.Itoa(p.PageSize))
	if p.ProductCategoryID != "" {
		q.Set("product_category_id", p.ProductCategoryID)
	}
	if p.CreatedByAI != "" {
		q.Set("created_by_ai", p.CreatedByAI)
	}
	return q
}

// EchotikRanklistResult 内部返回结果，透传上游原始 JSON。
type EchotikRanklistResult struct {
	RawResponse string `json:"-"`
	Code        int    `json:"code"`
	Message     string `json:"message"`
	RequestID   string `json:"request_id"`
	ItemCount   int    `json:"item_count"`
}

// EchotikUpstreamResponse EchoTik 上游响应 JSON 结构。
type EchotikUpstreamResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Data      any    `json:"data"`
}
