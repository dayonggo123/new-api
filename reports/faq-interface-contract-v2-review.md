# HarseTV FAQ 数据接口最终契约 v2 评估

> 评估日期：2026-06-17  
> 评估对象：《HarseTV FAQ 数据接口最终契约 v2》  
> 依据代码：
> - `model/prompt.go`
> - `model/article.go`
> - `web/src/pages/PromptDetail/index.jsx`
> - `web/src/pages/ArticleDetail/index.jsx`
> - `web/src/components/seo/SchemaOrg.jsx`

---

## 一、总体结论

**契约 v2 可以作为后端 FAQ 接口开发规范使用。**

相比上一版，v2 已经修正了关键的字段边界错误：
- Prompt 的 `i18n` 不再被误认为含 `faq`
- Article 的 `i18n` 不再被误认为含 SEO 字段
- FAQ 阈值统一为 `length > 0`，与当前代码一致

但仍有几处**建议补充**，以确保前后端协作无歧义：
1. 明确 FAQ 数据通过现有公开 API 返回，后端已做语言替换。
2. 说明 Prompt 的 `TitleI18n` 字段存在，但与 FAQ 无关。
3. 中文 FAQ 修改后的全语言重译机制，需要确认当前自动翻译队列是否支持。
4. "QA 抽检比例 ≥10%" 属于运营流程，技术契约中难以强制执行，建议单独作为 SOP。

---

## 二、逐条核对

### 2.1 修正说明

| 修正点 | v2 描述 | 与代码一致性 | 评估 |
|---|---|---|---|
| Prompt i18n 不含 faq | 仅含 content | ✅ 一致 | 已修正 |
| Article i18n 不含 SEO 字段 | 仅含 title/summary/content | ✅ 一致 | 已修正 |
| FAQ 阈值 | `> 0` | ✅ 一致 | 无需改动 |

### 2.2 字段边界总览

#### Prompt 表字段

契约描述基本准确。但建议补充：
- `TitleI18n` 字段存在（`map[string]string`），用于标题多语言，与 FAQ 无关。
- `GeoBlocks` 和 `GeoBlocksI18n` 字段存在，是高优先级 GEO 内容来源。

完整 Prompt 字段边界补充：

```go
type Prompt struct {
    // 内容字段
    Title       string
    Content     string    // 中文提示词原文
    ContentEn   string    // 英文提示词原文（en 优先使用）
    Description string

    // SEO 字段（根字段，中文优先）
    SeoTitle       string
    SeoDescription string
    SeoKeywords    string
    Intro          string
    Faq            string    // JSON 字符串，中文 FAQ

    // 多语言字段
    TitleI18n     string    // map[string]string，标题多语言
    I18n          string    // map[string]string，仅 content
    SeoI18n       string    // map[string]PromptSeoI18nEntry
    GeoBlocks     string    // GEO 结构化内容 JSON
    GeoBlocksI18n string    // GEO 结构化内容多语言 JSON
}
```

#### Article 表字段

契约描述准确。建议补充 `GeoBlocks` / `GeoBlocksI18n` 字段，因为 Prompt 页内容优先级表中提到 GEO blocks，Article 也应保持一致（虽然契约重点是 FAQ）。

### 2.3 FAQ 读取规则

| 规则 | 评估 |
|---|---|
| Prompt FAQ 优先从 `seo_i18n[lang].faq` 读取 | ✅ 正确 |
| Prompt 中文 fallback 到根字段 `faq` | ✅ 正确 |
| Article FAQ 优先从 `seo_i18n[lang].faq` 读取 | ✅ 正确 |
| Article 中文 fallback 到根字段 `faq` | ✅ 正确 |
| 触发阈值 `length > 0` | ✅ 与代码一致 |

**建议补充**：前端实际读取的是接口返回后的 `prompt.faq` / `article.faq` 字段，因为后端 `ApplyLanguage(lang)` 已经把 `seo_i18n[lang].faq` 覆盖到根字段。前端代码中：

```tsx
const currentFaqStr = activeLang === 'zh'
  ? prompt.faq
  : (seoI18n[activeLang]?.faq || prompt.faq);
```

虽然前端保留了 `seoI18n` 回退逻辑，但规范上应明确：**后端接口返回的数据中，`faq` 字段已经是当前语言的内容**。

### 2.4 FAQ 内容规范

| 规范项 | 评估 |
|---|---|
| JSON 数组格式，含 question/answer | ✅ 合理 |
| 中文问题 5-150 字，答案 50-200 字 | ✅ 合理 |
| 英文问题 10-200 chars，答案 80-300 words | ✅ 合理 |
| 真实相关、自然提及、组内去重 | ✅ 合理 |
| 跨内容不整组复制 | ✅ 合理 |
| 语言匹配 | ✅ 合理 |

**小建议**：
- "答案中自然出现产品名/功能名/模型名" 可以举例说明，例如 HarseTV、节点画布、Kling 等。
- "组内去重 80%" 可以用简单规则实现，如禁止两条 answer 完全相同，或 Levenshtein 相似度阈值。

### 2.5 Prompt 页内容优先级

| 优先级 | 内容模块 | 评估 |
|---|---|---|
| 高 | GEO blocks | ✅ 与项目策略一致 |
| 高 | Intro / 使用场景 | ✅ 合理 |
| 中 | FAQ | ✅ 合理 |
| 中 | Variants / Use cases | ⚠️ 字段待新增，需单独规划 |
| 低 | Sample images / Recommended params | ⚠️ 字段待新增，需单独规划 |

**建议**：如果 Variants / Use cases / Sample images / Recommended params 近期不实现，可以在契约中标注为"未来扩展"，避免后端困惑。

### 2.6 后端写入校验规则

| 校验项 | 评估 |
|---|---|
| 合法 JSON 数组 | ✅ 必须 |
| 每个元素含 question/answer | ✅ 必须 |
| question/answer 长度限制 | ✅ 合理 |
| 同组 answer 相似度不超过 80% | ✅ 合理，但需定义算法 |
| 跨内容不整组复制 | ✅ 合理，但需定义检测方式 |
| 语言与 lang 一致 | ✅ 必须 |

**建议补充**：
- 校验失败时的错误码和错误信息格式。
- 是否允许 `faq` 为空字符串？（当前代码 `parseFAQ` 会把空字符串、`'null'` 都返回空数组，所以可以。）

### 2.7 写入接口建议

| 写入方式 | 评估 |
|---|---|
| 后台录入 | ✅ 主要方式 |
| SEO 自动翻译 | ✅ 需要支持 |
| 批量导入 | ✅ 可选 |

**建议补充**：
- 后台录入时，是否允许只录入中文 FAQ，由系统自动翻译生成 `seo_i18n`？
- 批量导入接口的完整 URL 路径、鉴权方式、错误返回格式。

### 2.8 前后端分工确认

| 职责 | 后端 | 前端 | 评估 |
|---|---|---|---|
| FAQ 数据生产 | ✅ | ❌ | 正确 |
| FAQ 质量校验 | ✅ | ❌ | 正确 |
| FAQ 读取/渲染 | ❌ | ✅ | 正确 |
| FAQ Schema 注入 | ❌ | ✅ | 正确 |
| GEO blocks 生产 | ✅ | ❌ | 正确 |
| GEO blocks 渲染 | ❌ | ✅ | 正确 |

分工表清晰合理。

---

## 三、需要确认的技术问题

### 3.1 中文 FAQ 修改后是否自动触发全语言重译？

契约写：
> 翻译流程应保证：中文 FAQ 修改后，同步更新其他语言的 seo_i18n 翻译版本。

需要确认：
- 当前自动翻译队列是否监听 `faq` 字段变更？
- 是增量更新（只更新 FAQ 对应语言节点），还是全量重译（SEO 所有字段一起重译）？
- 如果自动翻译失败，如何提示运营人员？

### 3.2 当前公开 API 是否已返回正确字段？

契约未明确 API 路径。当前公开接口：
- `/api/public/prompts/:id`
- `/api/public/prompts/slug/:slug`
- `/api/public/articles/:id`
- `/api/public/articles/slug/:slug`

这些接口在返回前会调用 `ApplyLanguage(lang)`，因此前端拿到的 `faq` 字段已经是当前语言内容。契约应明确这一点，避免前端开发时绕过 `ApplyLanguage` 自己解析 `seo_i18n`。

### 3.3 "QA 抽检比例 ≥10%" 如何落地？

这是运营流程要求，技术侧无法强制。建议：
- 在管理后台增加"待 QA"标记和抽检记录；
- 或作为内容运营 SOP 单独文档，不放在技术契约中。

---

## 四、建议补充的契约条款

基于以上评估，建议在 v2 基础上补充以下内容：

### 4.1 API 返回说明

> 后端公开接口（`/api/public/prompts/:id`、`/api/public/articles/:id` 等）在返回数据前会调用 `ApplyLanguage(lang)`，将 `seo_i18n[lang].faq` 覆盖到根字段 `faq`。因此前端读取 `prompt.faq` / `article.faq` 即可，无需自行解析 `seo_i18n`。

### 4.2 Prompt 字段边界补充

> Prompt 表额外存在 `TitleI18n` 字段，用于标题多语言，与 FAQ 无关。

### 4.3 未来扩展字段标注

> Variants / Use cases / Sample images / Recommended params 为 Prompt 页未来扩展字段，当前版本不做强制要求。

### 4.4 校验失败处理

> 后端校验失败时，应返回明确错误信息，例如：
> - `faq format invalid: must be JSON array`
> - `faq item 0 missing question`
> - `faq answer too long: 320 chars (max 200)`

### 4.5 翻译触发机制

> 中文 FAQ 修改保存后，后端应标记对应内容的 `seo_i18n` FAQ 节点为待翻译，由自动翻译队列异步更新。如翻译失败，记录到 `seo_translation_error` 字段并在管理后台提示。

---

## 五、最终结论

**《HarseTV FAQ 数据接口最终契约 v2》整体可行，建议作为后端开发规范使用。**

该契约已经修正了上一版的关键错误，字段边界、读取规则、触发条件、内容规范、写入校验均与当前代码和项目策略基本一致。

**采纳前建议补充**：
1. API 返回时 `ApplyLanguage` 已经替换 `faq` 字段的说明；
2. Prompt `TitleI18n` 字段的存在说明；
3. 中文 FAQ 修改后自动触发翻译的机制说明；
4. 校验失败时的错误信息格式；
5. 将"QA 抽检比例 ≥10%"移到运营 SOP，技术契约中不强制。

补充以上 5 点后，契约可以作为最终版发布。
