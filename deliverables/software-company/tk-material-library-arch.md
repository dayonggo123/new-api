# TK 素材库架构设计

## 1. 实现方案

在 new-api 现有分层架构（router → controller → service → model）基础上，新增一个独立模块：

- **model**：`TKMaterial` 数据模型与数据库操作
- **service**：图片上传保存、Notion 导入、Notion HTTP 客户端
- **controller**：管理后台 API 与下游公开 API
- **router**：注册 `/api/admin/tk/materials/*` 与 `/api/public/tk/materials/*`
- **frontend**：`TKMaterialManagement` 后台管理页面

## 2. 新增/修改文件

### 后端

| 文件 | 说明 |
|------|------|
| `model/tk_material.go` | 数据模型与 CRUD/查询/随机方法 |
| `service/tk_material.go` | 上传处理、保存、Notion 导入逻辑 |
| `service/notion_client.go` | 轻量 Notion API HTTP 客户端 |
| `controller/tk_material.go` | 管理后台与下游 API Handler |
| `router/api-router.go` | 注册路由 |
| `model/main.go` | AutoMigrate 注册 `TKMaterial` |

### 前端

| 文件 | 说明 |
|------|------|
| `web/src/pages/TKMaterialManagement/index.jsx` | 素材库管理页面 |
| `web/src/App.jsx` | 注册 `/console/tk-material` 路由 |
| `web/src/components/layout/SiderBar.jsx` | 添加侧边栏菜单与路由映射 |
| `web/src/helpers/render.jsx` | 添加 `tk_material` 图标 |

## 3. 数据模型

```go
type TKMaterial struct {
    ID           int    `gorm:"primaryKey"`
    Category     string `gorm:"index;size:64"`
    URL          string `gorm:"type:text"`
    ThumbnailURL string `gorm:"type:text"`
    Filename     string
    FileType     string
    Size         int64
    Width        int
    Height       int
    Source       string `gorm:"index;size:32"` // upload / notion
    NotionPageID string `gorm:"index;size:128"`
    Status       int    `gorm:"default:1;index"`
    CreatedAt    int64
    UpdatedAt    int64
}
```

## 4. 接口定义

### 管理后台（AdminAuth）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/tk/materials` | 列表（category、keyword、status、page、page_size） |
| POST | `/api/admin/tk/materials` | 上传（multipart/form-data，category、files） |
| GET | `/api/admin/tk/materials/categories` | 返回所有分类 |
| GET | `/api/admin/tk/materials/stats` | 各分类统计 |
| DELETE | `/api/admin/tk/materials/:id` | 删除（软删除） |
| POST | `/api/admin/tk/materials/import/notion` | 触发 Notion 导入 |

### 下游公开

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/public/tk/materials` | 分页列表 |
| GET | `/api/public/tk/materials/:id` | 详情 |
| GET | `/api/public/tk/materials/random` | 随机 N 张（category、limit） |
| POST | `/api/public/tk/materials` | 上传（可选） |

## 5. 程序调用流程

### 上传流程

```
用户选择图片 → AdminUploadTKMaterial
  → UploadTKMaterialFiles（保存到 uploads/permanent/tk_materials/）
  → SaveTKMaterialFromUpload → DB
```

### Notion 导入流程

```
用户提交 Database ID → AdminImportTKMaterialsFromNotion
  → NewNotionClient.QueryDatabaseAll（分页拉取）
  → 遍历每页 Properties[category]
  → extractNotionImageURLs（files/url/rich_text）
  → 去重 → SaveTKMaterialFromNotion → DB
```

### 下游调用流程

```
客户端 → PublicListTKMaterials / PublicGetTKMaterial / PublicGetRandomTKMaterials
  → model 查询 → 返回 JSON
```

## 6. 任务列表（实现顺序）

1. 创建 `model/tk_material.go` 并注册迁移
2. 创建 `service/tk_material.go` 和 `service/notion_client.go`
3. 创建 `controller/tk_material.go`
4. 修改 `router/api-router.go` 注册路由
5. 创建前端页面 `TKMaterialManagement/index.jsx`
6. 修改 `App.jsx`、`SiderBar.jsx`、`render.jsx`
7. 编译后端、构建前端

## 7. 依赖包

- 后端：复用现有 Gin、GORM、google/uuid
- 前端：复用现有 Semi UI、lucide-react

## 8. 共享知识

- 文件上传复用 `uploads/permanent/` 目录，避免被清理任务删除
- 数据库兼容 SQLite/MySQL/PostgreSQL
- 随机排序使用 `RANDOM()`（SQLite/PostgreSQL）或 `RAND()`（MySQL）
- Notion 列名作为分类，支持 `files`、`url`、`rich_text` 三种图片来源
- 图片 URL 按 `category + url` 去重

## 9. 待明确事项

1. Notion Database ID 和 Token 来源（当前支持请求传入或环境变量）
2. 是否需要缩略图生成（当前 thumbnail_url = url）
3. 下游 API 是否需要 Token 鉴权（当前完全公开）
4. 是否需要导入任务进度持久化（当前单次同步返回结果）
