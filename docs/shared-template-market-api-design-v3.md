# 工作流模板云端分享 — 服务端设计 v3（优化稿）

> **修订日期**：2026-08-01
> **修订说明**：基于 v2 方案评估，修正跨数据库兼容性、路由冲突、状态机等 P0 问题，对齐 new-api 现有架构规范。

---

## ��计决策速览

| 决策点 | v2 方案 | v3 修正 | 理由 |
|--------|---------|---------|------|
| DB 类型 | PostgreSQL 专有（JSONB/BIGSERIAL/TIMESTAMPTZ/TSVECTOR） | GORM 通用模型（TEXT/int64） | new-api 须兼容 SQLite/MySQL/PG |
| 路由前缀 | `/api/templates` | `/api/shared-templates` | 避免与已实现的 `/api/curated/templates` 冲突 |
| 状态机 | pending → approved → published | pending → approved（即发布）/ rejected | 简化运营，一步完成审核发布 |
| 鉴权 | `role == 'admin'` | `middleware.AdminAuth()` | 复用已封装的中间件 |
| 响应格式 | `{success, data}` | `{success, message, data}` | 与 `common.ApiSuccess()` 保持一致 |
| `/mine` 接口 | 服务端摘要遗漏 | 补充 `GET /api/shared-templates/mine` | 完整方案中前端依赖此接口 |
| 素材包下载 | 公开直链 | 加 `/download` 签名 URL | 防滥用 |
| 上传限制 | 无 | 加频率限制 + 大小校验 | 防刷库 |

---

## 1. 数据库设计（GORM 兼容）

### 1.1 模板主表 `shared_templates`

```go
// model/shared_template.go
type SharedTemplate struct {
    Id             int             `json:"-" gorm:"primaryKey;autoIncrement"`
    TemplateId     string          `json:"id" gorm:"column:template_id;size:32;uniqueIndex"`
    Name           string          `json:"name" gorm:"size:200;not null"`
    Description    string          `json:"description" gorm:"type:text"`
    Category       string          `json:"category" gorm:"size:32;not null;index:idx_status_category"`
    AuthorId       int             `json:"authorId" gorm:"column:author_id;not null;index"`
    AuthorName     string          `json:"authorName" gorm:"column:author_name;size:100;not null"`
    Status         string          `json:"status" gorm:"size:20;not null;default:'pending';index:idx_status_category"`
    // 状态: pending / approved / rejected
    // approved 即已发布，市场可见
    RejectReason   string          `json:"rejectReason,omitempty" gorm:"column:reject_reason;type:text"`
    PlanJson       string          `json:"planJson" gorm:"column:plan_json;type:longtext;not null"`   // AgentTaskPlan JSON
    PlanVersion    int             `json:"planVersion" gorm:"column:plan_version;default:3"`
    AppMinVersion  string          `json:"appMinVersion,omitempty" gorm:"column:app_min_version;size:20"`
    ManifestJson   string          `json:"manifestJson,omitempty" gorm:"column:manifest_json;type:longtext"` // 素材清单 JSON
    AssetCount     int             `json:"assetCount" gorm:"column:asset_count;default:0"`
    ImageCount     int             `json:"imageCount" gorm:"column:image_count;default:0"`
    VideoCount     int             `json:"videoCount" gorm:"column:video_count;default:0"`
    TotalSize      int64           `json:"totalSize" gorm:"column:total_size;default:0"`
    HasAssets      bool            `json:"hasAssets" gorm:"column:has_assets;default:false"`
    ThumbnailUrl   string          `json:"thumbnailUrl,omitempty" gorm:"column:thumbnail_url;size:500"`
    UseCount       int             `json:"useCount" gorm:"column:use_count;default:0;index"`
    CreatedAt      int64           `json:"createdAt" gorm:"column:created_at;autoCreateTime;index"`
    UpdatedAt      int64           `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
    ApprovedAt     int64           `json:"approvedAt,omitempty" gorm:"column:approved_at;default:0"` // 审核通过时间
}

func (SharedTemplate) TableName() string {
    return "shared_templates"
}
```

**关键说明**：
- `PlanJson` 用 `longtext`（兼容 MySQL），SQLite/PG 会映射为 `TEXT`。
- `ManifestJson` 同理。不在 DB 里存 JSONB —— 跨 DB 用 `TEXT` 最安全。
- 时间全部用 `int64` Unix 时间戳，与现有 `curated_templates` 保持一致。
- `ApprovedAt` 记录发布时间，`sort=popular` 时可用于排序。
- `AssetCount / ImageCount / VideoCount / TotalSize / HasAssets` 在创建时从 manifest 解析并持久化，避免每次请求都解压 ZIP 统计。

### 1.2 素材表 `shared_template_assets`

```go
type SharedTemplateAsset struct {
    Id           int    `json:"-" gorm:"primaryKey;autoIncrement"`
    TemplateId   string `json:"templateId" gorm:"column:template_id;size:32;not null;index"`
    AssetType    string `json:"assetType" gorm:"column:asset_type;size:10;not null"` // image / video
    FileHash     string `json:"fileHash" gorm:"column:file_hash;size:64;not null;index"` // SHA-256
    OriginalPath string `json:"originalPath,omitempty" gorm:"column:original_path;size:500"`
    StorageKey   string `json:"storageKey" gorm:"column:storage_key;size:500;not null"`
    SizeBytes    int64  `json:"sizeBytes" gorm:"column:size_bytes;not null"`
    MimeType     string `json:"mimeType,omitempty" gorm:"column:mime_type;size:100"`
    CreatedAt    int64  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
}

func (SharedTemplateAsset) TableName() string {
    return "shared_template_assets"
}
```

> **注意**：外键约束不在 DDL 中声明，改为应用层保证。SQLite 外键默认关闭且容易出兼容性问题。删除模板时在 service 层级联清理。

### 1.3 审核记录表 `shared_template_audit_logs`

```go
type SharedTemplateAuditLog struct {
    Id         int    `json:"-" gorm:"primaryKey;autoIncrement"`
    TemplateId string `json:"templateId" gorm:"column:template_id;size:32;not null;index"`
    AdminId    int    `json:"adminId" gorm:"column:admin_id;not null"`
    AdminName  string `json:"adminName" gorm:"column:admin_name;size:128"`
    Action     string `json:"action" gorm:"size:20;not null"` // approve / reject
    Reason     string `json:"reason,omitempty" gorm:"type:text"`
    CreatedAt  int64  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
}

func (SharedTemplateAuditLog) TableName() string {
    return "shared_template_audit_logs"
}
```

### 1.4 使用记录表 `shared_template_uses`

```go
type SharedTemplateUse struct {
    Id         int    `json:"-" gorm:"primaryKey;autoIncrement"`
    TemplateId string `json:"templateId" gorm:"column:template_id;size:32;not null;index"`
    UserId     int    `json:"userId" gorm:"column:user_id;not null;uniqueIndex:idx_template_user"`
    UsedAt     int64  `json:"usedAt" gorm:"column:used_at;autoCreateTime"`
}

func (SharedTemplateUse) TableName() string {
    return "shared_template_uses"
}
```

> `uniqueIndex:idx_template_user` 确保同一用户重复使用不重复计数（UNIQUE 约束）。

---

## 2. API 接口设计

### 2.1 路由总览

```
# ─── 用户接口 ───
POST   /api/shared-templates                  # 分享模板           TokenAuth
GET    /api/shared-templates                  # 公开模板列表        TryUserAuth（可选）
GET    /api/shared-templates/:id              # 模板详情            TryUserAuth
GET    /api/shared-templates/:id/package      # 下载素材包         公开（签名 URL）
GET    /api/shared-templates/mine             # 我的分享            TokenAuth
DELETE /api/shared-templates/:id              # 删除模板            TokenAuth（仅作者）
POST   /api/shared-templates/:id/use          # 记录使用            TokenAuth

# ─── 管理员接口 ───
GET    /api/admin/shared-templates/pending    # 待审核列表          AdminAuth
POST   /api/admin/shared-templates/:id/audit  # 审核模板            AdminAuth
GET    /api/admin/shared-templates/all        # 全部模板（管理视图） AdminAuth
```

> **路由注册顺序**：`/mine` 必须注册在 `/:id` 之前，否则 Gin 会把 `mine` 当作 `:id` 参数。

### 2.2 接口详细设计

---

#### 2.2.1 分享模板

```
POST /api/shared-templates
Content-Type: multipart/form-data
Authorization: Bearer <token>
```

| 字段 | 类型 | 必填 | 约束 |
|------|------|------|------|
| name | text | 是 | ≤ 200 字符 |
| description | text | 否 | ≤ 500 字符 |
| category | text | 是 | ecommerce / portrait / landscape / commercial / creative / other |
| planJson | text | 是 | 合法 JSON，steps 非空，含 templateVersion 字段 |
| planVersion | text | 否 | 默认 3 |
| appMinVersion | text | 否 | 最低兼容版本 |
| thumbnail | file | 否 | jpg/png，≤ 2MB |
| package | file | 否 | ZIP，≤ 100MB |

**处理流程**：

```
1. 校验 JWT → 获取 userId、userName
2. 频率限制：同一用户每 10 秒最多 1 次提交
3. 校验必填字段
4. 解析 planJson 为合法 JSON，检查 steps 非空 && templateVersion >= 3
5. 敏感信息检测：扫描 planJson 中的 API Key、手机号、邮箱正则
6. [如有 package]：
   a. 校验 ZIP 完整性
   b. 解压读取 manifest.json，校验 version == 2
   c. 遍历 assets/，拒绝可执行/脚本文件
   d. 计算 SHA-256，去重检查
   e. 上传至对象存储
   f. 持久化 assetInfo（assetCount/imageCount/videoCount/totalSize）
7. [有 thumbnail]：压缩 400x300 JPEG，存对象存储
8. [无 thumbnail 有 package]：异步从素材第一张图生成缩略图（首次先设空）
9. 生成 UUID → 写入 DB，status = 'pending'
10. 异步内容机审（涉黄涉政暴恐）
11. 返回 { id, status, createdAt }
```

**响应**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": "a1b2c3d4e5f6",
    "status": "pending",
    "createdAt": 1756348800
  }
}
```

**错误**：

| 场景 | HTTP | 错误信息 |
|------|------|----------|
| 参数缺失 | 400 | `Missing required field: {name/category/planJson}` |
| planJson 不是合法 JSON | 400 | `planJson is not valid JSON` |
| planJson.steps 为空 | 400 | `Template has no steps` |
| templateVersion < 3 | 400 | `Template version too old, minimum required: 3` |
| 文件过大 | 413 | `File too large: {thumbnail/package}` |
| 含可执行文件 | 400 | `Executable file not allowed: {filename}` |
| 频率限制 | 429 | `Too many requests, please try again later` |
| 敏感信息 | 400 | `planJson contains sensitive data: {type}` |

---

#### 2.2.2 获取模板列表

```
GET /api/shared-templates?page=1&pageSize=20&category=ecommerce&keyword=穿搭&sort=newest
Authorization: Bearer <token> (可选)
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| pageSize | int | 20 | 每页数量（最大 50） |
| category | string | - | 分类筛选，不传 = 全部 |
| keyword | string | - | 模糊匹配 name + description |
| sort | string | newest | newest / popular |

**逻辑**：只返回 `status = 'approved'` 的模板。

**响应**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "total": 156,
    "page": 1,
    "pageSize": 20,
    "list": [
      {
        "id": "a1b2c3d4e5f6",
        "name": "电商穿搭模板",
        "description": "适合电商产品展示的穿搭视频模板",
        "thumbnailUrl": "https://cdn.example.com/templates/a1b2c3d4e5f6/thumbnail.jpg",
        "category": "ecommerce",
        "authorId": 1001,
        "authorName": "大勇",
        "status": "approved",
        "assetInfo": {
          "hasAssets": true,
          "assetCount": 5,
          "totalSize": 1048576,
          "imageCount": 3,
          "videoCount": 2
        },
        "useCount": 128,
        "createdAt": 1756348800,
        "updatedAt": 1756435200
      }
    ]
  }
}
```

> **注意**：列表接口不返回 `planJson`，减少响应体积。详情接口才返回完整 plan。

---

#### 2.2.3 获取模板详情

```
GET /api/shared-templates/:id
Authorization: Bearer <token> (可选)
```

**逻辑**：只返回 `status = 'approved'` 的模板。作者本人可通过 `/mine` 查看自己所有状态。

**响应**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": "a1b2c3d4e5f6",
    "name": "电商穿搭模板",
    "description": "...",
    "thumbnailUrl": "https://cdn.example.com/templates/a1b2c3d4e5f6/thumbnail.jpg",
    "category": "ecommerce",
    "authorId": 1001,
    "authorName": "大勇",
    "status": "approved",
    "planJson": "{...}",
    "planVersion": 3,
    "appMinVersion": "1.2.0",
    "assetInfo": {
      "hasAssets": true,
      "assetCount": 5,
      "totalSize": 1048576,
      "imageCount": 3,
      "videoCount": 2
    },
    "useCount": 128,
    "createdAt": 1756348800,
    "updatedAt": 1756435200,
    "approvedAt": 1756435200
  }
}
```

---

#### 2.2.4 我的分享

```
GET /api/shared-templates/mine?page=1&pageSize=20
Authorization: Bearer <token>
```

**逻辑**：返回当前用户的所有模板（含 pending / approved / rejected 所有状态）。

**响应**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "total": 5,
    "page": 1,
    "pageSize": 20,
    "list": [
      {
        "id": "a1b2c3d4e5f6",
        "name": "电商穿搭模板",
        "status": "approved",
        "rejectReason": null,
        "category": "ecommerce",
        "useCount": 128,
        "createdAt": 1756348800,
        "updatedAt": 1756435200
      },
      {
        "id": "b2c3d4e5f6a7",
        "name": "人像写真模板",
        "status": "rejected",
        "rejectReason": "含版权素材",
        "category": "portrait",
        "useCount": 0,
        "createdAt": 1756185600,
        "updatedAt": 1756272000
      }
    ]
  }
}
```

---

#### 2.2.5 下载素材包

```
GET /api/shared-templates/:id/package
```

**逻辑**：
1. 查询模板是否有素材（`hasAssets == true`）
2. 若无素材 → 204 No Content
3. 若有素材 → 生成临时签名 URL（有效期 15 分钟），302 重定向或流式代理返回
4. **不暴露存储直链**，防止被当免费 CDN

**响应**：
- `200` → `Content-Type: application/zip`
- `204` → 无素材

---

#### 2.2.6 删除模板

```
DELETE /api/shared-templates/:id
Authorization: Bearer <token>
```

**权限**：`authorId == userId`

**处理**：软删除（GORM `DeletedAt`）+ 异步清理对象存储。

---

#### 2.2.7 记录使用

```
POST /api/shared-templates/:id/use
Authorization: Bearer <token>
Content-Type: application/json

Body: {}
```

**逻辑**：
1. 插入 `shared_template_uses`（UNIQUE 防重复计数）
2. `UPDATE shared_templates SET use_count = use_count + 1`

---

#### 2.2.8 管理员 — 待审核列表

```
GET /api/admin/shared-templates/pending?page=1&pageSize=20
Authorization: Bearer <admin_token>
Middleware: AdminAuth
```

返回所有 `status = 'pending'` 的模板，包含完整 `planJson`。

---

#### 2.2.9 管理员 — 审核模板

```
POST /api/admin/shared-templates/:id/audit
Authorization: Bearer <admin_token>
Middleware: AdminAuth
Content-Type: application/json
```

**通过**：

```json
{ "action": "approve" }
```

**拒绝**：

```json
{
  "action": "reject",
  "reason": "含版权素材，请修改后重新提交"
}
```

**处理流程**：

```
1. AdminAuth 校验
2. 验证模板存在且 status = 'pending'
3. 验证 action 为 approve/reject
4. reject 时 reason 必填（≤ 500 字符）
5. 更新模板状态：
   - approve: status = 'approved', approved_at = now
   - reject: status = 'rejected', reject_reason = reason
6. 写入 shared_template_audit_logs
7. approve 时可选发送通知给作者
```

**响应**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": "a1b2c3d4e5f6",
    "status": "approved",
    "updatedAt": 1756435200
  }
}
```

---

#### 2.2.10 管理员 — 全部模板

```
GET /api/admin/shared-templates/all?page=1&pageSize=20&status=pending
Authorization: Bearer <admin_token>
Middleware: AdminAuth
```

管理视图，支持按 status 筛选，返回所有状态的模板。

---

## 3. 状态机

```
                  ┌─────────┐
                  │ pending │ ← 用户提交
                  └────┬────┘
                       │
            ┌──────────┼──────────┐
            ▼                     ▼
      ┌──────────┐          ┌──────────┐
      │ approved │          │ rejected │
      └──────────┘          └──────────┘
      (即发布上线)            (含驳回原因)

公开接口只暴露 approved；
/mine 可看 pending/approved/rejected；
被 rejected 后可重新提交（创建新记录，旧记录保留）。
```

> **简化说明**：去掉了 v2 方案中 `approved` → `published` 的两阶段发布。审核通过即上线。如需"定时发布"等能力，后续再加 `published_at` 字段和定时任务。

---

## 4. 文件存储

### 4.1 存储布局

```
{bucket}/
  shared-templates/
    {templateId}/
      package.zip      # 上传的原始素材包
      thumbnail.jpg    # 缩略图
      assets/          # 解压后的独立文件（可选，便于 CDN）
        images/
        videos/
```

### 4.2 存储策略

| 文件 | 访问方式 | 策略 |
|------|----------|------|
| `package.zip` | 仅服务端内部 + 签名 URL 代理 | 不公开，通过 `/package` 接口代理 |
| `thumbnail.jpg` | CDN 公开直链 | 公开访问 |
| `assets/` | 可选，CDN 直链 | 如需要客户端直接引用素材才开启 |

### 4.3 存储接口抽象

```go
// service/storage.go (已有或新增)
type ObjectStorage interface {
    Upload(key string, data []byte, contentType string) error
    GetSignedURL(key string, expire time.Duration) (string, error)
    Delete(key string) error
}
```

根据配置自动选择 S3 / 阿里云 OSS / 本地文件存储。

---

## 5. 安全与审核链路

```
用户上传
    │
    ▼
┌─────────────────┐
│ 1. 文件过滤      │  拒绝.exe/.bat/.sh/.js/.dll 等可执行/脚本文件
└────────┬────────┘
         ▼
┌─────────────────┐
│ 2. 敏感信息扫描   │  检测 planJson 中的 API Key / 手机号 / 邮箱正则
└────────┬────────┘
         ▼
┌─────────────────┐
│ 3. 入库 (pending) │
└────────┬────────┘
         ▼
┌─────────────────┐
│ 4. 异步内容机审    │  提取图片/视频关键帧 → 内容安全 API（涉黄涉政暴恐）
└────────┬────────┘
         ▼
┌─────────────────┐
│ 5. 管理员人审      │  审阅 plan 结构、缩略图、素材
└────────┬────────┘
         ▼
   approved / rejected
```

---

## 6. 响应 DTO 定义

```go
// dto/shared_template.go

type AssetInfo struct {
    HasAssets  bool  `json:"hasAssets"`
    AssetCount int   `json:"assetCount"`
    TotalSize  int64 `json:"totalSize"`
    ImageCount int   `json:"imageCount"`
    VideoCount int   `json:"videoCount"`
}

type SharedTemplateListItem struct {
    Id           string     `json:"id"`
    Name         string     `json:"name"`
    Description  string     `json:"description,omitempty"`
    ThumbnailUrl string     `json:"thumbnailUrl,omitempty"`
    Category     string     `json:"category"`
    AuthorId     int        `json:"authorId"`
    AuthorName   string     `json:"authorName"`
    Status       string     `json:"status"`
    AssetInfo    *AssetInfo `json:"assetInfo,omitempty"`
    UseCount     int        `json:"useCount"`
    CreatedAt    int64      `json:"createdAt"`
    UpdatedAt    int64      `json:"updatedAt"`
}

type SharedTemplateDetail struct {
    SharedTemplateListItem
    PlanJson      string `json:"planJson"`
    PlanVersion   int    `json:"planVersion"`
    AppMinVersion string `json:"appMinVersion,omitempty"`
    RejectReason  string `json:"rejectReason,omitempty"`
    ApprovedAt    int64  `json:"approvedAt,omitempty"`
}

type PaginatedList struct {
    Total    int64       `json:"total"`
    Page     int         `json:"page"`
    PageSize int         `json:"pageSize"`
    List     interface{} `json:"list"`
}
```

---

## 7. 开发清单

### 7.1 文件清单

```
model/shared_template.go          # GORM 模型 + 查询方法
model/main.go                     # AutoMigrate 注册
dto/shared_template.go            # 请求/响应 DTO
service/shared_template.go        # 业务逻辑
service/shared_template_audit.go  # 审核逻辑
controller/shared_template.go     # 用户接口 handler
controller/shared_template_admin.go # 管理接口 handler
router/api-router.go              # 路由注册
```

### 7.2 与现有 curated_templates 的关系

两者**独立运行**，不互相依赖。但可共享：
- 公共分页参数解析（`common.GetPageQuery`）
- 公共响应封装（`common.ApiSuccess / ApiErrorMsg`）
- 存储抽象层

路由不冲突：
```
/api/curated/templates                          # 精选模板（运营配置）
/api/shared-templates                           # UGC 模板市场（新）
/api/admin/curated/templates                    # 精选模板管理
/api/admin/shared-templates                     # UGC 模板管理
```

### 7.3 MVP 阶段建议

先做**纯文本模板分享**（无素材包），验证审核流程：

| 任务 | 估时 |
|------|------|
| model + AutoMigrate | 0.5 天 |
| DTO 定义 | 0.5 天 |
| 分享模板（POST）+ 详情（GET） | 1 天 |
| 列表 + 分页 + 筛选 + 排序 | 1 天 |
| /mine 我的分享 | 0.5 天 |
| 删除模板 | 0.5 天 |
| 管理审核 + 审计日志 | 1 天 |
| 路由注册 + 测试 | 0.5 天 |
| **合计** | **5.5 天** |

---

## 8. 与方案 v2 的差异对照表

| 条目 | v2 | v3 | 原因 |
|------|----|----|------|
| 表名 | `workflow_templates` | `shared_templates` | 与 curated 区分更清晰 |
| 主键 | `VARCHAR(32)` 单主键 | `BIGINT 自增` + `template_id` 业务 ID | GORM 惯例，性能更好 |
| JSON 字段 | `JSONB` | `TEXT` / `longtext` | SQLite/PG/MySQL 通用 |
| 时间类型 | `TIMESTAMPTZ` | `int64` | 跨 DB 一致 |
| 状态 | pending/approved/rejected/published | pending/approved/rejected | 简化，approved 即发布 |
| 审核 API | 无 describe | `approve → status='approved', approved_at=now` | 一步完成 |
| `/mine` | 遗漏 | 补充，含分页 | 前端需要 |
| 管理员全量列表 | 无 | `GET /admin/.../all` | 运营需要 |
| 记录使用 | 无 | `POST /.../:id/use` | 统计使用次数 |
| 素材下载 | 直接返回 ZIP | 签名 URL + 302 重定向 | 防滥用 |
| 上传频率限制 | 无 | 每用户 10s 限 1 次 | 防刷 |
| 内容机审 | 只提概念 | 明确为异步步骤 | 落地方案 |
| 缩略图生成 | 服务端从 ZIP 提取 | 优先用上传的 thumbnail，无则异步生成 | 渐进实现 |
