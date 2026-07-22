# 统一文件上传 + 图片/视频模型对接方案（最终版）

> **版本**：v1.0  
> **日期**：2026-07-07  
> **状态**：方案已确认，代码暂不改动  
> **适用范围**：所有通过 new-api 上传文件并调用图片/视频生成能力的下游应用

---

## 1. 目标

将 new-api 内部分散的本地文件上传逻辑收敛为一套统一的 Upload Service，实现：

- 下游/前端只认 **一个上传入口**；
- 上传后统一落盘，并返回 **可公网访问的 URL**；
- 支持图片、视频、音频、通用文件等任意类型；
- 支持多种输入方式：`multipart` 文件、`base64` data URI、HTTP URL；
- 旧接口保持兼容，内部复用同一套逻辑；
- 可扩展对象存储（R2）作为后端，但第一阶段不做 R2；
- 上传后的 URL 可直接用于 `/v1/images/generations` 和 `/v1/videos/generations` 的参考图参数。

---

## 2. 统一接口清单

下游应用只需要记 **3 个接口** + 1 个轮询接口：

| 步骤 | 接口 | 说明 |
|------|------|------|
| 1. 上传文件 | `POST /api/v1/upload?type=image&permanent=true` | 单个/多个文件 → 返回持久 URL |
| 2. 图片生成 | `POST /v1/images/generations` | 传 `model` + `prompt` + `image` / `image_urls` |
| 3. 视频生成 | `POST /v1/videos/generations` | 传 `model` + `prompt` + `ref_images` / `video_urls` |
| 4. 轮询结果 | `GET /v1/images/tasks/{task_id}` / `GET /v1/videos/{task_id}` | 异步任务查结果 |

---

## 3. 统一上传接口（`/api/v1/upload`）

### 3.1 请求

```bash
POST /api/v1/upload?type=image&permanent=true
Authorization: Bearer <token>
Content-Type: multipart/form-data

file=@cat.jpg
```

Query 参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 文件类型分类：`image`、`video`、`audio`、`file`、`material` |
| `permanent` | bool | 否 | 默认 `false`。图片/视频参考图建议传 `true` |
| `prefix` | string | 否 | 额外路径前缀，仅允许 `[a-zA-Z0-9_-]` |
| `retain_name` | bool | 否 | 默认 `false`，使用 UUID 避免冲突 |

请求体：支持 `file` 单文件或 `files` 多文件。

### 3.2 支持的输入方式

| 输入方式 | Content-Type | 请求体示例 | 说明 |
|----------|--------------|------------|------|
| multipart 文件 | `multipart/form-data` | `file=@cat.jpg` / `files=@1.jpg&files=@2.jpg` | 最常见 |
| base64 data URI | `application/json` | `{"image":"data:image/png;base64,xxxxx"}` | 兼容旧 `/upload_images/json` |
| HTTP URL | `application/json` | `{"url":"https://example.com/cat.jpg"}` | 透传或下载后缓存 |

### 3.3 响应

```json
{
  "success": true,
  "data": [
    {
      "url": "https://heharse.cloud/api/uploads/permanent/image/20260707/abc123.png",
      "filename": "abc123.png",
      "original_name": "cat.jpg",
      "size": 123456,
      "mime_type": "image/png",
      "type": "image",
      "permanent": true
    }
  ]
}
```

失败响应：

```json
{
  "success": false,
  "message": "file size exceeds limit",
  "code": "FILE_TOO_LARGE"
}
```

### 3.4 认证

复用现有中间件：

- 下游接口：`Authorization: Bearer <token>`（支持 `sk-` 前缀）
- 管理后台：Session / JWT
- 不需要额外权限控制

---

## 4. 存储目录设计

统一本地存储根目录：`./uploads`

```
./uploads/
├── image/20260707/{uuid}.png        # 临时图片，默认 3 天清理
├── video/20260707/{uuid}.mp4        # 临时视频，默认 3 天清理
├── audio/20260707/{uuid}.mp3
├── file/20260707/{uuid}.pdf
├── material/20260707/{uuid}.jpg
├── permanent/
│   ├── image/20260707/{uuid}.png     # 持久图片，跳过清理
│   ├── video/20260707/{uuid}.mp4
│   └── file/20260707/{uuid}.pdf
└── img/                               # 图片代理缓存，保持独立
    └── _index.json
```

说明：

- 旧目录（`uploads/permanent/`、`uploads/videos/`、`uploads/tk_materials/`）里的文件不迁移，继续可访问。
- 新上传统一走新目录结构。
- R2 接入后，路径结构保持一致，只是后端改为对象存储。

---

## 5. 文件大小与 MIME 限制

| type | 默认大小限制 | 允许 MIME |
|------|-------------|----------|
| `image` | 50 MB | image/* |
| `video` | 500 MB | video/* |
| `audio` | 100 MB | audio/* |
| `file` | 100 MB | */* |
| `material` | 100 MB | image/*, video/* |

- 超过限制返回 HTTP 413。
- 默认限制可在 `system_setting` 或 `operation_setting` 中配置。

---

## 6. 统一 Upload Service 设计

新增 `service/upload.go`：

```go
type UploadConfig struct {
    Type       string // image/video/audio/file/material
    Permanent  bool
    Prefix     string
    RetainName bool
}

type UploadResult struct {
    URL          string `json:"url"`
    Filename     string `json:"filename"`
    OriginalName string `json:"original_name"`
    Size         int64  `json:"size"`
    MimeType     string `json:"mime_type"`
    Type         string `json:"type"`
    Permanent    bool   `json:"permanent"`
}

// UploadFile 保存单个 multipart 文件并返回 URL
func UploadFile(ctx context.Context, file *multipart.FileHeader, cfg UploadConfig) (*UploadResult, error)

// UploadFiles 批量保存文件
func UploadFiles(ctx context.Context, files []*multipart.FileHeader, cfg UploadConfig) ([]*UploadResult, error)

// UploadBase64 保存 base64 data URI
func UploadBase64(ctx context.Context, dataURI string, cfg UploadConfig) (*UploadResult, error)

// UploadFromURL 下载远程 URL 并保存到本地
func UploadFromURL(ctx context.Context, fileURL string, cfg UploadConfig) (*UploadResult, error)

// DeleteUploadedFile 删除已上传文件（本地或 R2）
func DeleteUploadedFile(ctx context.Context, url string) error

// BuildUploadURL 根据请求上下文生成绝对 URL
func BuildUploadURL(c *gin.Context, relativePath string) string
```

---

## 7. 旧接口兼容策略

旧接口保留 URL 和请求格式不变，但内部调用 `service.UploadFile` / `UploadFiles` / `UploadBase64`：

| 旧接口 | 映射到统一 type | 处理方式 |
|--------|----------------|----------|
| `/uapi/v1/upload_images` | `image` | 内部调用 `UploadFile` |
| `/uapi/v1/upload_images/json` | `image` | 内部调用 `UploadBase64` |
| `/uapi/v1/upload_videos` | `video` | 内部调用 `UploadFile` |
| `/api/admin/tk/materials` | `material` | 内部调用 `UploadFile` |
| `/api/public/tk/materials` | `material` | 内部调用 `UploadFile` |
| `/uapi/v1/r2/upload-image` | `image` | 第二阶段再做 R2 接入 |
| `/api/admin/releases` | 保持独立 | 安装包命名和下载策略特殊 |
| `/api/article-media` | 保持 DB base64 | 该接口特殊，暂不迁移 |
| `/api/prompt-media` | 保持 DB base64 | 该接口特殊，暂不迁移 |

---

## 8. 实施阶段

### 第一阶段（推荐先做）

1. 新增 `service/upload.go`：统一保存、URL 生成、大小/MIME 校验；
2. 新增 `POST /api/v1/upload` 路由和 controller；
3. 改造 `/uapi/v1/upload_images`、`/uapi/v1/upload_images/json`、`/uapi/v1/upload_videos` 复用新逻辑；
4. 统一落盘到 `./uploads/{type}/{date}/{uuid}.{ext}` 或 `./uploads/permanent/{type}/{date}/{uuid}.{ext}`；
5. 更新 `service/upload_cleanup.go` 兼容新目录结构；
6. 补充 `POST /api/v1/upload` 的测试；
7. 认证复用现有 JWT/API Key 中间件。

### 第二阶段（后续做）

1. R2 对象存储后端接入；
2. 改造 `/api/admin/tk/materials` 复用统一上传；
3. 支持 URL 下载后缓存（`UploadFromURL`）；
4. 配置化大小限制接入 `operation_setting`。

---

## 9. 影响范围

### 代码文件

| 文件 | 影响 |
|------|------|
| `service/upload.go` | 新增 |
| `controller/upload.go` | 修改，注册新接口，旧接口复用 |
| `router/api-router.go` | 注册 `POST /api/v1/upload` |
| `service/upload_cleanup.go` | 修改，适配新目录 |
| `service/tk_material.go` | 第二阶段修改 |
| `web/src/pages/TKMaterialManagement/index.jsx` | 可能需适配 URL 结构变化 |
| `controller/upload_r2.go` | 第二阶段修改 |
| `controller/app_release.go` | 不改 |
| `controller/article-media.go` / `prompt-media.go` | 不改 |

### 部署/运维

| 项 | 影响 |
|----|------|
| Docker 挂载 | `./uploads` 目录不变，但子目录结构变化 |
| nginx | 大文件仍需 `client_max_body_size`；`client_max_body_size 500M` 建议 |
| 清理任务 | 必须同步修改，否则新目录文件不被清理或 permanent 文件被误清 |
| 历史文件 | 不迁移，URL 继续有效 |

### 前端/下游

| 项 | 影响 |
|----|------|
| 新接口 | 下游可以切换到 `/api/v1/upload` |
| 旧接口 | 返回格式不变，但 URL 路径会变 |
| 参考图 | 建议统一用 URL 形式传给图片/视频接口 |

---

## 10. 风险与注意事项

1. **旧接口 URL 路径变化**：如果下游或数据库硬编码了 `uploads/permanent/` 或 `uploads/videos/`，新文件路径会变化。旧文件 URL 不变，新文件 URL 走新目录。
2. **清理任务必须同步改**：`service/upload_cleanup.go` 需要按 `type/date` 目录扫描，同时跳过 `permanent`。
3. **路径注入**：`prefix` 参数必须白名单校验，禁止 `../`。
4. **nginx body size**：视频/大文件上传仍需 nginx 配合 `client_max_body_size`。
5. **MIME 校验**：优先用 `http.DetectContentType` 读取文件头做二次校验，避免伪造扩展名。
6. **R2 暂不做**：第一阶段只实现本地存储，但 Upload Service 接口预留 R2 扩展点。
7. **数据库 base64 接口**：`article-media` / `prompt-media` 不参与统一文件系统，避免破坏现有功能。

---

## 11. 下游调用流程

```
1. 上传本地文件
   POST /api/v1/upload?type=image&permanent=true
   Content-Type: multipart/form-data
   file=@cat.jpg

2. 拿到 URL
   {
     "success": true,
     "data": [{"url": "https://heharse.cloud/api/uploads/permanent/image/20260707/abc123.png"}]
   }

3. 调用图片/视频生成
   POST /v1/images/generations
   {
     "model": "nano-banana-2",
     "prompt": "把这只猫变成宇航员",
     "image_urls": ["https://heharse.cloud/api/uploads/permanent/image/20260707/abc123.png"],
     "size": "1K"
   }

4. 异步任务返回 task_id
   {
     "id": "task_xxx",
     "task_id": "task_xxx",
     "status": "queued"
   }

5. 轮询结果
   GET /v1/images/tasks/task_xxx
```

---

## 12. 相关文档

- `docs/下游应用端统一对接文档（图片+视频）.md`：完整的下游调用示例和 SDK
- `API_DOCUMENTATION.md`：现有接口文档
- `docs/GEMINIGEN_API.md`：GeminiGen 渠道专属接口
- `docs/下游应用端对接文档.md`：通用下游接口清单
