# TK 素材库下游 API 文档

## 基础信息

- **Base URL**: `https://heharse.cloud/api`
- **协议**: HTTP/HTTPS
- **数据格式**: JSON
- **认证**: 当前接口为公开接口，无需额外 Token；受全局 API 速率限制影响。
- **图片访问**: 素材图片直接通过 `url` 字段访问，例如 `https://heharse.cloud/uploads/tk_materials/{filename}`。

## 响应格式

所有接口统一返回：

```json
{
  "success": true,
  "message": "",
  "data": { ... }
}
```

错误时 `success` 为 `false`，`message` 携带错误信息。

---

## 1. 分页获取素材列表

```
GET /api/public/tk/materials
```

### 请求参数（Query）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 否 | 分类筛选。完整分类值，如 `浴室`、`分析 UGC/男` |
| keyword | string | 否 | 搜索文件名或分类 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，最大 100 |

### 请求示例

```bash
curl -X GET "https://heharse.cloud/api/public/tk/materials?category=浴室&page=1&page_size=20"
```

### 响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "data": [
      {
        "id": 1,
        "category": "浴室",
        "url": "https://heharse.cloud/uploads/tk_materials/1948f533-95dd-4bec-8162-45ee46499141.jpg",
        "thumbnail_url": "https://heharse.cloud/uploads/tk_materials/1948f533-95dd-4bec-8162-45ee46499141.jpg",
        "filename": "1948f533-95dd-4bec-8162-45ee46499141.jpg",
        "file_type": "image/jpeg",
        "size": 123456,
        "width": 0,
        "height": 0,
        "source": "upload",
        "notion_page_id": "",
        "status": 1,
        "created_at": 1751445428,
        "updated_at": 1751445428
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20
  }
}
```

---

## 2. 获取单张素材详情

```
GET /api/public/tk/materials/:id
```

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 素材 ID |

### 请求示例

```bash
curl -X GET "https://heharse.cloud/api/public/tk/materials/1"
```

### 响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 1,
    "category": "浴室",
    "url": "https://heharse.cloud/uploads/tk_materials/1948f533-95dd-4bec-8162-45ee46499141.jpg",
    "thumbnail_url": "https://heharse.cloud/uploads/tk_materials/1948f533-95dd-4bec-8162-45ee46499141.jpg",
    "filename": "1948f533-95dd-4bec-8162-45ee46499141.jpg",
    "file_type": "image/jpeg",
    "size": 123456,
    "width": 0,
    "height": 0,
    "source": "upload",
    "notion_page_id": "",
    "status": 1,
    "created_at": 1751445428,
    "updated_at": 1751445428
  }
}
```

---

## 3. 随机获取素材

```
GET /api/public/tk/materials/random
```

### 请求参数（Query）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 否 | 分类。支持逗号分隔多个分类，如 `浴室,客厅,分析 UGC/男` |
| limit | int | 否 | 总条数，默认 1，最大 100。多个分类时会尽量均分 |

### 请求示例

```bash
# 单分类随机 5 张
curl -X GET "https://heharse.cloud/api/public/tk/materials/random?category=浴室&limit=5"

# 多分类随机 6 张（每个分类尽量 2 张）
curl -X GET "https://heharse.cloud/api/public/tk/materials/random?category=浴室,客厅,分析 UGC/男&limit=6"
```

### 响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "data": [
      {
        "id": 3,
        "category": "浴室",
        "url": "https://heharse.cloud/uploads/tk_materials/xxxx.jpg",
        "thumbnail_url": "https://heharse.cloud/uploads/tk_materials/xxxx.jpg",
        "filename": "xxxx.jpg",
        "file_type": "image/jpeg",
        "size": 123456,
        "width": 0,
        "height": 0,
        "source": "upload",
        "notion_page_id": "",
        "status": 1,
        "created_at": 1751445428,
        "updated_at": 1751445428
      }
    ]
  }
}
```

---

## 4. 下游上传素材（可选）

```
POST /api/public/tk/materials
```

### 请求参数（Form-Data）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 是 | 分类，如 `浴室` 或 `分析 UGC/男` |
| files | file | 是 | 图片文件，支持多个（多图请传同名 files） |

### 请求示例

```bash
curl -X POST "https://heharse.cloud/api/public/tk/materials" \
  -F "category=分析 UGC/男" \
  -F "files=@/path/to/image1.jpg" \
  -F "files=@/path/to/image2.jpg"
```

### 响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "saved": [
      {
        "id": 10,
        "category": "分析 UGC/男",
        "url": "https://heharse.cloud/uploads/tk_materials/aaaa.jpg",
        "thumbnail_url": "https://heharse.cloud/uploads/tk_materials/aaaa.jpg",
        "filename": "aaaa.jpg",
        "file_type": "image/jpeg",
        "size": 123456,
        "width": 0,
        "height": 0,
        "source": "upload",
        "notion_page_id": "",
        "status": 1,
        "created_at": 1751445428,
        "updated_at": 1751445428
      }
    ],
    "failed": []
  }
}
```

---

## 分类说明

当前支持的分类如下：

### 场景（一级分类：场景）

| 完整分类值 | 说明 |
|-----------|------|
| 浴室 | |
| 客厅 | |
| 厨房 | |
| 卧室 | |
| 车库 | |
| 院子 | |
| 街景 | |
| 健身房 | |
| 车 | |
| 机场 | |
| 农村 | |
| 公园 | |
| 超市 | |
| 仓库 | |

### 分析 UGC（一级分类：分析 UGC）

| 完整分类值 | 说明 |
|-----------|------|
| 分析 UGC/男 | |
| 分析 UGC/女 | |

> 接口中 `category` 字段必须传入完整分类值，例如 `分析 UGC/男`，不能只传 `分析 UGC` 或 `男`。

---

## 错误码

| 状态 | 说明 |
|------|------|
| 200 + success=false | 业务错误，message 携带具体原因 |
| 400 | 请求参数错误 |
| 404 | 素材不存在或已下架 |
| 429 | 触发全局 API 速率限制 |
| 500 | 服务器内部错误 |

---

## 下游对接建议

1. **随机取图推荐**：使用 `/api/public/tk/materials/random?category=浴室&limit=1` 作为最常用接口。
2. **批量取图**：`limit` 建议不超过 20，避免单请求过大。
3. **多分类轮询**：如需从多个场景各取若干张，使用逗号分隔 `category` 参数。
4. **缓存**：图片 URL 为静态文件，可长期缓存；列表接口建议按需拉取，不要高频轮询。
5. **图片加载失败兜底**：如 `url` 无法访问，可尝试用 `thumbnail_url` 或标记为失效。
