package dto

// ========== 请求 DTO ==========

// SharedTemplateListQuery 模板列表查询参数
type SharedTemplateListQuery struct {
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Sort     string `form:"sort"` // newest / popular
}

// SharedTemplateShareRequest 分享模板请求（JSON body，MVP 阶段无文件上传）
type SharedTemplateShareRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Category      string `json:"category" binding:"required"`
	PlanJson      string `json:"planJson" binding:"required"`
	PlanVersion   int    `json:"planVersion"`
	AppMinVersion string `json:"appMinVersion"`
	ThumbnailUrl  string `json:"thumbnailUrl"`
	ThumbnailType string `json:"thumbnailType"`
}

// AuditRequest 审核请求
type AuditRequest struct {
	Action string `json:"action" binding:"required"` // approve / reject
	Reason string `json:"reason"`                    // reject 时必填
}

// SharedTemplateUseRequest 记录使用请求
type SharedTemplateUseRequest struct {
	// 预留，后续可加 source 等字段
}

// AdminSharedTemplateListQuery 管理员模板列表查询参数
type AdminSharedTemplateListQuery struct {
	Status   string `form:"status"` // 筛选状态：pending/approved/rejected，空=全部
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// ========== 响应 DTO ==========

// SharedTemplateAssetInfo 素材信息
type SharedTemplateAssetInfo struct {
	HasAssets  bool  `json:"hasAssets"`
	AssetCount int   `json:"assetCount"`
	TotalSize  int64 `json:"totalSize"`
	ImageCount int   `json:"imageCount"`
	VideoCount int   `json:"videoCount"`
}

// SharedTemplateListItem 模板列表项
type SharedTemplateListItem struct {
	Id            string                   `json:"id"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	ThumbnailUrl  string                   `json:"thumbnailUrl,omitempty"`
	ThumbnailType string                   `json:"thumbnailType,omitempty"`
	Category      string                   `json:"category"`
	AuthorId      int                      `json:"authorId"`
	AuthorName    string                   `json:"authorName"`
	Status        string                   `json:"status"`
	Hidden        bool                     `json:"hidden"`
	AssetInfo     *SharedTemplateAssetInfo `json:"assetInfo,omitempty"`
	UseCount      int                      `json:"useCount"`
	CreatedAt     int64                    `json:"createdAt"`
	UpdatedAt     int64                    `json:"updatedAt"`
}

// SharedTemplateDetail 模板详情
type SharedTemplateDetail struct {
	SharedTemplateListItem
	PlanJson      string `json:"planJson"`
	PlanVersion   int    `json:"planVersion"`
	AppMinVersion string `json:"appMinVersion,omitempty"`
	RejectReason  string `json:"rejectReason,omitempty"`
	ApprovedAt    int64  `json:"approvedAt,omitempty"`
}

// SharedTemplateMineItem 我的模板列表项（精简版）
type SharedTemplateMineItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	RejectReason string `json:"rejectReason,omitempty"`
	Category     string `json:"category"`
	UseCount     int    `json:"useCount"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}

// SharedTemplateListResponse 模板列表分页响应
type SharedTemplateListResponse struct {
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"pageSize"`
	List     []SharedTemplateListItem `json:"list"`
}

// SharedTemplateMineListResponse 我的模板列表分页响应
type SharedTemplateMineListResponse struct {
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
	List     []SharedTemplateMineItem  `json:"list"`
}

// SharedTemplateCreateResponse 创建模板响应
type SharedTemplateCreateResponse struct {
	Id        string `json:"id"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

// AuditResponse 审核响应
type AuditResponse struct {
	Id        string `json:"id"`
	Status    string `json:"status"`
	UpdatedAt int64  `json:"updatedAt"`
}
