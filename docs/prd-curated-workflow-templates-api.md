# 一键同款精选模板市场 PRD

> 版本：v1.0  
> 适用范围：new-api 后端（Storyboard-Copilot 下游）  
> 基地址：`https://<your-domain>/api`  
> 语言：中文

---

## 1. 项目信息

- **Language**: 中文
- **Programming Language**: Go（new-api 后端：Gin + GORM）
- **Project Name**: `curated_workflow_templates_api`
- **原始需求**: Storyboard-Copilot 前端已实现 oneClickReplicate 功能，原先使用本地 `curatedTemplates.json` 维护精选模板数据；现需要在 new-api 后端提供 HTTP API 供数，替换本地静态文件，支持模板列表、模板详情、执行计划、分类列表等能力，并便于后续运营动态上下架模板。

---

## 2. 产品定义

### 2.1 Product Goals

1. **数据上收**: 将前端本地 `curatedTemplates.json` 迁移至 new-api 后端，由服务端统一维护模板数据，支持运营动态更新，无需重新发版。
2. **即用接口**: 提供 4 个稳定的下游读取接口，覆盖列表、详情、执行计划、分类，并保持与前端现有 camelCase 数据模型兼容。
3. **低门槛集成**: 读取接口默认公开访问，减少前端鉴权改造成本；后台管理接口复用现有 `AdminAuth` 权限体系。

### 2.2 User Stories

- As a Storyboard-Copilot 用户，我希望在首页看到后台配置的精选模板列表，以便快速选择并一键复刻同款工作流。
- As a 运营人员，我希望在 new-api 管理后台增删改模板和分类，并控制模板是否上线，以便及时调整推荐内容。
- As a 前端开发者，我希望模板数据与分类列表由后端统一返回，以便替换本地 JSON 并减少版本依赖。
- As a 平台管理员，我希望模板数据模型与 new-api 现有表结构（GORM）兼容，以便复用数据库迁移和缓存机制。

---

## 3. 技术规范

### 3.1 架构与兼容性调整

| 原始 PRD 建议 | 适配 new-api 后的建议 | 说明 |
|-------------|----------------------|------|
| URL 前缀 `/api/v1` | **使用 `/api`** | new-api 已将 Web/管理接口统一挂在 `/api` 下；`/v1` 用于 Relay 转发，避免冲突。 |
| 鉴权建议 Bearer Token / API Key | **读取接口无需鉴权；管理接口复用 `middleware.AdminAuth()`** | 与 `/public/*` 和 banner 等运营接口保持一致。后续如需会员模板可再扩展。 |
| 数据模型独立定义 | **复用 GORM 表模型，JSON 字段用 `gorm:"type:json"` 或 text 存储** | 与 `skill.go`、`marketing_banner.go` 等模型保持一致。 |
| 版本号 v1 | 通过 URL 或响应字段暂不显式标识；如未来升级，再引入 `/api/v2/curated/...` | 保持简单，优先可用性。 |

### 3.2 需求池

#### P0（必须完成）

| 编号 | 需求 | 范围 | 验收标准 |
|------|------|------|---------|
| P0-1 | 模板列表接口 | `GET /api/curated/templates` | 支持 category、keyword、page、pageSize、sortBy、includeDetails 参数；返回分页列表（默认 pageSize=20）；默认返回精简字段，不包含完整 `executionPlan`。 |
| P0-2 | 模板详情接口 | `GET /api/curated/templates/:id` | 返回完整模板数据，包含 `inputSlots`、`params`、`executionPlan`。 |
| P0-3 | 执行计划接口 | `GET /api/curated/templates/:id/execution-plan` | 返回 `AgentTaskPlan` 完整结构，与前端 camelCase 字段名一致。 |
| P0-4 | 分类列表接口 | `GET /api/curated/categories` | 返回可用分类 key、名称、排序、是否启用；前端默认 `all` 用于筛选。 |
| P0-5 | 数据库模型与初始化 | `model/curated_template.go` + `model/curated_category.go` | 创建 `curated_templates`、`curated_categories` 表；支持软删除（`deleted_at`）或状态字段；提供按 `template_id` 查询。 |
| P0-6 | 后台管理接口 CRUD | `/api/admin/curated/templates`、`/api/admin/curated/categories` | 复用 `middleware.AdminAuth()`，支持模板的创建、更新、删除、启用/禁用、排序；分类的创建、更新、删除、排序。 |
| P0-7 | 媒体 URL 规范 | 模板与分类数据 | 封面图、预览视频 URL 统一为绝对地址（CDN 或 R2/S3）；分类图标也可使用 URL 或图标名。 |

#### P1（应该完成）

| 编号 | 需求 | 范围 | 验收标准 |
|------|------|------|---------|
| P1-1 | 服务端缓存 | Memory Cache / Redis | 列表与分类数据缓存 5-15 分钟，详情缓存可更短；更新接口清除缓存。 |
| P1-2 | 搜索与筛选增强 | 列表接口 | `keyword` 支持对 `title`、`description`、`prompt` 模糊搜索；`category` 支持多个值（逗号分隔）。 |
| P1-3 | 排序规则扩展 | 列表接口 | `sortBy` 支持 `hot`、`newest`、`price_asc`、`price_desc`，默认 `hot`。 |
| P1-4 | 默认值字段 | 模板模型 | `defaultMediaUrl`、`defaultMediaFileName` 支持用户上传前的占位示例。 |

#### P2（可选）

| 编号 | 需求 | 范围 | 验收标准 |
|------|------|------|---------|
| P2-1 | 增量更新机制 | 列表 / 分类接口 | 支持 `If-None-Match` / `If-Modified-Since` 或返回 `lastUpdatedAt` 字段，前端可基于时间戳增量刷新。 |
| P2-2 | 动态执行计划 | 执行计划接口 | 根据用户输入参数动态替换占位符后生成执行计划，而非直接返回静态 JSON。 |
| P2-3 | 会员可见模板 | 模板模型 | 增加 `visibility` 字段（public / members-only），读取接口结合 `TryUserAuth()` 与订阅状态过滤。 |
| P2-4 | 多语言模板 | 模板与分类数据 | 支持 `titleI18n`、`descriptionI18n` 等字段，按用户语言返回。 |

---

## 4. UI / 交互设计稿（API 端点列表）

### 4.1 下游读取接口（公开访问）

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/curated/templates` | 无 | 获取模板列表（分页、筛选、排序） |
| GET | `/api/curated/templates/:id` | 无 | 获取模板详情 |
| GET | `/api/curated/templates/:id/execution-plan` | 无 | 获取模板执行计划 |
| GET | `/api/curated/categories` | 无 | 获取分类列表 |

### 4.2 管理后台接口（AdminAuth）

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/admin/curated/templates` | AdminAuth | 获取全部模板（管理视图） |
| POST | `/api/admin/curated/templates` | AdminAuth | 创建模板 |
| PUT | `/api/admin/curated/templates/:id` | AdminAuth | 更新模板 |
| DELETE | `/api/admin/curated/templates/:id` | AdminAuth | 删除/禁用模板 |
| PATCH | `/api/admin/curated/templates/:id/status` | AdminAuth | 启用/禁用模板 |
| GET | `/api/admin/curated/categories` | AdminAuth | 获取全部分类 |
| POST | `/api/admin/curated/categories` | AdminAuth | 创建分类 |
| PUT | `/api/admin/curated/categories/:id` | AdminAuth | 更新分类 |
| DELETE | `/api/admin/curated/categories/:id` | AdminAuth | 删除分类 |

### 4.3 接口参数与响应示例

#### 4.3.1 模板列表：`GET /api/curated/templates`

**请求参数**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| category | string | 否 | all | 分类 key，如 `hot`、`commercial`，`all` 表示全部 |
| keyword | string | 否 | - | 模糊搜索关键词 |
| page | int | 否 | 1 | 页码 |
| pageSize | int | 否 | 20 | 每页数量，最大 100 |
| sortBy | string | 否 | hot | hot / newest / price_asc / price_desc |
| includeDetails | bool | 否 | false | 是否返回完整执行计划（默认精简） |

**响应示例**

```json
{
  "success": true,
  "message": "",
  "data": {
    "total": 128,
    "page": 1,
    "pageSize": 20,
    "list": [
      {
        "id": "template-001",
        "title": "周星驰经典喜剧复刻",
        "category": "zhouxingchi",
        "coverImageUrl": "https://cdn.example.com/covers/zhouxingchi.jpg",
        "previewMediaUrl": "https://cdn.example.com/previews/zhouxingchi.mp4",
        "description": "一键复刻周星驰式无厘头喜剧短视频",
        "estimatedPrice": 0.15,
        "inputSlots": [
          {
            "id": "script",
            "type": "text",
            "label": "输入脚本或创意",
            "required": true,
            "token": "{{script}}"
          }
        ],
        "params": {
          "resolution": "1080p",
          "duration": 15,
          "aspectRatio": "9:16",
          "featureEnabled": true,
          "model": "kling"
        }
      }
    ]
  }
}
```

#### 4.3.2 模板详情：`GET /api/curated/templates/:id`

返回与列表同结构，但包含完整的 `prompt`、`inputSlots`、`params`、`executionPlan`。`id` 为 `template_id`（字符串唯一标识）。

#### 4.3.3 执行计划：`GET /api/curated/templates/:id/execution-plan`

```json
{
  "success": true,
  "message": "",
  "data": {
    "summary": "周星驰风格视频生成",
    "templateVersion": "1.0.0",
    "templateId": "template-001",
    "templateParams": { "resolution": "1080p", "aspectRatio": "9:16" },
    "executionMode": "batch",
    "batchSize": 1,
    "groupData": { "nodes": [] },
    "steps": [
      {
        "id": "step-1",
        "description": "生成角色草图",
        "action": "generate",
        "nodeType": "imageGenNode",
        "nodeData": { ... },
        "connectFrom": [],
        "note": "",
        "waitForUser": false,
        "editableFields": [],
        "skillId": "",
        "skillOptions": [],
        "skillSelectedModels": {},
        "imageCount": 1,
        "autoExecute": true
      }
    ]
  }
}
```

#### 4.3.4 分类列表：`GET /api/curated/categories`

```json
{
  "success": true,
  "message": "",
  "data": {
    "categories": [
      { "key": "all", "name": "全部", "sortOrder": 0, "enabled": true },
      { "key": "zhouxingchi", "name": "周星驰", "sortOrder": 10, "enabled": true },
      { "key": "hot", "name": "热门", "sortOrder": 20, "enabled": true }
    ]
  }
}
```

> 说明：`all` 分类由后端返回，便于前端统一处理；前端仍保留 `all` 为默认筛选值。

### 4.4 数据模型建议（GORM）

```go
// CuratedTemplate 一键同款精选模板
type CuratedTemplate struct {
	Id              int    `json:"-" gorm:"primaryKey;autoIncrement"`
	TemplateId      string `json:"id" gorm:"column:template_id;size:64;uniqueIndex"`
	Title           string `json:"title" gorm:"not null"`
	Category        string `json:"category" gorm:"size:64;index"`
	CoverImageUrl   string `json:"coverImageUrl" gorm:"column:cover_image_url"`
	PreviewMediaUrl string `json:"previewMediaUrl" gorm:"column:preview_media_url"`
	Description     string `json:"description" gorm:"type:text"`
	Prompt          string `json:"prompt" gorm:"type:text"`
	InputSlots      string `json:"-" gorm:"column:input_slots;type:json"`
	Params          string `json:"-" gorm:"column:params;type:json"`
	EstimatedPrice  float64 `json:"estimatedPrice" gorm:"column:estimated_price"`
	ExecutionPlan   string `json:"-" gorm:"column:execution_plan;type:json"`
	SortOrder       int     `json:"sortOrder" gorm:"column:sort_order;default:0;index"`
	Enabled         bool    `json:"enabled" gorm:"default:true"`
	HotScore        int     `json:"hotScore" gorm:"column:hot_score;default:0"`
	CreatedAt       int64   `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       int64   `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (CuratedTemplate) TableName() string {
	return "curated_templates"
}

// CuratedCategory 模板分类
type CuratedCategory struct {
	Id        int    `json:"-" gorm:"primaryKey;autoIncrement"`
	Key       string `json:"key" gorm:"size:64;uniqueIndex"`
	Name      string `json:"name" gorm:"not null"`
	IconUrl   string `json:"iconUrl" gorm:"column:icon_url"`
	SortOrder int    `json:"sortOrder" gorm:"column:sort_order;default:0;index"`
	Enabled   bool   `json:"enabled" gorm:"default:true"`
	CreatedAt int64  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt int64  `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (CuratedCategory) TableName() string {
	return "curated_categories"
}
```

序列化层可在 controller 中将 JSON 字符串字段解析为对象，保证前端收到 camelCase 结构。`nodeData` 保持原结构直接返回 JSON 对象。

### 4.5 错误码规范

复用 new-api 现有错误响应格式：

```json
{
  "success": false,
  "message": "错误描述",
  "data": null
}
```

HTTP 状态码与业务码映射：

| HTTP 状态码 | 业务码 | 说明 |
|------------|--------|------|
| 200 | 0 | 成功 |
| 400 | 40001 | 参数错误（如 pageSize 超限） |
| 400 | 40002 | 分类不存在 |
| 400 | 40003 | 模板数据 JSON 解析失败 |
| 401 | 40101 | 管理接口 Token 无效或过期 |
| 403 | 40301 | 管理接口权限不足 |
| 404 | 40401 | 模板不存在 |
| 429 | 42901 | 请求频率超限 |
| 500 | 50000 | 服务端内部错误 |

---

## 5. 待确认问题与建议方案

基于“尽快上线、优先保证可用性”的目标，给出建议方案：

| 序号 | 问题 | 建议方案 |
|------|------|---------|
| 1 | 模板封面图和预览视频是否由上游 CDN 提供，还是前端本地兜底？ | **由 CDN / 对象存储提供绝对 URL**。建议复用 new-api 已集成的 R2/S3 上传能力，后台创建模板时上传封面/预览视频并返回 URL；前端不再兜底，避免包体积过大。 |
| 2 | 是否支持分页？ | **支持分页**。`GET /api/curated/templates` 必须实现 `page` / `pageSize` 参数，默认 20 条、最大 100 条，返回 `total` 便于前端分页。 |
| 3 | 是否支持缓存和增量更新？ | **V1 先支持服务端缓存**（P1），列表与分类缓存 5-15 分钟；**增量更新（P2）**通过返回 `updatedAt` 或 `lastUpdatedAt` 字段，由前端自行决定刷新时机，避免过度设计。 |
| 4 | 分类列表是否固定由前端维护，还是也由上游返回？ | **由上游返回**。`GET /api/curated/categories` 返回分类 key、名称、排序、启用状态；前端保留 `all` 为默认筛选值，但列表来源切换为后端。 |
| 5 | 是否需要用户鉴权？是否区分会员可见模板？ | **V1 不鉴权、不区分会员**。读取接口全部公开，降低前端接入成本。在模板模型中预留 `visibility`（public / members-only）字段，P2 再扩展会员鉴权。 |
| 6 | 执行计划是否支持动态生成？ | **V1 静态返回**。`executionPlan` 以 JSON 形式存储在数据库，按模板 ID 直接返回。动态生成（根据用户输入替换变量、重新编排步骤）作为 P2 优化项。 |
| 7 | 列表接口是否返回完整模板数据？ | **列表返回精简数据**。默认不包含完整 `executionPlan`，通过 `includeDetails=false` 控制；详情与执行计划接口返回完整数据。保证列表性能，避免传输大量 JSON。 |

---

## 6. 附录：与前端数据模型的兼容性说明

1. **字段命名**：API 响应统一使用 camelCase，与前端现有类型 `CuratedWorkflowTemplate`、`CuratedInputSlot`、`CuratedTemplateParams`、`AgentTaskPlan`、`AgentStep` 保持一致。
2. **媒体 URL**：`coverImageUrl`、`previewMediaUrl` 支持绝对 URL 或以 `/` 开头的相对路径；V1 推荐绝对 URL。
3. **JSON 字段**：`inputSlots`、`params`、`executionPlan` 在数据库中以 JSON 文本存储，controller 中序列化后返回对象，不直接暴露原始字符串。
4. **扩展字段**：预留 `hotScore`、`sortOrder`、`enabled` 等运营字段，便于后台调整排序与上下线。

---

*文档版本：v1.0*  
*作者：产品经理（software-product-manager）*  
*日期：2026-07-23*
