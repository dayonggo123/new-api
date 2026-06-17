# 后台 SEO 中心逻辑总览

> 梳理范围：`F:\new api` Go 后端，聚焦 SEO/GEO 相关 controller、service、model、router 与配置。

---

## 一、整体架构

SEO 中心按功能可拆成 **6 大模块**：

| 模块 | 主要职责 | 核心入口文件 |
|------|----------|--------------|
| 1. SEO 配置 | AI 模型、Google Indexing API、站点域名 | `setting/operation_setting/seo_setting.go` |
| 2. 内容 SEO 管理 | Prompt / Article 的 SEO 字段生成、编辑、批量翻译、审计 | `controller/prompt.go`, `controller/article.go`, `service/article_seo.go`, `service/seo_batch_translate.go` |
| 3. SEO 审计 | 规则审计 + AI 审计，输出评分与优化建议 | `service/seo_audit.go`, `service/article_seo_audit.go` |
| 4. 关键词研究 | AI 关键词研究、预设模板、历史记录 | `service/seo_research.go`, `controller/seo_research.go`, `model/seo_research.go` |
| 5. Google 索引 | 手动/批量提交 URL、查询索引状态、翻译后自动提交 | `service/google_indexing.go`, `controller/seo_indexing.go` |
| 6. SEO 监控 | 流量/排名/健康度监控（当前多为模拟/占位） | `service/seo_monitor.go`, `controller/seo_monitor.go` |
| 7. GEO 结构化内容 | Prompt/Article 的 GEO Blocks 生成与多语言翻译 | `service/geo_blocks_generator.go` |
| 8. Sitemap | 公开接口输出文章/提示词 sitemap 数据 | `controller/sitemap.go` |

---

## 二、路由总览（`router/api-router.go`）

所有 SEO 相关接口均挂载在 `/api` 下，且基本都需要 `AdminAuth`。

### 2.1 Prompt SEO 路由组：`/api/prompt/seo`

```go
seoRoute := apiRouter.Group("/prompt/seo")
seoRoute.Use(middleware.AdminAuth())
{
    seoRoute.GET("/list", controller.GetPromptSEOList)                     // SEO 列表（分页 + 搜索）
    seoRoute.GET("/:id", controller.GetPromptSEODetail)                    // 单条详情
    seoRoute.PUT("/:id", controller.UpdatePromptSEOFields)                 // 更新 SEO 字段
    seoRoute.POST("/:id/regenerate", controller.RegeneratePromptSEO)       // AI 重新生成 SEO
    seoRoute.POST("/:id/audit", controller.AuditPromptSEOHandler)          // 执行 SEO 审计
    seoRoute.GET("/:id/audits", controller.GetPromptSEOAHistory)           // 审计历史
    seoRoute.GET("/:id/report", controller.GetPromptSEOReport)             // 单条 Markdown 报告
    seoRoute.GET("/stats", controller.GetPromptSEOStats)                   // 统计
    seoRoute.GET("/translate-stats", controller.GetPromptTranslateStats)
    seoRoute.GET("/all-translate-stats", controller.GetPromptAllTranslateStats)
    seoRoute.GET("/trends", controller.GetPromptSEOTrends)
    seoRoute.GET("/low-score", controller.GetLowScorePrompts)              // 低分 Prompt
    seoRoute.GET("/report-all", controller.GetAllSEOReport)                // 全部 Markdown 报告
    seoRoute.POST("/audit-batch", controller.BatchAuditPromptSEO)          // 批量审计
    seoRoute.POST("/batch-translate", controller.BatchTranslatePromptSEO)  // 批量翻译
    seoRoute.GET("/batch-translate/:task_id", controller.GetBatchTranslatePromptSEOStatus)
}
```

### 2.2 Article SEO 路由组：`/api/article/seo`

```go
articleSEORoute := apiRouter.Group("/article/seo")
articleSEORoute.Use(middleware.AdminAuth())
{
    articleSEORoute.GET("/list", controller.GetArticleSEOList)
    articleSEORoute.GET("/:id", controller.GetArticleSEO)
    articleSEORoute.PUT("/:id", controller.UpdateArticleSEOFields)
    articleSEORoute.POST("/:id/regenerate", controller.RegenerateArticleSEO)
    articleSEORoute.POST("/:id/audit", controller.AuditArticleSEOHandler)
    articleSEORoute.GET("/:id/audits", controller.GetArticleSEOAHistory)
    articleSEORoute.GET("/:id/report", controller.GetArticleSEOReport)
    articleSEORoute.GET("/report-all", controller.GetAllArticleSEOReport)
}
```

### 2.3 通用 SEO 路由

```go
apiRouter.POST("/admin/seo/research", middleware.AdminAuth(), controller.ResearchSEOKeywords)
apiRouter.GET("/admin/seo/research/templates", middleware.AdminAuth(), controller.GetSEOQuickTemplates)
apiRouter.GET("/admin/seo/research/history", middleware.AdminAuth(), controller.ListSEOResearchHistories)
apiRouter.GET("/admin/seo/research/history/:id", middleware.AdminAuth(), controller.GetSEOResearchHistory)
apiRouter.DELETE("/admin/seo/research/history/:id", middleware.AdminAuth(), controller.DeleteSEOResearchHistory)

apiRouter.GET("/admin/seo/audit", middleware.AdminAuth(), controller.GetSEOAudit)
apiRouter.GET("/admin/seo/internal-links", middleware.AdminAuth(), controller.GetInternalLinkSuggestions)

apiRouter.POST("/admin/seo/indexing", middleware.AdminAuth(), controller.SubmitToGoogleIndexing)
apiRouter.POST("/admin/seo/indexing/batch", middleware.AdminAuth(), controller.BatchSubmitToGoogleIndexing)
apiRouter.GET("/admin/seo/indexing/status", middleware.AdminAuth(), controller.GetGoogleIndexingStatus)

apiRouter.GET("/admin/seo/monitor", middleware.AdminAuth(), controller.GetSEOMonitorData)
apiRouter.GET("/admin/seo/monitor/history", middleware.AdminAuth(), controller.GetSEOMonitorHistory)
apiRouter.GET("/admin/seo/monitor/summary", middleware.AdminAuth(), controller.GetSEOHealthSummary)
apiRouter.POST("/admin/seo/monitor/simulate", middleware.AdminAuth(), controller.SimulateSEOMonitorData)
apiRouter.POST("/admin/seo/monitor/update", middleware.AdminAuth(), controller.UpdateSEOMonitorData)
```

### 2.4 公开 Sitemap 路由

```go
apiRouter.GET("/articles", controller.GetSitemapArticles)   // IP 限流 100/min
apiRouter.GET("/prompts", controller.GetSitemapPrompts)
```

---

## 三、配置模型（`setting/operation_setting/seo_setting.go`）

```go
type SEOSetting struct {
    SeoAIEnabled         bool   // 是否启用 AI 自动生成 SEO
    SeoAIModel           string // AI 模型，默认 gpt-4o-mini
    SeoAIBaseURL         string // API 基础地址
    SeoAIApiKey          string // API Key
    GoogleIndexingAPIKey string // Google Indexing API OAuth 令牌
    SiteDomain           string // 网站域名，如 harse.tv
}
```

> 注意：`GoogleIndexingAPIKey` 当前逻辑里直接作为 Bearer Token 使用，但 Google Indexing API 实际需要的是 **OAuth 2.0 access_token**，这里可能存在实现偏差。

---

## 四、内容 SEO 管理逻辑

### 4.1 Prompt SEO 字段

Prompt 表核心 SEO 字段：
- `seo_keywords`
- `intro`
- `faq`
- `seo_i18n`（JSON，多语言 SEO 字段）
- `title_i18n` / `i18n`（内容多语言）
- `geo_blocks` / `geo_blocks_i18n`
- `seo_translation_error`

生成入口：
- `generatePromptSEO()` → `service.GenerateSEOForPrompt()` → `service.UpdatePromptSEO()`

### 4.2 Article SEO 字段

Article 表核心 SEO 字段：
- `seo_title`
- `seo_description`
- `seo_keywords`
- `intro`
- `faq`
- `seo_i18n`
- `geo_blocks` / `geo_blocks_i18n`

生成入口：
- `generateArticleSEO()` → `service.GenerateSEOForArticle()` → `service.UpdateArticleSEO()`

### 4.3 生成逻辑（`service/article_seo.go`）

AI 生成提示词结构：
```
systemPrompt: 让 AI 生成 seo_title(50-60字符)、seo_description(150-160字符)、
              seo_keywords(8-12个)、intro(200-400字符)、faq(3-5个)

userPromptTemplate: Title / Summary / Content Preview / Author / Tags
```

生成后写入 DB；若 AI 没返回 intro，用 summary 或 content 前 300 字兜底。

### 4.4 批量 SEO 翻译（`service/seo_batch_translate.go`）

**目标语言**：`["en","fr","ru","ja","vi","ko","es","de","pt","it","ar"]`（11 种）

**流程**：
1. 构建 sourceSEO JSON：`{"seo_keywords":"...", "intro":"...", "faq":[...]}`
2. 逐个语言调用 `translateSEOJSONWithAI()`（整 JSON 翻译，保持 key 不变）
3. 校验翻译结果是否仍含中文（目标语言非中文时）
4. 写入 `seo_i18n` 字段
5. 异步触发 SEO 审计

**自动轮询**：启动后 3 分钟启动，每分钟扫描一次：
- 从未翻译过的记录（`seo_i18n` 为空）
- 已部分翻译的记录（冷却 3 分钟）
- 失败重试记录（`seo_translation_error != ''`）
- 每批最多 20 条，处理完 Prompt 再处理 Article

---

## 五、SEO 审计逻辑

### 5.1 规则审计（`service/seo_audit.go`）

统一入口：`AuditSEO(recordType, recordID)`，支持 `article` / `prompt`。

**6 个审计维度及权重**：

| 维度 | 权重 | 检查项 |
|------|------|--------|
| Title | 20% | 是否为空、长度 10-70 字符、是否含分隔符 |
| Content | 25% | 是否为空、字数、标题关键词是否出现在内容中 |
| SEO Fields | 20% | `seo_keywords` 是否为空/数量、intro 长度、faq 是否存在 |
| URL/Slug | 10% | 是否为空、长度、是否包含标题关键词 |
| Multi-language | 15% | 是否有英文翻译、是否覆盖 ≥10 种语言 |
| GEO Blocks | 10% | 是否存在、长度是否 ≥200 字符 |

**输出结构**：
- `overall_score`（加权总分）
- `dimension_scores`（各维度得分）
- `issues`（字段级问题，含 `auto_fixable` 标记）
- `suggestions`
- `categories`（前端兼容格式）
- `critical_issues` / `quick_wins`

### 5.2 AI 审计（`service/article_seo_audit.go`）

入口：`AuditArticleSEO(article)`

AI 按 5 个维度打分：
- Completeness
- Keyword Quality
- Title Quality
- Description Quality
- Structured Data / Technical

结果异步保存到 `article_seo_audit` 表。

### 5.3 内链建议（`service/auto_internal_link.go`）

入口：`SuggestInternalLinks(recordType, recordID, limit)`，目前靠关键词匹配推荐相关文章/Prompt。

---

## 六、关键词研究逻辑（`service/seo_research.go`）

### 6.1 入口

`POST /api/admin/seo/research`
请求体：`{ "seed_keyword": "...", "language": "en" }`

### 6.2 AI 提示词

- `systemPrompt`：要求 AI 返回严格 JSON，包含 seed/extended/long-tail/high-roi keywords、topic_clusters、content_gaps
- `userPrompt`：固定上下文为 harse.tv（节点画布、AI 视频提示词库、12 语言支持）

### 6.3 数据清洗与兜底

- 解析 JSON 失败时，进入 `fallbackParseResearchResult()`：先正则提取 `"keyword": "value"`，再逐行提取
- `cleanupGarbageKeywords()`：过滤 JSON 字段名、无意义词、纯符号/数字、过长词
- 若 AI 没返回高 ROI 词或主题簇，自动从已有词中补全

### 6.4 模型（`model/seo_research.go`）

核心结构：
- `KeywordItem`: keyword / search_volume / intent / difficulty / business_value / roi_score / suggested_url
- `TopicCluster`: name / pillar_keyword / pillar_volume / cluster_keywords / content_type / priority
- `ContentGap`: keyword / volume / competitors / gap_type / priority / suggested_action

预设 15 个快速模板，覆盖 AI video prompts、node canvas、ComfyUI、Sora、Kling 等方向。

---

## 七、Google 索引逻辑（`service/google_indexing.go`）

### 7.1 提交索引

- 单个：`SubmitToGoogleIndexing(url, type)`，type 默认 `URL_UPDATED`
- 批量：`BatchSubmitToGoogleIndexing(urls)`，最多 100 个，每个间隔 500ms
- 调用 endpoint：`https://indexing.googleapis.com/v3/urlNotifications:publish`

### 7.2 自动提交

`AutoSubmitAfterTranslation(recordType, recordID, slug)`：
- 翻译完成后异步提交 `/article/{slug}` 或 `/prompt/{slug}`
- 依赖 `GoogleIndexingAPIKey` 和 `SiteDomain` 配置

### 7.3 查询索引状态

`GetIndexingStatus(url)`：
- endpoint：`urlNotifications/metadata?url=...`
- 返回：`indexed` / `not_indexed` / `unknown`

---

## 八、SEO 监控逻辑（`service/seo_monitor.go`）

当前状态：**多为内存数据 + 模拟数据**。

- `GetSEOMonitorData()`：返回内存中当前监控数据
- `GetSEOMonitorHistory(days)`：返回最近 N 天历史
- `UpdateSEOMonitorData(data)`：手动更新，旧数据移入历史
- `SimulateMonitorData()`：生成模拟数据用于演示
- `UpdateMonitorFromGSC()`：TODO，未实现真实 GSC API 调用

数据字段：organic_traffic、indexed_pages、ranking_keywords、avg_position、top_keywords、health_score、issues。

---

## 九、GEO 结构化内容（`service/geo_blocks_generator.go`）

### 9.1 Prompt GEO Blocks

生成 JSON：
```json
{
  "scenarios": "80-150 字，首句直接回答适用场景",
  "steps": ["3-5 个操作步骤"],
  "tips": "60-120 字实用技巧"
}
```

### 9.2 Article GEO Blocks

生成 JSON：
```json
{
  "what": "30-80 字定义",
  "why": "80-150 字价值",
  "how": ["3-6 个步骤"],
  "summary": "< 40 字核心结论"
}
```

### 9.3 多语言翻译

生成后异步翻译到 11 种语言，写入 `geo_blocks_i18n`；另有每 10 分钟一次的自动轮询补齐缺失语言。

---

## 十、Sitemap 逻辑（`controller/sitemap.go`）

公开接口：
- `GET /api/articles?page=&pageSize=`：返回文章 sitemap 数据（id, slug, updatedAt RFC3339）
- `GET /api/prompts?page=&pageSize=`：返回提示词 sitemap 数据

数据库层：`model.GetPublicArticlesForSitemap()` / `model.GetPublicPromptsForSitemap()`，分页 Session 隔离修复见 `commit e92e3095`。

---

## 十一、关键数据表

| 表名 | 用途 |
|------|------|
| `prompts` | 含 `seo_keywords` / `intro` / `faq` / `seo_i18n` / `geo_blocks` 等字段 |
| `articles` | 含 `seo_title` / `seo_description` / `seo_keywords` / `intro` / `faq` / `seo_i18n` / `geo_blocks` 等 |
| `prompt_seo_audit` | Prompt 规则审计历史 |
| `article_seo_audit` | Article AI 审计历史 |
| `seo_research_histories` | 关键词研究历史 |

---

## 十二、当前潜在问题 / 注意点

1. **Google Indexing API 认证方式**：代码里把 `GoogleIndexingAPIKey` 当 Bearer Token，但 Google 官方要求 OAuth 2.0 access_token。
2. **SEO 监控未接真实数据**：GSC API 集成仍是 TODO，当前靠手动更新或模拟数据。
3. **Prompt 和 Article 的审计实现不一致**：Prompt 走规则审计（`service/seo_audit.go`），Article 走 AI 审计（`service/article_seo_audit.go`）。
4. **SEO 自动翻译轮询**：默认启用，可通过 `DISABLE_SEO_AUTO_TRANSLATE=true` 或 Option 表 `AutoTranslateEnabled=false` 关闭。
5. **关键词研究搜索量是 AI 估算**：非真实 Ahrefs/Semrush 数据，仅供内容规划参考。

---

*文档生成时间：2026-06-17*
