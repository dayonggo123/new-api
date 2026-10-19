package dto

// CollectLinkSubmitRequest POST /api/collect/link
// token 放 body 而非 header：调用方是 iOS 快捷指令，配置 header 交互成本高
type CollectLinkSubmitRequest struct {
	Token string `json:"token" binding:"required"`
	URL   string `json:"url" binding:"required"`
}

type CollectAckResult struct {
	Id     int64  `json:"id"`
	Status int    `json:"status"` // 只接受 2（成功）或 3（失败）
	Error  string `json:"error"`
}

// CollectAckRequest POST /api/collect/ack
type CollectAckRequest struct {
	Token   string             `json:"token" binding:"required"`
	Results []CollectAckResult `json:"results" binding:"required"`
}

// CollectDeviceCreateRequest POST /api/collect/device
type CollectDeviceCreateRequest struct {
	Name string `json:"name"`
}
