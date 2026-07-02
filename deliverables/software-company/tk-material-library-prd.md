# TK 素材库功能 PRD

## 1. 产品目标

在 new-api 后台新增「TK 素材库」模块，用于集中管理 TikTok/电商场景图片素材。支持：
- 后台上传、分类、管理素材
- 从 Notion「达人 IP 库」表格自动导入场景图片及分类
- 提供标准化下游 API 供客户端调用

## 2. 用户故事

| 角色 | 故事 | 优先级 |
|------|------|--------|
| 运营/管理员 | 上传本地图片到 TK 素材库，按场景分类 | P0 |
| 运营/管理员 | 在后台查看、搜索、筛选、删除素材 | P0 |
| 系统 | 自动从 Notion「达人 IP 库」导入指定列图片 | P0 |
| 下游客户端 | 通过 API 按场景分类获取素材列表 | P0 |
| 下游客户端 | 随机获取指定场景的 N 张图片 | P1 |

## 3. 需求池（P0/P1/P2）

### P0 必须完成

1. **后台管理页面**
   - 左侧菜单新增「TK 素材库」
   - 列表页：缩略图、分类标签、上传时间、操作（删除/复制 URL）
   - 支持按分类筛选、分页、关键词搜索
   - 支持单张/批量上传（拖拽或选择文件）
   - 上传时可选择分类

2. **数据模型**
   - 新增 `tk_materials` 表
   - 字段：id、category、url、thumbnail_url、filename、file_type、size、width、height、source、notion_page_id、status、created_at、updated_at

3. **Notion 导入**
   - 读取指定 Notion 数据库
   - 导入列：浴室、客厅、厨房、卧室、车库、院子、街景、健身房、车、机场、农村、公园、超市、仓库
   - 列名作为 `category`
   - 图片 URL 去重、保存
   - 支持手动触发导入 + 后台任务记录

4. **下游 API**
   - `GET /api/public/tk/materials` — 分页列表
   - `GET /api/public/tk/materials/random` — 按分类随机 N 张
   - `GET /api/public/tk/materials/:id` — 详情
   - `POST /api/public/tk/materials` — 上传（可选，下游用）

### P1 建议

- 上传图片压缩/生成缩略图
- 导入任务进度显示
- 批量删除
- 素材分类管理（增删改）

### P2 可选

- 图片 AI 打标
- 使用量统计
- 私有/公开权限控制

## 4. 数据模型

```go
type TKMaterial struct {
    ID           int    `gorm:"primaryKey"`
    Category     string // 分类：浴室、客厅、厨房、卧室、车库、院子、街景、健身房、车、机场、农村、公园、超市、仓库
    URL          string // 原图地址
    ThumbnailURL string // 缩略图地址
    Filename     string
    FileType     string
    Size         int64
    Width        int
    Height       int
    Source       string // upload / notion
    NotionPageID string // 来源 Notion page id
    Status       int    // 0 禁用 1 启用
    CreatedAt    int64
    UpdatedAt    int64
}
```

## 5. 接口设计

### 5.1 管理后台 API（需 admin 权限）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/tk/materials` | 列表（分页、分类筛选、关键词搜索） |
| POST | `/api/admin/tk/materials` | 单张/批量上传 |
| DELETE | `/api/admin/tk/materials/:id` | 删除素材 |
| POST | `/api/admin/tk/materials/import/notion` | 触发 Notion 导入 |
| GET | `/api/admin/tk/materials/import/logs` | 导入任务记录 |

### 5.2 下游 API（公开/需 key）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/public/tk/materials` | 分页列表 |
| GET | `/api/public/tk/materials/:id` | 详情 |
| GET | `/api/public/tk/materials/random` | 按分类随机 N 张 |

### 5.3 下游 API 参数示例

**GET /api/public/tk/materials**
```
Query:
  category=客厅
  page=1
  page_size=20
  keyword=
```

**GET /api/public/tk/materials/random**
```
Query:
  category=客厅
  limit=5
```

## 6. 待确认问题

1. Notion 数据库 ID 和访问方式是什么？（Integration Token 已配置？）
2. Notion 表格里的图片是 `files` 类型还是 `url` 文本类型？
3. 图片上传到本地服务器还是对象存储（R2/S3/又拍云）？
4. 下游 API 是否需要 token 鉴权，还是完全公开？
5. 随机接口是否需要按分类权重或最新优先？

## 7. 技术栈

- 后端：Go 1.22 + Gin + GORM
- 前端：React 18 + Semi Design + Vite
- 第三方：Notion API（通过官方 SDK 或 REST 调用）
- 存储：复用 new-api 现有文件上传方案（本地或对象存储）
