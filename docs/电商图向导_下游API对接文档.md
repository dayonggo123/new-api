# 电商图向导 — 下游 API 对接文档

> 版本：v1.0 | 生成日期：2026-05-23
> Base URL：`https://{your-domain}`

---

## 鉴权方式

所有接口需在请求头中携带 API Key：

```
Authorization: Bearer {apiKey}
```

`apiKey` 在后台「控制台 → 令牌」中创建。

---

## 通用响应格式

```json
{
  "success": true,
  "message": "",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| success | boolean | `true` 成功，`false` 失败 |
| message | string | 失败时返回错误描述 |
| data | any | 成功时返回业务数据 |

**错误示例：**
```json
{
  "success": false,
  "message": "category and platform are required"
}
```

---

## 接口 1：获取模特呈现方式

获取所有已启用的模特呈现方式配置，用于前端展示选择项。

### 请求

```
GET /api/ecommerce/model-poses
```

### 响应

```json
{
  "success": true,
  "message": "",
  "data": [
    {
      "id": 1,
      "pose_id": "no_model",
      "label": "无模特 / 纯产品",
      "description": "仅展示产品本身，适合白底主图、细节特写等",
      "cover_image_url": "https://cdn.example.com/poses/no-model.jpg",
      "sort_order": 0,
      "status": 1,
      "created_time": 1716451200,
      "updated_time": 1716451200
    },
    {
      "id": 2,
      "pose_id": "hold_product",
      "label": "手持产品",
      "description": "模特手持产品展示，突出产品的尺寸、质感和使用感",
      "cover_image_url": "https://cdn.example.com/poses/hold-product.jpg",
      "sort_order": 1,
      "status": 1,
      "created_time": 1716451200,
      "updated_time": 1716451200
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 自增主键 |
| pose_id | string | 唯一标识，如 `no_model`、`hold_product` |
| label | string | 显示名称 |
| description | string | 详细描述 |
| cover_image_url | string | 封面参考图 URL |
| sort_order | int | 排序权重（升序） |
| status | int | `1` = 启用，`2` = 禁用 |
| created_time | int64 | Unix 时间戳（秒） |
| updated_time | int64 | Unix 时间戳（秒） |

---

## 接口 2：获取案例库品类列表

获取所有已启用的案例库品类，用于前端展示品类选择。

### 请求

```
GET /api/ecommerce/case-categories
```

### 响应

```json
{
  "success": true,
  "message": "",
  "data": [
    {
      "id": 1,
      "category_id": "clothing",
      "category_name": "服装/服饰",
      "cover_image_url": "https://cdn.example.com/cases/clothing.jpg",
      "requires_model": true,
      "sort_order": 0,
      "status": 1,
      "created_time": 1716451200,
      "updated_time": 1716451200
    },
    {
      "id": 2,
      "category_id": "electronics",
      "category_name": "3C数码/电子产品",
      "cover_image_url": "https://cdn.example.com/cases/electronics.jpg",
      "requires_model": false,
      "sort_order": 1,
      "status": 1,
      "created_time": 1716451200,
      "updated_time": 1716451200
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 自增主键 |
| category_id | string | 唯一标识，如 `clothing`、`electronics` |
| category_name | string | 品类显示名称 |
| cover_image_url | string | 品类封面图 URL |
| requires_model | boolean | 该品类是否需要模特展示 |
| sort_order | int | 排序权重（升序） |
| status | int | `1` = 启用，`2` = 禁用 |
| created_time | int64 | Unix 时间戳（秒） |
| updated_time | int64 | Unix 时间戳（秒） |

---

## 接口 3：获取案例详情

根据品类和平台获取具体的案例详情数据，用于生成 Prompt。

### 请求

```
GET /api/ecommerce/case-detail?category={categoryId}&platform={platformId}
```

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 是 | 品类 ID，如 `clothing` |
| platform | string | 是 | 平台 ID，如 `taobao` |

### 响应

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 1,
    "category_id": "clothing",
    "platform_id": "taobao",
    "platform_name": "淘宝/天猫",
    "visual_features": "[\"竖屏750x1000px为主\", \"模特上身展示+场景搭配\", \"高饱和度背景突出季节感\", \"多张主图形成场景故事线\"]",
    "composition": "[\"全身/半身/细节三连击\", \"首图视觉冲击最大化\", \"行动线引导视线至卖点\"]",
    "lighting": "明亮均匀，突出面料质感和颜色还原度",
    "background_style": "高饱和度纯色或当季场景，强化氛围感",
    "case_reference": "ItoshIroshI法式复古女装（小红书月销百万）——首图用场景搭配+高饱和背景",
    "created_time": 1716451200,
    "updated_time": 1716451200
  }
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 自增主键 |
| category_id | string | 品类 ID |
| platform_id | string | 平台 ID |
| platform_name | string | 平台显示名称 |
| visual_features | string | **JSON 数组字符串**，视觉特征列表（3-5 条） |
| composition | string | **JSON 数组字符串**，构图方式列表（3-5 条） |
| lighting | string | 光影处理说明 |
| background_style | string | 背景风格说明 |
| case_reference | string | 参考案例名称/描述 |
| created_time | int64 | Unix 时间戳（秒） |
| updated_time | int64 | Unix 时间戳（秒） |

> **注意**：`visual_features` 和 `composition` 返回的是 JSON 数组字符串，前端需要 `JSON.parse()` 解析。

### 错误响应

```json
{
  "success": false,
  "message": "category and platform are required"
}
```

```json
{
  "success": false,
  "message": "record not found"
}
```

---

## 前端调用示例

### 模特呈现方式

```javascript
const res = await fetch('https://heharse.cloud/api/ecommerce/model-poses', {
  headers: { 'Authorization': 'Bearer sk-xxx' }
});
const { success, data } = await res.json();
if (success) {
  // data: EcommerceModelPose[]
  console.log(data[0].label); // "无模特 / 纯产品"
}
```

### 案例库品类

```javascript
const res = await fetch('https://heharse.cloud/api/ecommerce/case-categories', {
  headers: { 'Authorization': 'Bearer sk-xxx' }
});
const { success, data } = await res.json();
if (success) {
  // data: EcommerceCaseCategory[]
  const needsModel = data.filter(c => c.requires_model);
}
```

### 案例详情

```javascript
const res = await fetch(
  'https://heharse.cloud/api/ecommerce/case-detail?category=clothing&platform=taobao',
  { headers: { 'Authorization': 'Bearer sk-xxx' } }
);
const { success, data } = await res.json();
if (success) {
  const visualFeatures = JSON.parse(data.visual_features);
  const composition = JSON.parse(data.composition);
  console.log(visualFeatures[0]); // "竖屏750x1000px为主"
}
```

---

## 缓存建议

| 接口 | 调用时机 | 缓存策略 |
|------|---------|---------|
| `model-poses` | Wizard 首次打开时 | localStorage 缓存 24h |
| `case-categories` | Wizard 首次打开时 | localStorage 缓存 24h |
| `case-detail` | 需要生成 Prompt 时按需调用 | 内存缓存（按 `category+platform` 维度） |

### 降级策略

接口调用失败时，前端应保留当前硬编码数据作为 fallback，确保功能可用。

---

## 附录：品类与平台 ID 对照表（示例）

### 品类 ID

| ID | 名称 |
|----|------|
| `clothing` | 服装/服饰 |
| `electronics` | 3C数码/电子产品 |
| `beauty` | 美妆/护肤 |
| `food` | 食品饮料 |
| `home` | 家居/家电 |
| `jewelry` | 珠宝/配饰 |

### 平台 ID（常用）

| ID | 名称 |
|----|------|
| `taobao` | 淘宝/天猫 |
| `jd` | 京东 |
| `pdd` | 拼多多 |
| `douyin` | 抖音 |
| `xiaohongshu` | 小红书 |
| `amazon` | 亚马逊 |
| `temu` | Temu |

> 实际可用的品类和平台 ID 以后台配置为准，不建议在前端硬编码。
