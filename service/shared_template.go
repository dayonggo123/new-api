package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/storage"
	"gorm.io/gorm"
)

const (
	defaultSharedTemplatePageSize = 20
	maxSharedTemplatePageSize     = 50

	// r2ShortPrefix 是 R2 对象的短路径存储格式：r2://<bucket>/<key>
	// 无签名参数，永不过期，且长度远小于 presigned URL（解决 MySQL VARCHAR(500) 超限报 1406）。
	// 对外输出时通过 resolveTemplateThumbnailUrl 动态生成服务器永久代理地址或新鲜签名。
	r2ShortPrefix = "r2://"
)

// r2SignedURLRe 匹配 R2 presigned URL：https://<account>.r2.cloudflarestorage.com/<bucket>/<key>?<signature params>
var r2SignedURLRe = regexp.MustCompile(`^https://[a-zA-Z0-9_-]+\.r2\.cloudflarestorage\.com/([^/]+)/(.+)$`)

// ========== 参数校验 ==========

// validCategories 分类白名单
var validSharedTemplateCategories = map[string]bool{
	"ecommerce":  true,
	"portrait":   true,
	"landscape":  true,
	"commercial": true,
	"creative":   true,
	"other":      true,
}

// validateSharedTemplateCategory 校验分类
func validateSharedTemplateCategory(category string) bool {
	return validSharedTemplateCategories[category]
}

// validatePlanJson 校验 planJson 为合法 JSON 且 steps 非空
func validatePlanJson(planJson string) error {
	var plan map[string]interface{}
	if err := json.Unmarshal([]byte(planJson), &plan); err != nil {
		return fmt.Errorf("planJson is not valid JSON")
	}

	// 检查 steps 字段
	steps, ok := plan["steps"]
	if !ok {
		return fmt.Errorf("planJson missing 'steps' field")
	}
	stepsArr, ok := steps.([]interface{})
	if !ok || len(stepsArr) == 0 {
		return fmt.Errorf("planJson has no valid steps")
	}

	// 检查 templateVersion >= 3
	if version, ok := plan["templateVersion"].(float64); ok {
		if int(version) < 3 {
			return fmt.Errorf("template version too old, minimum required: 3")
		}
	}

	return nil
}

// logSharedTemplatePlanInfo 记录分享请求中 planJson 的关键字段摘要，
// 用于排查客户端字段（如 selectedCustomPrompt*）是否到达服务端
func logSharedTemplatePlanInfo(userId int, planJson string) {
	summary := map[string]interface{}{
		"length": len(planJson),
	}
	// 提取 planJson 中是否包含自定义提示词相关字段（不做完整解析，避免误伤大 plan）
	for _, key := range []string{
		"selectedCustomPrompt", "selected_custom_prompt", "customPrompt", "custom_prompt",
		"promptId", "prompt_id", "promptTitle", "prompt_title", "promptContent", "prompt_content",
		"inputHint", "input_hint",
	} {
		if strings.Contains(planJson, key) {
			summary[key] = true
		}
	}
	common.SysLog(fmt.Sprintf("shared-template share: userId=%d planJson summary=%v", userId, summary))
}

// 匹配 Windows 盘符路径: F:\... 或 C:/...
// 贪婪匹配到空白/JSON 分隔符为止，再在过滤时剔除 X:// 伪盘符（https:// 的 s:）
var planWinPathRe = regexp.MustCompile(`[A-Za-z]:[\\/][^"',}\s\]]+`)

// 匹配 Unix 绝对路径（排除 URL 和 /uploads 相对资源）
var planUnixPathRe = regexp.MustCompile(`/(?:Users|home|data|tmp|var|opt|usr|private)[^\s"',}\]]+`)

// detectLocalAssetPaths 检测 planJson 中是否包含本机本地路径（如 F:\...、C:/...、/Users/...），
// 这些路径其他用户无法访问，分享前必须由客户端上传素材并替换为公网 URL。
// 注意：必须在 JSON 解析后的 plan 树上递归扫描——若直接扫原始 JSON 文本，
// 转义序列 \n（字面反斜杠+n）会被 [A-Za-z]:[\\/] 误判为盘符路径，
// 导致提示词中的 "e:\n\n【Basic" 之类内容被误报为本地路径而拒绝分享。
func detectLocalAssetPaths(planJson string) []string {
	seen := make(map[string]bool)
	var result []string
	add := func(matches []string) {
		for _, m := range matches {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			// Windows 路径在嵌套 JSON 字符串中可能带双重反斜杠（\\），归一化后去重，
			// 避免同一路径被外层序列化字符串与内层解码字符串各报一次
			key := strings.ReplaceAll(m, `\\`, `\`)
			if seen[key] {
				continue
			}
			// 剔除伪盘符：https:// 中的 s://（盘符后紧跟 :// 视为 URL 而非本地路径）
			if strings.HasPrefix(key, "://") || len(key) >= 3 && key[1] == ':' && key[2] == '/' {
				continue
			}
			// 排除已经是 URL 的情况
			if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
				continue
			}
			seen[key] = true
			if len(result) < 10 {
				result = append(result, key)
			}
		}
	}
	scan := func(s string) {
		add(planWinPathRe.FindAllString(s, -1))
		add(planUnixPathRe.FindAllString(s, -1))
	}

	// 优先在解析后的 plan 树上递归扫描（解码后的字符串中 \n 是真正空白，不会被误判）
	var plan interface{}
	if err := json.Unmarshal([]byte(planJson), &plan); err == nil {
		collectPlanStringValues(plan, scan)
		return result
	}
	// 兜底：JSON 解析失败时扫描原始文本（validatePlanJson 前置校验通常已保证可解析）
	scan(planJson)
	return result
}

// collectPlanStringValues 深度遍历 plan 树，对每个字符串值调用 visit；
// 若字符串本身是嵌套 JSON（如序列化的 nodeData），继续展开其内部字符串再扫描。
func collectPlanStringValues(v interface{}, visit func(string)) {
	switch t := v.(type) {
	case string:
		visit(t)
		// 嵌套 JSON 字符串展开，避免其内部转义序列再次触发误报
		trimmed := strings.TrimSpace(t)
		if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && len(trimmed) > 2 {
			var inner interface{}
			if err := json.Unmarshal([]byte(trimmed), &inner); err == nil {
				collectPlanStringValues(inner, visit)
			}
		}
	case []interface{}:
		for _, e := range t {
			collectPlanStringValues(e, visit)
		}
	case map[string]interface{}:
		for _, e := range t {
			collectPlanStringValues(e, visit)
		}
	}
}

// validateNoLocalAssetPaths 校验 planJson 中没有本机本地路径，有则返回错误
func validateNoLocalAssetPaths(planJson string) error {
	localPaths := detectLocalAssetPaths(planJson)
	if len(localPaths) > 0 {
		return fmt.Errorf("planJson contains local file paths that other users cannot access, please upload assets first: %s (and %d more)", localPaths[0], len(localPaths)-1)
	}
	return nil
}

// normalizeTemplateThumbnailUrl 规范化模板封面 URL：
//   - 空值或已是公网 URL 直接返回
//   - 相对路径（如 /uploads/xxx.jpg）自动拼接公网域名
//   - localhost / 内网地址前缀替换为公网域名（取 UPLOADS_PUBLIC_URL 的站点根）
func normalizeTemplateThumbnailUrl(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return url
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		// 若是 localhost / 127.0.0.1 / 内网地址，替换为公网域名
		if isLocalOrPrivateURL(url) {
			publicRoot := getUploadsPublicRoot()
			if publicRoot != "" {
				if idx := strings.Index(url, "/uploads"); idx >= 0 {
					return publicRoot + url[idx:]
				}
			}
		}
		return url
	}
	// 相对路径（/uploads/...）
	if strings.HasPrefix(url, "/") {
		publicRoot := getUploadsPublicRoot()
		if publicRoot != "" {
			return publicRoot + url
		}
	}
	return url
}

// getUploadsPublicRoot 从 UPLOADS_PUBLIC_URL 提取站点根（如 https://heharse.cloud）
func getUploadsPublicRoot() string {
	publicURL := os.Getenv("UPLOADS_PUBLIC_URL")
	if publicURL == "" {
		return ""
	}
	publicURL = strings.TrimRight(publicURL, "/")
	publicURL = strings.TrimSuffix(publicURL, "/uploads")
	return publicURL
}

// normalizeR2SignedURL 将 R2 presigned URL 规范化为短路径 r2://<bucket>/<key>。
// 签名参数（X-Amz-*）直接丢弃，避免超长 URL 触发 MySQL Error 1406，也避免签名过期后失效。
// 仅在 R2 已配置时转换（否则保留完整 URL，由 TEXT 列容纳）。
func normalizeR2SignedURL(rawURL string) string {
	if !storage.R2Enabled() {
		return rawURL
	}
	m := r2SignedURLRe.FindStringSubmatch(strings.TrimSpace(rawURL))
	if len(m) != 3 {
		return rawURL
	}
	bucket, keyWithQuery := m[1], m[2]
	if idx := strings.IndexAny(keyWithQuery, "?#"); idx >= 0 {
		keyWithQuery = keyWithQuery[:idx]
	}
	if bucket == "" || keyWithQuery == "" {
		return rawURL
	}
	return r2ShortPrefix + bucket + "/" + keyWithQuery
}

// resolveTemplateThumbnailUrl 将数据库存储的 thumbnailUrl 转为客户端可访问的 URL：
//   - r2://<bucket>/<key>（新格式）→ 服务器永久代理地址 {root}/api/public/r2?bucket=..&key=..
//     （R2 未配置或公网域名缺失时降级为直接动态签名）
//   - 完整 R2 presigned URL（历史数据）→ 同样转为永久代理地址（避免签名过期后裂图）
//   - 其他 URL（CDN/自有上传）→ 原样返回
func resolveTemplateThumbnailUrl(storeUrl string) string {
	storeUrl = strings.TrimSpace(storeUrl)
	if storeUrl == "" {
		return storeUrl
	}

	bucket, key := "", ""
	switch {
	case strings.HasPrefix(storeUrl, r2ShortPrefix):
		path := strings.TrimPrefix(storeUrl, r2ShortPrefix)
		b, k, ok := strings.Cut(path, "/")
		if ok {
			bucket, key = b, k
		}
		// r2:// 短路径仅在 R2 可用时可解析；否则没有可用的完整 URL 可返回
		if bucket == "" || key == "" {
			return storeUrl
		}
		if !storage.R2Enabled() {
			return ""
		}
		if root := getUploadsPublicRoot(); root != "" {
			return root + "/api/public/r2?bucket=" + url.QueryEscape(bucket) + "&key=" + url.QueryEscape(key)
		}
		if signed, err := storage.PresignBucketObject(bucket, key); err == nil && signed != "" {
			return signed
		}
		return ""
	default:
		if m := r2SignedURLRe.FindStringSubmatch(storeUrl); len(m) == 3 {
			bucket, key = m[1], m[2]
			if idx := strings.IndexAny(key, "?#"); idx >= 0 {
				key = key[:idx]
			}
		}
		// 历史完整 R2 URL：R2 可用时转为永久代理地址；不可用时原样返回（完整 URL 兜底）
		if bucket != "" && key != "" && storage.R2Enabled() {
			if root := getUploadsPublicRoot(); root != "" {
				return root + "/api/public/r2?bucket=" + url.QueryEscape(bucket) + "&key=" + url.QueryEscape(key)
			}
			if signed, err := storage.PresignBucketObject(bucket, key); err == nil && signed != "" {
				return signed
			}
		}
	}

	// 非 R2 URL 或无法解析：原样返回
	return storeUrl
}

// isLocalOrPrivateURL 判断 URL 是否指向本机或内网（localhost、127.x、10.x、192.168.x、172.16-31.x）
func isLocalOrPrivateURL(url string) bool {
	lower := strings.ToLower(url)
	if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") {
		return true
	}
	hostPart := url
	if idx := strings.Index(url, "://"); idx >= 0 {
		hostPart = url[idx+3:]
	}
	if idx := strings.Index(hostPart, "/"); idx >= 0 {
		hostPart = hostPart[:idx]
	}
	if idx := strings.Index(hostPart, ":"); idx >= 0 {
		hostPart = hostPart[:idx]
	}
	if strings.HasPrefix(hostPart, "10.") || strings.HasPrefix(hostPart, "192.168.") {
		return true
	}
	if strings.HasPrefix(hostPart, "172.") {
		parts := strings.Split(hostPart, ".")
		if len(parts) >= 2 {
			second, err := strconv.Atoi(parts[1])
			if err == nil && second >= 16 && second <= 31 {
				return true
			}
		}
	}
	return false
}

// ========== 公开接口 ==========

// GetSharedTemplates 获取公开模板列表
func GetSharedTemplates(query *dto.SharedTemplateListQuery) (*dto.SharedTemplateListResponse, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.GetSharedTemplates(query, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, toSharedTemplateListItem(t))
	}

	return &dto.SharedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// GetSharedTemplateDetail 获取模板详情（只返回 approved 的）
func GetSharedTemplateDetail(templateId string) (*dto.SharedTemplateDetail, error) {
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}
	if template.Status != model.SharedTemplateStatusApproved {
		return nil, fmt.Errorf("template not found")
	}
	return toSharedTemplateDetail(template), nil
}

// ========== 用户接口 ==========

// ShareTemplate 用户分享模板
func ShareTemplate(userId int, userName string, req *dto.SharedTemplateShareRequest) (*dto.SharedTemplateCreateResponse, error) {
	// 1. 参数校验
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)

	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Name) > 200 {
		return nil, fmt.Errorf("name too long, max 200 characters")
	}
	if len(req.Description) > 500 {
		return nil, fmt.Errorf("description too long, max 500 characters")
	}
	if !validateSharedTemplateCategory(req.Category) {
		return nil, fmt.Errorf("invalid category")
	}
	if req.PlanJson == "" {
		return nil, fmt.Errorf("planJson is required")
	}

	// 2. 校验 planJson
	if err := validatePlanJson(req.PlanJson); err != nil {
		return nil, err
	}
	// 2.1 校验 planJson 中没有本机本地路径（素材必须上传后替换为公网 URL）
	if err := validateNoLocalAssetPaths(req.PlanJson); err != nil {
		return nil, err
	}
	logSharedTemplatePlanInfo(userId, req.PlanJson)

	// 3. 生成模板 ID
	templateId := common.GetUUID()
	if len(templateId) > 32 {
		templateId = templateId[:32]
	}

	// 4. 设置默认值
	planVersion := req.PlanVersion
	if planVersion <= 0 {
		planVersion = 3
	}

	// 5. 规范化 ThumbnailUrl：相对路径或 localhost/内网地址修正为公网 URL，
	//    避免封面在其他用户端无法访问（图片裂开）；R2 presigned URL 压缩为 r2:// 短路径，
	//    避免超长（513+ 字符）触发 MySQL Error 1406 Data too long
	req.ThumbnailUrl = normalizeTemplateThumbnailUrl(req.ThumbnailUrl)
	req.ThumbnailUrl = normalizeR2SignedURL(req.ThumbnailUrl)

	// 6. 保存到数据库
	template := &model.SharedTemplate{
		TemplateId:    templateId,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		AuthorId:      userId,
		AuthorName:    userName,
		Status:        model.SharedTemplateStatusPending,
		PlanJson:      req.PlanJson,
		PlanVersion:   planVersion,
		AppMinVersion: req.AppMinVersion,
		ThumbnailUrl:  req.ThumbnailUrl,
		ThumbnailType: req.ThumbnailType,
	}

	if err := template.Insert(); err != nil {
		return nil, fmt.Errorf("failed to create template: %v", err)
	}

	return &dto.SharedTemplateCreateResponse{
		Id:        templateId,
		Status:    model.SharedTemplateStatusPending,
		CreatedAt: template.CreatedAt,
	}, nil
}

// GetMySharedTemplates 获取用户自己的模板列表
func GetMySharedTemplates(userId, page, pageSize int) (*dto.SharedTemplateMineListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.GetSharedTemplatesByAuthor(userId, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateMineItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, dto.SharedTemplateMineItem{
			Id:           t.TemplateId,
			Name:         t.Name,
			Status:       t.Status,
			RejectReason: t.RejectReason,
			Category:     t.Category,
			UseCount:     t.UseCount,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}

	return &dto.SharedTemplateMineListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// DeleteSharedTemplate 删除模板（仅作者）
func DeleteSharedTemplate(templateId string, userId int) error {
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return err
	}

	// 权限校验
	if template.AuthorId != userId {
		return fmt.Errorf("not authorized to delete this template")
	}

	return template.SoftDelete()
}

// RecordSharedTemplateUse 记录模板使用
func RecordSharedTemplateUse(templateId string, userId int) error {
	// 检查模板存在且已发布
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return err
	}
	if template.Status != model.SharedTemplateStatusApproved {
		return fmt.Errorf("template not available")
	}

	// 尝试插入使用记录（FirstOrCreate 跨 DB 兼容，UNIQUE 约束防重复计数）
	if err := model.FindOrCreateTemplateUse(templateId, userId); err != nil {
		return err
	}

	// 增加使用计数
	return model.IncrementSharedTemplateUseCount(templateId)
}

// ========== 管理员接口 ==========

// AdminDeleteSharedTemplate 管理员删除模板（软删除 + 记录审计日志）。
// 与作者删除（DeleteSharedTemplate）不同：管理员可删除任意状态的模板。
func AdminDeleteSharedTemplate(templateId string, adminId int, adminName string) error {
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return err
	}

	if err := template.SoftDelete(); err != nil {
		return fmt.Errorf("failed to delete template: %v", err)
	}

	// 记录审计日志，失败不阻塞主流程
	auditLog := &model.SharedTemplateAuditLog{
		TemplateId: templateId,
		AdminId:    adminId,
		AdminName:  adminName,
		Action:     "delete",
		Reason:     "",
	}
	if err := auditLog.Insert(); err != nil {
		common.SysLog(fmt.Sprintf("failed to write audit log for template %s: %v", templateId, err))
	}
	return nil
}

// GetPendingSharedTemplates 获取待审核列表
func GetPendingSharedTemplates(page, pageSize int) (*dto.SharedTemplateListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.GetPendingSharedTemplates(page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		item := toSharedTemplateListItem(t)
		// 待审核接口返回完整 planJson（放在详情里）
		items = append(items, item)
	}

	return &dto.SharedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// AdminListSharedTemplates 管理后台获取全部模板
func AdminListSharedTemplates(query *dto.AdminSharedTemplateListQuery) (*dto.SharedTemplateListResponse, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = defaultSharedTemplatePageSize
	}
	if pageSize > maxSharedTemplatePageSize {
		return nil, fmt.Errorf("pageSize must be between 1 and %d", maxSharedTemplatePageSize)
	}

	templates, total, err := model.AdminListSharedTemplates(query.Status, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SharedTemplateListItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, toSharedTemplateListItem(t))
	}

	return &dto.SharedTemplateListResponse{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		List:     items,
	}, nil
}

// GetPendingSharedTemplateDetail 获取待审核模板详情（含完整 planJson）
func GetPendingSharedTemplateDetail(templateId string) (*dto.SharedTemplateDetail, error) {
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}
	return toSharedTemplateDetail(template), nil
}

// AuditSharedTemplate 审核模板
func AuditSharedTemplate(templateId string, adminId int, adminName string, req *dto.AuditRequest) (*dto.AuditResponse, error) {
	// 1. 校验 action
	action := strings.TrimSpace(req.Action)
	if action != "approve" && action != "reject" {
		return nil, fmt.Errorf("invalid action, must be 'approve' or 'reject'")
	}

	// 2. reject 时 reason 必填
	reason := strings.TrimSpace(req.Reason)
	if action == "reject" && reason == "" {
		return nil, fmt.Errorf("reason is required when rejecting")
	}
	if len(reason) > 500 {
		return nil, fmt.Errorf("reason too long, max 500 characters")
	}

	// 3. 获取模板
	template, err := model.GetSharedTemplateByTemplateId(templateId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}

	// 4. 状态校验
	if template.Status != model.SharedTemplateStatusPending {
		return nil, fmt.Errorf("template is not in pending status, current: %s", template.Status)
	}

	// 5. 更新状态
	newStatus := model.SharedTemplateStatusApproved
	if action == "reject" {
		newStatus = model.SharedTemplateStatusRejected
	}
	if err := model.UpdateSharedTemplateStatus(templateId, newStatus, reason); err != nil {
		return nil, fmt.Errorf("failed to update template status: %v", err)
	}

	// 6. 记录审核日志
	auditLog := &model.SharedTemplateAuditLog{
		TemplateId: templateId,
		AdminId:    adminId,
		AdminName:  adminName,
		Action:     action,
		Reason:     reason,
	}
	if err := auditLog.Insert(); err != nil {
		// 审核日志写入失败不阻塞主流程
		common.SysLog(fmt.Sprintf("failed to write audit log for template %s: %v", templateId, err))
	}

	return &dto.AuditResponse{
		Id:        templateId,
		Status:    newStatus,
		UpdatedAt: common.GetTimestamp(),
	}, nil
}

// ========== 内部转换函数 ==========

// toSharedTemplateListItem 将 model 转为列表项 DTO
func toSharedTemplateListItem(t *model.SharedTemplate) dto.SharedTemplateListItem {
	item := dto.SharedTemplateListItem{
		Id:            t.TemplateId,
		Name:          t.Name,
		Description:   t.Description,
		ThumbnailUrl:  resolveTemplateThumbnailUrl(t.ThumbnailUrl),
		ThumbnailType: t.ThumbnailType,
		Category:      t.Category,
		AuthorId:      t.AuthorId,
		AuthorName:    t.AuthorName,
		Status:        t.Status,
		UseCount:      t.UseCount,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
	if t.HasAssets {
		item.AssetInfo = &dto.SharedTemplateAssetInfo{
			HasAssets:  t.HasAssets,
			AssetCount: t.AssetCount,
			TotalSize:  t.TotalSize,
			ImageCount: t.ImageCount,
			VideoCount: t.VideoCount,
		}
	}
	return item
}

// toSharedTemplateDetail 将 model 转为详情 DTO
func toSharedTemplateDetail(t *model.SharedTemplate) *dto.SharedTemplateDetail {
	return &dto.SharedTemplateDetail{
		SharedTemplateListItem: toSharedTemplateListItem(t),
		PlanJson:               t.PlanJson,
		PlanVersion:            t.PlanVersion,
		AppMinVersion:          t.AppMinVersion,
		RejectReason:           t.RejectReason,
		ApprovedAt:             t.ApprovedAt,
	}
}
