# HarseTV 后端 FAQ 数据接口文档评估

> 评估日期：2026-06-17  
> 评估范围：提示词详情页（Prompt Detail）与文章详情页（Article Detail）FAQ 数据接口方案  
> 依据代码：
> - `model/prompt.go` / `model/article.go`
> - `web/src/components/seo/SchemaOrg.jsx`
> - `web/src/pages/PromptDetail/index.jsx`
> - `web/src/pages/ArticleDetail/index.jsx`
> - 项目记忆（`MEMORY.md`）中 GEO + SEO 策略方向

---

## 一、总体结论

**方案方向基本正确，但存在与现有实现不一致的关键错误，需要修订后才能作为后端开发依据。**

主要问题：
1. **文章 FAQ 数据来源写错**：文档写从 `i18n` 读取，实际代码从 `seo_i18n` 读取。
2. **FAQ 触发阈值不一致**：文档写 `faq.length >= 3`，实际前端与 Schema 组件均按 `> 0` 触发。
3. **提示词 FAQ 优先级与项目记忆冲突**：项目记忆明确 Prompt 页 GEO 重点不是 FAQ，文档却将提示词 FAQ 列为"高优先级"。
4. **答案长度建议与现有规范冲突**：文档写 100-300 词，项目记忆写 50-150 字。

建议文档修订后重新发布，避免后端按错误规范写入数据。

---

## 二、逐条核对

### 2.1 提示词详情页 FAQ

| 文档描述 | 实际实现 | 结论 |
|---|---|---|
| 从 `seo_i18n` 读取多语言 FAQ | `Prompt.ApplyLanguage(lang)` 从 `seo_i18n[lang].Faq` 覆盖到 `p.Faq`；前端再从 `seoI18n[activeLang]?.faq` 读取 | ✅ 一致 |
| `faq` 支持 array 或 string | 数据库字段为 `TEXT`，实际存储的是 JSON 字符串（数组序列化后）；前端 `parseFAQ()` 统一 `JSON.parse` | ⚠️ 表述需明确：数据库/接口层面是 JSON 字符串，业务层面是数组 |
| `faq.length >= 3` 才注入 Schema | `FAQPageSchema` 判断 `!faqs.length` 即返回 null；PromptDetail 判断 `currentFaqList.length > 0` 即渲染 | ❌ 不一致，实际门槛是 `> 0` |
| 回退读取根字段 `faq` | 前端 `seoI18n[activeLang]?.faq || prompt.faq`；后端 `ApplyLanguage` 未命中目标语言时保持根字段不变 | ✅ 一致 |
| 提示词 FAQ 高优先级 | 项目记忆："Prompt 页的 GEO 重点不是 FAQ，而是使用场景和结构化内容" | ❌ 与项目策略冲突 |

### 2.2 文章详情页 FAQ

| 文档描述 | 实际实现 | 结论 |
|---|---|---|
| 从 `i18n` 读取多语言 FAQ | `Article.ApplyLanguage(lang)` 中 FAQ 来自 `seo_i18n[lang].Faq`；`i18n` 仅含 `title/summary/content` | ❌ **重大错误** |
| 其余格式/触发条件/数量建议 | 与提示词页一致，实际按 `> 0` 触发 | ❌ 阈值不一致 |

### 2.3 GEO 写作规范

| 文档建议 | 项目记忆规范 | 结论 |
|---|---|---|
| 首句直接给结论 | 首句直接给结论 | ✅ 一致 |
| 答案长度 100-300 词 | 答案长度建议 50-150 字 | ❌ 冲突，需统一 |
| 自然语言问题、结构化、关键词嵌入 | 与项目记忆一致 | ✅ 一致 |

---

## 三、发现的具体问题

### 问题 1：文章 FAQ 数据来源错误（高风险）

文档写：
> 前端从文章数据的 `i18n` 字段中读取多语言 FAQ。

实际：
- `ArticleSEO18n` 结构体在 `seo_i18n` 中，包含 `Faq` 字段。
- `ArticleContent18n` 结构体在 `i18n` 中，仅含 `title/summary/content`。
- 前端 `ArticleDetail` 读取的是 `seoI18n[activeLang]?.faq`。

**风险**：如果后端按文档把 FAQ 写入 `articles.i18n`，前端永远读取不到，FAQ 区块和 Schema 都不会渲染。

### 问题 2：FAQ 触发阈值不一致（中风险）

文档写 `faq.length >= 3`，但代码层面：
- `FAQPageSchema`：`if (!faqs.length) return null;`
- PromptDetail / ArticleDetail：`currentFaqList.length > 0`

**风险**：后端/运营可能误以为 1-2 条 FAQ 不会被使用，从而不写入；实际上只要写入就会渲染。建议统一为 `>=3` 或更新代码以匹配文档。

### 问题 3：提示词 FAQ 优先级与项目策略冲突（中风险）

项目记忆（2026-06-09 确认）：
> Prompt 页的 GEO 重点不是 FAQ，而是使用场景和结构化内容。

文档将"提示词 FAQ"列为"高优先级"，并要求为 Top 20 提示词补 FAQ。

**风险**：资源错配，建议优先补充 GEO blocks（使用场景/结构化内容），FAQ 作为辅助。

### 问题 4：答案长度规范冲突（低风险）

- 文档：100-300 词
- 项目记忆：50-150 字

**风险**：AI 生成 FAQ 时参数不统一，导致内容过长或过短。建议统一为项目记忆中的 50-150 字，或重新评估后修订记忆。

### 问题 5：缺少 FAQ 质量校验接口说明（低风险）

文档未说明：
- 后端写入 FAQ 时是否需要校验 JSON 格式？
- 是否需要校验 `question` / `answer` 非空？
- 是否需要校验目标语言非中文？
- 多语言 FAQ 是通过管理后台手动录入，还是由 `/api/translate/batch` 或 SEO 自动翻译生成？

---

## 四、修改建议

### 4.1 必须修正

1. **文章 FAQ 数据来源**：改为"前端从文章数据的 `seo_i18n` 字段中读取多语言 FAQ"，并明确 `i18n` 只负责 `title/summary/content`。
2. **FAQ 触发阈值**：
   - 方案 A（推荐）：将文档阈值从 `>=3` 改为 `> 0`，与代码一致；
   - 方案 B：保持 `>=3`，但同步修改 `SchemaOrg.jsx` 和详情页组件，增加 `>=3` 判断。
3. **提示词 FAQ 优先级**：将提示词 FAQ 从"高优先级"调整为"中/低优先级"，并在"优先填充建议"中补充 GEO blocks / 使用场景为最高优先级。
4. **答案长度**：统一为 50-150 字，与项目记忆一致；如需调整，先更新 `MEMORY.md` 再更新文档。

### 4.2 建议补充

1. **FAQ 数据写入接口**：明确 FAQ 是通过管理后台表单录入、SEO 自动翻译，还是批量导入。
2. **FAQ 字段格式示例**：给出 `seo_i18n` 的完整 JSON 示例，包括 `seo_keywords`、`intro`、`faq` 三个字段。
3. **回退机制说明**：补充"当目标语言无 FAQ 时，前端回退到根字段 `faq`（中文）"。
4. **校验规则**：建议后端写入前校验 `faq` 为合法 JSON 数组，且每条包含非空 `question` 和 `answer`。
5. **多语言一致性**：建议同一套 FAQ 的问题语义在各语言间保持一致，避免 AI 搜索引擎误判为低质量内容。

### 4.3 可选增强

- 在管理后台增加 FAQ 预览功能（渲染后的 Accordion + JSON-LD）。
- 增加 FAQ 数量/质量埋点，观察哪些 FAQ 被用户展开。

---

## 五、结论

该文档**不能直接作为后端开发规范使用**，需要先修正以下 4 点：

1. 文章 FAQ 数据来源从 `i18n` 改为 `seo_i18n`；
2. 统一 FAQ 触发阈值（文档与代码二选一）；
3. 调整提示词 FAQ 优先级，与项目 GEO 策略一致；
4. 统一答案长度规范为 50-150 字。

修正后，文档可以作为后端 FAQ 数据填充和前端 GEO 优化的有效参考。
