# HarseTV FAQ 数据接口契约 v2 技术问题确认报告

> 确认日期：2026-06-17  
> 确认人：Senior Developer  
> 依据代码：
> - `service/seo_batch_translate.go`
> - `controller/auto_faq.go`
> - `model/prompt.go`
> - `model/article.go`

---

## 一、问题 1：中文 FAQ 修改后是否自动触发全语言重译？

### 结论

**当前不是实时自动触发，而是基于 SEO 自动翻译轮询 + 30 分钟冷却时间。**

### 详细机制

SEO 自动翻译轮询在 `service/seo_batch_translate.go` 中实现，启动 3 分钟后开始，扫描条件如下：

#### Prompt / Article 翻译触发条件

1. **从未翻译过**：`seo_i18n` 为空，但根字段 `faq` / `intro` / `seo_keywords` 有内容
2. **已部分翻译**：`seo_i18n` 不为空，且 `updated_time < 30 分钟前`
3. **失败重试**：`seo_translation_error != ""` 且 `updated_time < 30 分钟前`

```go
const batchSize = 20
cooldown := time.Now().Add(-30 * time.Minute).Unix()
```

### 对契约的影响

| 场景 | 是否会自动重译 FAQ | 说明 |
|---|---|---|
| 新增内容，seo_i18n 为空 | ✅ 会 | 下次轮询会扫描到并翻译 |
| 修改中文 FAQ，seo_i18n 已有数据 | ⚠️ 不会立即 | 需要等待 30 分钟冷却时间 |
| 修改中文 FAQ 后手动清空 seo_i18n | ✅ 会 | 等同于重新翻译 |
| 之前翻译失败，有 error 记录 | ✅ 会 | 冷却后会进入重试队列 |

### 建议写入契约

> 中文 FAQ 修改后，其他语言的 `seo_i18n` 不会立即自动更新。系统通过 SEO 自动翻译轮询处理，冷却时间为 30 分钟。如需立即更新，可手动清空对应内容的 `seo_i18n` 字段，或调用批量翻译接口触发。

### 是否需要改进

如果业务要求"修改中文 FAQ 后立即同步多语言"，需要新增机制：
- 在 `Prompt.Update()` / `Article.Update()` 中检测 `Faq` 字段变更；
- 变更时自动清空 `seo_i18n` 或标记待翻译状态；
- 或提供显式的"重新翻译 SEO"管理后台按钮。

---

## 二、问题 2：批量导入接口是否存在？

### 结论

**当前没有"批量导入已有 FAQ JSON"的接口。**

当前 FAQ 相关接口都是** AI 自动生成 FAQ**：

| 接口 | 方法 | 功能 |
|---|---|---|
| `/api/admin/prompts/:id/auto-faq` | POST | 为单个 Prompt AI 自动生成 FAQ |
| `/api/admin/articles/:id/auto-faq` | POST | 为单篇文章 AI 自动生成 FAQ |
| `/api/admin/prompts/auto-faq/batch` | POST | 批量为 Prompts AI 自动生成 FAQ（单次最多 20 个） |
| `/api/admin/articles/auto-faq/batch` | POST | 批量为文章 AI 自动生成 FAQ（单次最多 20 个） |
| `/api/admin/auto-faq/batch/:task_id` | GET | 查询批量生成任务状态 |

### AI 生成 FAQ 的当前规范

从 `controller/auto_faq.go` 中可见：

| 内容类型 | FAQ 数量 | 答案长度 | 首句结论 |
|---|---|---|---|
| 文章 | 3-5 个 | 50-150 中文字符 | 要求 |
| Prompt | 2-3 个 | 50-150 中文字符 | 要求 |

### 对契约的影响

契约 v2 第七节"写入接口建议"中的"批量导入"当前**没有对应实现**。如果需要该功能，需要后端新增接口。

### 建议写入契约

> 当前系统仅支持通过管理后台单条/批量 AI 自动生成 FAQ，不支持批量导入已有 FAQ JSON。如需批量导入功能，需后端新增独立接口。

### 是否需要新增批量导入接口

建议根据运营需求决定：
- 如果 FAQ 主要由 AI 生成 + 人工微调，当前接口已足够；
- 如果有大量历史 FAQ 需要迁移，建议新增 `/api/admin/prompts/faq/batch-import` 和 `/api/admin/articles/faq/batch-import`。

---

## 三、问题 3：Variants / Use cases / Sample images / Recommended params 是否已预留字段？

### 结论

**当前代码中没有这些字段，都是"未来扩展"字段。**

### Prompt 模型当前字段

```go
type Prompt struct {
    // ... 其他字段
    Variables     string  // JSON array of variable definitions（变量定义）
    GeoBlocks     string  // GEO 结构化内容 JSON
    GeoBlocksI18n string  // GEO 结构化内容多语言 JSON
}
```

- `Variables` 已存在，但它是提示词变量定义（如 `{{subject}}`），不是 `variants` 或 `use_cases`。
- `GeoBlocks` 是当前 Prompt 页最高优先级的结构化内容来源。

### Article 模型

```go
type Article struct {
    // ... 其他字段
    GeoBlocks     string
    GeoBlocksI18n string
}
```

同样没有 `variants` / `use_cases` / `sample_images` / `recommended_params`。

### 对契约的影响

契约 v2 第五节 Prompt 页内容优先级表中：

| 优先级 | 内容模块 | 字段来源 |
|---|---|---|
| 中 | Variants / Use cases | 待后端新增字段 |
| 低 | Sample images / Recommended params | 待后端新增字段 |

这些应明确标注为"未来扩展"，避免后端误以为字段已存在。

### 建议写入契约

> `Variants / Use cases / Sample images / Recommended params` 为 Prompt 页未来扩展字段，当前版本未实现。本期仅实现 GEO blocks、Intro、FAQ。

---

## 四、综合确认结果

| 待确认问题 | 结论 | 对契约 v2 的影响 |
|---|---|---|
| 中文 FAQ 修改后是否自动触发全语言重译？ | 否，基于 30 分钟冷却轮询 | 需补充说明，或新增实时同步机制 |
| 批量导入接口是否存在？ | 否，只有 AI 自动生成接口 | 需说明当前不支持，或新增接口 |
| 未来扩展字段是否已预留？ | 否，当前没有这些字段 | 需明确标注为"未来扩展" |

---

## 五、建议补充的契约条款

基于以上确认，建议在契约 v2 中补充：

### 5.1 FAQ 翻译触发机制

```
中文 FAQ 修改后，其他语言的 seo_i18n 不会实时自动更新。
系统通过 SEO 自动翻译轮询处理：
- 轮询间隔：10 分钟
- 冷却时间：30 分钟（同一内容两次翻译间隔）
- 触发条件：seo_i18n 为空 / 冷却时间已过 / 上次翻译失败

如需立即更新，可：
1. 手动清空 seo_i18n 字段；
2. 调用 SEO 批量翻译接口；
3. 在管理后台点击"重新翻译"。
```

### 5.2 批量导入接口说明

```
当前版本暂无 FAQ 批量导入接口。
已有接口：
- POST /api/admin/prompts/:id/auto-faq
- POST /api/admin/articles/:id/auto-faq
- POST /api/admin/prompts/auto-faq/batch
- POST /api/admin/articles/auto-faq/batch
- GET /api/admin/auto-faq/batch/:task_id

如需批量导入历史 FAQ，需后端新增接口。
```

### 5.3 未来扩展字段标注

```
Variants / Use cases / Sample images / Recommended params 为 Prompt 页未来扩展字段，
当前版本未实现，本期不做强制要求。
```

---

## 六、最终建议

契约 v2 在补充以上 3 条说明后，可以作为最终版发布。

如果业务要求**修改中文 FAQ 后立即同步多语言**，建议后端新增一个显式机制（监听 `Faq` 字段变更并触发 SEO 翻译），而不是依赖当前轮询机制。
