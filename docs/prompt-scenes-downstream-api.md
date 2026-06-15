# 提示词业务场景（Scenes）下游接口文档

> **版本**：v1.0  
> **生效时间**：2026-06-15  
> **状态**：已上线  
> **适用范围**：所有调用提示词公开/管理接口的下游应用、官网、H5、小程序、第三方合作方

---

## 1. 变更说明

提示词接口在返回数据中新增了 `scenes` 字段，用于标识每个提示词的**业务场景标签**。

`scenes` 由后端基于提示词现有的 `tags` + `title` + `description` + `content` + `content_en` 自动派生，**不需要调用方参与计算**，也不影响原有字段。

### 核心特点

- **零侵入**：所有旧接口路径、参数、鉴权均不变
- **自动派生**：命中规则即返回，未命中返回空数组
- **多对多**：一个提示词可同时属于多个业务场景
- **规则可热更新**：后端 `model/scene_derive.go` 维护规则表

---

## 2. 受影响接口

以下接口的响应中，`items` 内每个提示词对象或详情对象均会包含 `scenes` 字段。

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/public/prompts` | GET | 公开提示词列表 |
| `/api/public/prompts?ids=1,2,3` | GET | ID 批量查询 |
| `/api/public/prompts/:id` | GET | 通过 ID 获取详情 |
| `/api/public/prompts/slug/:slug` | GET | 通过 slug 获取详情 |
| `/api/prompt/:id` | GET | 管理后台详情（需登录） |

> 未列出的接口暂不支持 `scenes` 字段。

---

## 3. Scenes 字段结构

每个提示词对象新增字段：

```json
{
  "scenes": [
    {
      "scene": "电商",
      "icon": "🛒",
      "color": "orange"
    },
    {
      "scene": "产品摄影",
      "icon": "📷",
      "color": "orange"
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `scene` | string | 业务场景中文名，唯一标识 |
| `icon` | string | 场景对应的 emoji 图标 |
| `color` | string | 场景推荐配色，用于前端 Tag/徽章展示。取值参考 Semi UI Tag 的 `color` 属性 |

### 空值情况

若提示词未命中任何场景规则，返回空数组：

```json
{
  "scenes": []
}
```

---

## 4. 预置业务场景清单

当前共 20 个预置业务场景，后续可按需扩展。

| 场景名 | icon | color | 关键词示例（后端匹配逻辑） |
|--------|------|-------|----------------------------|
| 电商 | 🛒 | orange | product、商品、e-commerce、shop、listing、white background、studio |
| 短视频 | 📱 | pink | tiktok、reels、shorts、竖屏、vertical、vlog |
| TVC | 📺 | red | commercial、广告、tvc、brand campaign、product video |
| 广告 | 📢 | red | ad、advertising、广告、banner、campaign |
| 纪录片 | 🎬 | volcano | documentary、纪录片、interview、真实、candid |
| 教学 | 📚 | geekblue | tutorial、教学、education、course、教程、how-to |
| 社交媒体 | 💬 | lime | instagram、social media、ins、weibo、小红书、种草 |
| 品牌设计 | 🎨 | gold | brand、logo、identity、vi、品牌、海报、poster |
| 个人IP | 👤 | purple | personal brand、个人ip、自媒体人、influencer |
| 自媒体 | 🎙️ | purple | content creator、自媒体、blogger、creator |
| 直播 | 🔴 | red | live stream、直播、livestream、broadcast |
| 出版印刷 | 📖 | orange | print、publishing、出版、印刷、magazine、book |
| 游戏CG | 🎮 | green | game、游戏、cg、character、unreal、fantasy、character design |
| 影视后期 | 🎞️ | blue | film、影视、vfx、post-production、editing |
| 动画制作 | ✨ | cyan | animation、动画、anime、motion、cartoon |
| 产品摄影 | 📷 | orange | product photography、产品摄影、product photo、白底图 |
| 人像写真 | 👤 | purple | portrait、人像、headshot、face、model、写真 |
| 风景摄影 | 🏞️ | blue | landscape、风景、nature、mountain、ocean、sunset |
| 美食 | 🍜 | magenta | food、美食、cuisine、dish、restaurant |
| 时尚 | 👗 | cyan | fashion、时尚、runway、couture、街拍、streetwear |

> 说明：以上关键词仅作示意，后端实际采用更完整的规则匹配 `tags`、`title`、`description`、`content`、`content_en`。

---

## 5. 接口示例

### 5.1 列表接口

**请求**

```http
GET /api/public/prompts?page=1&page_size=20&sort=created_time&order=desc
```

**响应**

```json
{
  "success": true,
  "data": {
    "page": 1,
    "page_size": 20,
    "total": 959,
    "items": [
      {
        "id": 1923,
        "category_id": 5,
        "category_name": "摄影",
        "title": "产品白底图摄影提示词",
        "description": "用于生成电商产品主图、白底图、场景图",
        "slug": "product-white-bg-photo",
        "cover_image_url": "https://cdn.example.com/xxx.jpg",
        "usage_count": 1234,
        "created_time": 1718236800,
        "updated_time": 1718236800,
        "scenes": [
          { "scene": "电商", "icon": "🛒", "color": "orange" },
          { "scene": "产品摄影", "icon": "📷", "color": "orange" }
        ]
      }
    ]
  }
}
```

### 5.2 批量查询接口

**请求**

```http
GET /api/public/prompts?ids=1923,1845,1760&lang=en
```

**响应**

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1923,
        "title": "Product White Background Photo Prompt",
        "scenes": [
          { "scene": "电商", "icon": "🛒", "color": "orange" },
          { "scene": "产品摄影", "icon": "📷", "color": "orange" }
        ]
      },
      { "id": 1845, "title": "...", "scenes": [] },
      { "id": 1760, "title": "...", "scenes": [{ "scene": "短视频", "icon": "📱", "color": "pink" }] }
    ],
    "total": 3
  }
}
```

### 5.3 详情接口

**请求**

```http
GET /api/public/prompts/1923
```

**响应**

```json
{
  "success": true,
  "data": {
    "id": 1923,
    "category_name": "摄影",
    "title": "产品白底图摄影提示词",
    "description": "...",
    "content": "...",
    "tags": ["product", "white background", "studio"],
    "scenes": [
      { "scene": "电商", "icon": "🛒", "color": "orange" },
      { "scene": "产品摄影", "icon": "📷", "color": "orange" }
    ]
  }
}
```

---

## 6. 下游接入示例

### 6.1 普通前端渲染

```javascript
const prompt = await fetch('/api/public/prompts/1923').then(r => r.json());

const scenes = prompt.data.scenes || [];

scenes.forEach(s => {
  const tag = document.createElement('span');
  tag.className = `tag tag-${s.color}`;
  tag.textContent = `${s.icon} ${s.scene}`;
  container.appendChild(tag);
});
```

### 6.2 React 组件（Semi UI Tag）

```jsx
import { Tag, Space } from '@douyinfe/semi-ui';

function PromptScenes({ scenes }) {
  if (!scenes || scenes.length === 0) return null;

  return (
    <Space wrap>
      {scenes.map(s => (
        <Tag key={s.scene} color={s.color} size="small">
          {s.icon} {s.scene}
        </Tag>
      ))}
    </Space>
  );
}
```

### 6.3 按场景过滤列表

由于列表接口返回的每个对象都已带 `scenes`，调用方可在本地做二次过滤：

```javascript
const targetScene = '电商';

const response = await fetch('/api/public/prompts?page=1&page_size=100');
const { data } = await response.json();

const filtered = data.items.filter(p =>
  (p.scenes || []).some(s => s.scene === targetScene)
);
```

> 当前 `scenes` 为后端派生，暂不支持按场景作为服务端查询参数。如需服务端级过滤，请提交需求升级。

---

## 7. 注意事项

1. **字段稳定性**：`scenes` 是后端自动派生，不存储在数据库中，规则更新后历史数据会同步生效。
2. **顺序性**：同一提示词命中的多个场景按规则匹配度排序，调用方不应依赖固定顺序做业务判断。
3. **color 取值**：`color` 字段为方便前端展示使用，不同场景可能使用相同 color，建议以 `scene` 字段作为唯一标识。
4. **空数组**：未命中场景时返回 `[]`，调用方需做好空值兼容。
5. **扩展性**：新增业务场景无需改动接口协议，只需后端更新规则表即可。

---

## 8. 后续升级方向

| 阶段 | 方案 | 说明 |
|------|------|------|
| v1（当前） | 后端派生 | 基于现有 tags 自动计算，零数据库改动 |
| v2 | 后端缓存字段 | 在 `prompts` 表增加 `derived_scenes` 字段，避免重复计算 |
| v3 | 独立场景库 | 建立 `prompt_scenes` 表 + 多对多关系，支持运营后台管理场景 |

---

## 9. 变更记录

| 日期 | 版本 | 变更内容 |
|------|------|----------|
| 2026-06-15 | v1.0 | 新增 `scenes` 字段，后端自动派生 20 个业务场景标签 |
