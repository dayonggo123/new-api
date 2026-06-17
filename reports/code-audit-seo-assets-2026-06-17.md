# SEO 技术资产代码审计报告

- 项目路径: f:\new api
- 审计日期: 2026-06-17
- 审计范围: 与 SEO、GEO、多语言、站点地图、内容生成相关的全部前后端代码、数据模型、API 路由、管理页面及配置
- 说明: 本报告仅做只读盘点，未修改任何业务代码

---

## 1. 项目整体结构

### 1.1 后端

- 语言 / 框架: Go 1.25.1，使用 Gin 作为 Web 框架，GORM 作为 ORM
- 模块名: github.com/QuantumNous/new-api
- 入口与路由: f:\new api\router\api-router.go 集中注册所有 /api/* 路由
- 主要目录:
  - f:\new api\model — 数据模型与数据库查询
  - f:\new api\controller — HTTP 控制器（Handler）
  - f:\new api\service — 业务逻辑与 AI 调用
  - f:\new api\setting\operation_setting — 运营配置（含 SEO 配置）
  - f:\new api\docs — SEO 相关文档

### 1.2 前端

- 框架: React 18.2.0 + Vite 5.2.0 + react-router-dom 6.3.0
- UI 库: @douyinfe/semi-ui（Semi UI）+ Tailwind CSS
- SEO 渲染: react-helmet-async 3.0.0
- 图表: @visactor/react-vchart 1.8.8
- 主要目录:
  - f:\new api\web\src\pages — 页面组件
  - f:\new api\web\src\components\seo — 可复用 SEO 组件
  - f:\new api\web\src\components\layout — 布局与侧边栏

---

## 2. 数据模型：Article / Prompt 与 SEO 字段

### 2.1 文章模型 f:\new api\model\article.go

Article 结构体已内置完整的 SEO 与多语言字段（节选）：

    SeoTitle       string         `json:"seo_title" gorm:"type:text"`
    SeoDescription string         `json:"seo_description" gorm:"type:text"`
    SeoKeywords    string         `json:"seo_keywords" gorm:"type:text"`
    Intro          string         `json:"intro" gorm:"type:text"`
    Faq            string         `json:"faq" gorm:"type:text"`
    I18n           string         `json:"i18n" gorm:"type:longtext"`
    SeoI18n        string         `json:"seo_i18n" gorm:"type:longtext"`
    GeoBlocks      string         `json:"geo_blocks" gorm:"type:longtext"`
    GeoBlocksI18n  string         `json:"geo_blocks_i18n" gorm:"type:longtext"`
    Slug           string         `json:"slug" gorm:"uniqueIndex;size:255"`

SEO 多语言项结构：

    type ArticleSEO18n struct {
        SeoTitle       string `json:"seo_title,omitempty"`
        SeoDescription string `json:"seo_description,omitempty"`
        SeoKeywords    string `json:"seo_keywords,omitempty"`
        Intro          string `json:"intro,omitempty"`
        Faq            string `json:"faq,omitempty"`
    }

ApplyLanguage(lang) 方法会根据 seo_i18n、i18n、geo_blocks_i18n 替换当前对象字段，缺失时保持默认中文。

站点地图查询：

    func GetPublicArticlesForSitemap(startIdx int, num int) (items []*ArticleSitemapItem, total int64, err error)

### 2.2 提示词模型 f:\new api\model\prompt.go

Prompt 结构体同样内置 SEO/GEO/多语言字段（节选）：

    Slug          string `json:"slug" gorm:"uniqueIndex;size:255"`
    SeoKeywords   string `json:"seo_keywords" gorm:"type:text"`
    Intro         string `json:"intro" gorm:"type:text"`
    Faq           string `json:"faq" gorm:"type:text"`
    I18n          string `json:"i18n" gorm:"type:text"`
    TitleI18n     string `json:"title_i18n" gorm:"type:text"`
    SeoI18n       string `json:"seo_i18n" gorm:"type:text"`
    GeoBlocks     string `json:"geo_blocks" gorm:"type:text"`
    GeoBlocksI18n string `json:"geo_blocks_i18n" gorm:"type:text"`

SEO 多语言项结构：

    type PromptSEO18n struct {
        SeoKeywords string `json:"seo_keywords,omitempty"`
        Intro       string `json:"intro,omitempty"`
        Faq         string `json:"faq,omitempty"`
    }

ApplyLanguage(lang) 处理 title_i18n、content_en / i18n、geo_blocks_i18n、seo_i18n。

站点地图查询：

    func GetPublicPromptsForSitemap(startIdx int, num int) (items []*PromptSitemapItem, total int64, err error)

Slug 生成工具：

    func GenerateSlug(title string) string

位于 f:\new api\model\article.go（第 443 行起），文章与提示词共用同一 slug 生成逻辑。

---

## 3. SEO 后端服务层

| 文件 | 作用 |
|------|------|
| f:\new api\service\article_seo.go | 调用 AI 为文章生成 seo_title、seo_description、seo_keywords、intro、faq，并写入数据库 |
| f:\new api\service\ai_helper.go | SEO 与翻译共用的 AI 调用辅助函数、prompt 渲染、JSON 提取 |
| f:\new api\service\seo_batch_translate.go | 批量翻译 SEO 字段（/api/prompt/seo/batch-translate 等业务实现） |
| f:\new api\service\seo_audit.go | 基于规则的 SEO 评分与审计逻辑 |
| f:\new api\service\article_seo_audit.go | 文章 SEO 审计具体实现与统计查询 |
| f:\new api\service\prompt_seo_audit.go | 提示词 SEO 审计具体实现与统计查询 |
| f:\new api\service\seo_research.go | AI 关键词研究（seed keyword -> 高 ROI 词、长尾词、主题聚类等） |
| f:\new api\service\google_indexing.go | Google Indexing API 提交 |
| f:\new api\service\seo_monitor.go | SEO 健康监控与模拟数据生成 |
| f:\new api\service\content_generator.go | /api/admin/content/generate 内容生成服务 |
| f:\new api\service\seo_keyword_task.go | SEO 关键词任务队列管理 |
| f:\new api\service\auto_internal_link.go | 自动内链建议 |

### 3.1 文章 SEO 生成

f:\new api\service\article_seo.go 中：

    func GenerateSEOForArticle(article *model.Article) (*ArticleSEOArticleResult, error)
    func UpdateArticleSEO(articleId int, result *ArticleSEOArticleResult)

生成结果结构：

    type ArticleSEOArticleResult struct {
        SeoTitle       string    `json:"seo_title"`
        SeoDescription string    `json:"seo_description"`
        SeoKeywords    string    `json:"seo_keywords"`
        Intro          string    `json:"intro"`
        Faq            []FaqItem `json:"faq"`
    }

### 3.2 批量翻译

f:\new api\service\seo_batch_translate.go 实现了提示词/文章 SEO 字段的异步批量翻译，控制器暴露：

- POST /api/prompt/seo/batch-translate
- GET /api/prompt/seo/batch-translate/:task_id

对应 controller 中的 BatchTranslatePromptSEO 与 GetBatchTranslatePromptSEOStatus。


---

## 4. SEO 控制器层

| 文件 | 作用 |
|------|------|
| f:\new api\controller\sitemap.go | 公开站点地图接口 /api/articles、/api/prompts |
| f:\new api\controller\prompt.go | 提示词 CRUD，含 SEO 字段更新与异步 SEO 生成 |
| f:\new api\controller\article.go | 文章 CRUD，含 SEO 字段更新与异步 SEO 生成 |
| f:\new api\controller\seo_research.go | 关键词研究接口 /api/admin/seo/research/* |
| f:\new api\controller\seo_research_history.go | 关键词研究历史管理 |
| f:\new api\controller\seo_monitor.go | SEO 监控接口 /api/admin/seo/monitor/* |
| f:\new api\controller\seo_indexing.go | Google 索引提交接口 /api/admin/seo/indexing/* |
| f:\new api\controller\seo_audit.go | SEO 审计与内链建议接口 /api/admin/seo/audit、/api/admin/seo/internal-links |
| f:\new api\controller\prompt_seo.go | 提示词 SEO 管理接口（列表、详情、编辑、重生成、审计、报告） |
| f:\new api\controller\article_seo.go | 文章 SEO 管理接口（列表、详情、编辑、重生成、审计、报告） |

---

## 5. API 路由总览（f:\new api\router\api-router.go）

### 5.1 提示词 SEO 路由

    seoRoute := apiRouter.Group("/prompt/seo")
    seoRoute.Use(middleware.AdminAuth())
    {
        seoRoute.GET("/list", controller.GetPromptSEOList)
        seoRoute.GET("/:id", controller.GetPromptSEODetail)
        seoRoute.PUT("/:id", controller.UpdatePromptSEOFields)
        seoRoute.POST("/:id/regenerate", controller.RegeneratePromptSEO)
        seoRoute.POST("/:id/audit", controller.AuditPromptSEOHandler)
        seoRoute.GET("/:id/audits", controller.GetPromptSEOAHistory)
        seoRoute.GET("/:id/report", controller.GetPromptSEOReport)
        seoRoute.GET("/stats", controller.GetPromptSEOStats)
        seoRoute.GET("/translate-stats", controller.GetPromptTranslateStats)
        seoRoute.GET("/all-translate-stats", controller.GetPromptAllTranslateStats)
        seoRoute.GET("/trends", controller.GetPromptSEOTrends)
        seoRoute.GET("/low-score", controller.GetLowScorePrompts)
        seoRoute.GET("/report-all", controller.GetAllSEOReport)
        seoRoute.POST("/audit-batch", controller.BatchAuditPromptSEO)
        seoRoute.POST("/batch-translate", controller.BatchTranslatePromptSEO)
        seoRoute.GET("/batch-translate/:task_id", controller.GetBatchTranslatePromptSEOStatus)
    }

### 5.2 文章 SEO 路由

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
        articleSEORoute.GET("/stats", controller.GetAllArticleSEOReport)
        articleSEORoute.GET("/translate-stats", controller.GetArticleTranslateStats)
        articleSEORoute.GET("/all-translate-stats", controller.GetArticleAllTranslateStats)
        articleSEORoute.GET("/low-score", controller.GetLowScoreArticlesHandler)
    }

### 5.3 站点地图公开接口

    GET /api/articles?page={page}&pageSize={pageSize}
    GET /api/prompts?page={page}&pageSize={pageSize}

由 f:\new api\controller\sitemap.go 实现。

### 5.4 SEO 中心相关管理接口

    POST   /api/admin/seo/research
    GET    /api/admin/seo/research/templates
    GET    /api/admin/seo/research/history
    GET    /api/admin/seo/research/history/:id
    DELETE /api/admin/seo/research/history/:id

    GET /api/admin/seo/audit
    GET /api/admin/seo/internal-links

    POST /api/admin/seo/indexing
    POST /api/admin/seo/indexing/batch
    GET  /api/admin/seo/indexing/status

    GET    /api/admin/seo/monitor
    GET    /api/admin/seo/monitor/history
    GET    /api/admin/seo/monitor/summary
    POST   /api/admin/seo/monitor/simulate
    POST   /api/admin/seo/monitor/update

    POST /api/admin/content/generate

### 5.5 自动化批量任务接口

    POST /api/admin/articles/:id/auto-faq
    POST /api/admin/prompts/:id/auto-faq
    POST /api/admin/articles/auto-faq/batch
    POST /api/admin/prompts/auto-faq/batch
    GET  /api/admin/auto-faq/batch/:task_id

    POST /api/admin/articles/:id/geo-blocks
    POST /api/admin/prompts/:id/geo-blocks
    POST /api/admin/articles/geo-blocks/batch
    POST /api/admin/prompts/geo-blocks/batch
    GET  /api/admin/geo-blocks/batch/:task_id

    GET  /api/admin/auto-translate-status
    PUT  /api/admin/auto-translate-status
    GET  /api/admin/auto-translate/:task_id
    GET  /api/admin/auto-translate-queue
    POST /translate/batch
    POST /translate/queue
    GET  /translate/queue/:id

    POST /api/admin/prompts/regenerate-slugs

---

## 6. 站点地图实现细节

f:\new api\controller\sitemap.go 提供两个公开接口：

    type SitemapArticleItem struct {
        Id        string `json:"id"`
        Slug      string `json:"slug"`
        UpdatedAt string `json:"updatedAt"` // ISO 8601
    }

    type SitemapPromptItem struct {
        Id        string `json:"id"`
        Slug      string `json:"slug"`
        UpdatedAt string `json:"updatedAt"` // ISO 8601
    }

    func GetSitemapArticles(c *gin.Context) // GET /api/articles
    func GetSitemapPrompts(c *gin.Context)  // GET /api/prompts

分页参数：

- page 默认 1
- pageSize 默认 100，最大 500

返回格式统一为：

    {
      "success": true,
      "data": {
        "list": [...],
        "total": 1000,
        "page": 1,
        "pageSize": 100,
        "totalPages": 10
      }
    }

提示词 slug 回退逻辑：若 Slug 为空，先使用 GenerateSlug(Title)，再回退到数字 ID。


---

## 7. 前端 SEO 组件

### 7.1 f:\new api\web\src\components\seo\SEO.jsx

基于 react-helmet-async 的统一 SEO 元数据组件，支持：

- title + 站点后缀拼接
- description、keywords
- Canonical URL
- Hreflang 多语言 alternate 标签与 x-default
- Robots index,follow / noindex,nofollow
- Open Graph（og:title、og:description、og:site_name、og:url、og:type、og:locale、og:image）
- Twitter Card
- Author / Copyright 元数据

### 7.2 f:\new api\web\src\components\seo\SchemaOrg.jsx

Schema.org JSON-LD 结构化数据组件集合：

- SoftwareApplicationSchema — 首页/关于页软件应用结构化数据
- FAQPageSchema — FAQ 页面结构化数据
- WebPageSchema — 通用网页结构化数据
- ArticleSchema — 文章结构化数据
- ProductSchema — 定价/产品结构化数据

### 7.3 文章详情页 SEO 应用 f:\new api\web\src\pages\ArticleDetail\index.jsx

- 使用 SEO 组件注入 title/description/canonical/hreflang
- 使用 WebPageSchema、ArticleSchema、FAQPageSchema 注入结构化数据
- 支持 12 语言 hreflang：zh、en、fr、ru、ja、vi、ko、es、de、it、pt、ar
- 支持通过 ?lang=xxx URL 参数切换语言
- 支持数字 ID 与 slug 两种访问方式：
  - /api/public/articles/:id
  - /api/public/articles/slug/:slug

---

## 8. 管理后台页面与路由

### 8.1 路由入口 f:\new api\web\src\App.jsx

已存在以下 SEO/GEO 相关管理路由：

    <Route path="/console/seo-center" element={<AdminRoute><SEOCenter /></AdminRoute>} />
    <Route path="/console/seo" element={<AdminRoute><SEOManagement /></AdminRoute>} />
    <Route path="/console/seo-trends" element={<AdminRoute><SEOTrends /></AdminRoute>} />
    <Route path="/console/geo" element={<AdminRoute><GEOManagement /></AdminRoute>} />
    <Route path="/console/prompt" element={<AdminRoute><Prompt /></AdminRoute>} />
    <Route path="/console/article" element={<AdminRoute><ArticleManagement /></AdminRoute>} />

### 8.2 侧边栏导航 f:\new api\web\src\components\layout\SiderBar.jsx

在运营子菜单中已注册：

    { text: t("SEO 中心"), itemKey: "seo_center", to: "/seo-center", tooltip: { content: t("一站式 SEO 自动化：关键词研究、内容生成、监控仪表盘") } },
    { text: t("SEO 趋势"), itemKey: "seo_trends", to: "/seo-trends", tooltip: { content: t("查看提示词 SEO 审核分数趋势和统计数据") } },
    { text: t("GEO 管理"), itemKey: "geo", to: "/geo", tooltip: { content: t("管理提示词和文章的结构化 GEO 内容，支持多语言") } },
    { text: t("提示词库"), itemKey: "prompt", to: "/prompt" },
    { text: t("文章管理"), itemKey: "article", to: "/article" },

### 8.3 已存在的 SEO 管理页面

| 文件 | 作用 |
|------|------|
| f:\new api\web\src\pages\SEOCenter\index.jsx | SEO 中心首页：关键词研究、内容生成、监控仪表盘 |
| f:\new api\web\src\pages\SEOManagement\index.jsx | 提示词 SEO 管理（列表、编辑、重生成、审计） |
| f:\new api\web\src\pages\ArticleSEOManagement\index.jsx | 文章 SEO 管理（列表、编辑、重生成、审计） |
| f:\new api\web\src\pages\SEOTrends\index.jsx | SEO 趋势与统计图表 |
| f:\new api\web\src\pages\GEOManagement\index.jsx | GEO 内容块管理 |
| f:\new api\web\src\pages\ArticleManagement\index.jsx | 文章列表与编辑入口 |
| f:\new api\web\src\pages\ArticleEditor\index.jsx | 文章编辑器（含 SEO 字段编辑） |
| f:\new api\web\src\pages\Setting\Operation\SettingsSEO.jsx | SEO AI 设置（模型、API Key、站点域名、Google Indexing 等） |

---

## 9. 现有 SEO Center 页面

f:\new api\web\src\pages\SEOCenter\index.jsx 已是一个功能完整的 SEO 中心，包含三个 Tab：

1. Keyword Research（关键词研究）
   - 调用 GET /api/admin/seo/research/templates 加载快速模板
   - 调用 POST /api/admin/seo/research 执行 AI 关键词研究
   - 调用 GET /api/admin/seo/research/history 查看历史记录
2. Content Generator（内容生成）
   - 调用 POST /api/admin/content/generate 生成内容
3. Monitor Dashboard（监控仪表盘）
   - 调用 /api/admin/seo/monitor/* 获取监控数据

重要结论：项目已经存在 /console/seo-center 页面与对应后端接口。因此“SEO 中心”功能的集成方向不是从零新建，而是在现有页面和接口基础上扩展更多模块（如审计概览、批量翻译任务、Google 索引提交、GEO 管理等）。

---

## 10. SEO 配置模型

f:\new api\setting\operation_setting\seo_setting.go：

    type SEOSetting struct {
        SeoAIEnabled         bool   `json:"seo_ai_enabled"`
        SeoAIModel           string `json:"seo_ai_model"`
        SeoAIBaseURL         string `json:"seo_ai_base_url"`
        SeoAIApiKey          string `json:"seo_ai_api_key"`
        GoogleIndexingAPIKey string `json:"google_indexing_api_key"`
        SiteDomain           string `json:"site_domain"`
    }

该配置通过 config.GlobalConfig.Register("seo_setting", &seoSetting) 注册，前端管理位置为 f:\new api\web\src\pages\Setting\Operation\SettingsSEO.jsx。


---

## 11. SEO 中心可接入的关键代码位置

### 11.1 前端扩展点

1. f:\new api\web\src\pages\SEOCenter\index.jsx
   - 已有 Tabs 结构，新增 TabPane 即可扩展“审计概览”、“批量任务”、“Google 索引”等模块。
2. f:\new api\web\src\components\layout\SiderBar.jsx
   - seo_center 导航项已存在，tooltip 文案可随功能扩展同步更新。
3. f:\new api\web\src\App.jsx
   - 如需新增独立页面（如 /console/seo-audit、/console/seo-indexing），可在此添加 Route。
4. f:\new api\web\src\components\seo\SEO.jsx / SchemaOrg.jsx
   - 新增 SEO 中心页面时直接复用这两个组件注入元数据和结构化数据。

### 11.2 后端扩展点

1. f:\new api\router\api-router.go
   - 所有 SEO 相关路由集中在此，新增 /api/admin/seo/xxx 接口建议放在现有 admin/seo/* 分组附近。
2. f:\new api\service 目录
   - 新建或扩展服务文件（如 seo_center.go）封装 SEO 中心综合业务。
   - 已有 seo_audit.go、seo_monitor.go、google_indexing.go、seo_batch_translate.go 可直接被 SEO 中心聚合调用。
3. f:\new api\controller 目录
   - 新增控制器函数并在 api-router.go 注册。
4. f:\new api\model\article.go / f:\new api\model\prompt.go
   - 若需新增 SEO 统计字段（如 seo_score、last_audit_time），可在这两个模型中增加字段并执行数据库迁移。
5. f:\new api\model\article_seo_audit.go / f:\new api\model\prompt_seo_audit.go
   - 已有 SEO 审计历史表，SEO 中心可直接查询历史记录、低分条目、趋势数据。

### 11.3 批量任务接入

- 复用 f:\new api\service\seo_batch_translate.go 的任务 ID 机制。
- 复用 f:\new api\service\seo_keyword_task.go 管理关键词任务队列。

### 11.4 站点地图与公开页面接入

- SEO 中心可调用 /api/articles 与 /api/prompts 获取全量 slug 列表，用于生成 XML Sitemap 或批量提交 Google Indexing。
- 前端公开页 ArticleDetail、PromptDetail 已具备 SEO 元数据与结构化数据，SEO 中心无需改动公开页即可受益。

---

## 12. SEO 相关文档

| 文件 | 作用 |
|------|------|
| f:\new api\seo-center-overview.md | SEO 中心功能概览文档 |
| f:\new api\docs\SEO_API.md | SEO API 文档 |
| f:\new api\docs\seo-sitemap-api.md | 站点地图 API 文档 |
| f:\new api\docs\SEO_CLAUDESEO_INTEGRATION.md | ClaudeSEO 集成说明 |
| f:\new api\docs\seo-automation-plan.md | SEO 自动化计划 |
| f:\new api\docs\seo-kw-research-sop.md | 关键词研究 SOP |

---

## 13. 完整文件清单

### 13.1 后端（Go）

- f:\new api\go.mod
- f:\new api\router\api-router.go
- f:\new api\model\article.go
- f:\new api\model\prompt.go
- f:\new api\model\article_seo_audit.go
- f:\new api\model\prompt_seo_audit.go
- f:\new api\model\seo_research.go
- f:\new api\controller\sitemap.go
- f:\new api\controller\prompt.go
- f:\new api\controller\article.go
- f:\new api\controller\prompt_seo.go
- f:\new api\controller\article_seo.go
- f:\new api\controller\seo_research.go
- f:\new api\controller\seo_research_history.go
- f:\new api\controller\seo_monitor.go
- f:\new api\controller\seo_indexing.go
- f:\new api\controller\seo_audit.go
- f:\new api\service\article_seo.go
- f:\new api\service\ai_helper.go
- f:\new api\service\seo_batch_translate.go
- f:\new api\service\seo_audit.go
- f:\new api\service\article_seo_audit.go
- f:\new api\service\prompt_seo_audit.go
- f:\new api\service\seo_research.go
- f:\new api\service\google_indexing.go
- f:\new api\service\seo_monitor.go
- f:\new api\service\content_generator.go
- f:\new api\service\seo_keyword_task.go
- f:\new api\service\auto_internal_link.go
- f:\new api\setting\operation_setting\seo_setting.go

### 13.2 前端（React）

- f:\new api\web\package.json
- f:\new api\web\src\App.jsx
- f:\new api\web\src\components\layout\SiderBar.jsx
- f:\new api\web\src\components\seo\SEO.jsx
- f:\new api\web\src\components\seo\SchemaOrg.jsx
- f:\new api\web\src\pages\SEOCenter\index.jsx
- f:\new api\web\src\pages\SEOManagement\index.jsx
- f:\new api\web\src\pages\ArticleSEOManagement\index.jsx
- f:\new api\web\src\pages\SEOTrends\index.jsx
- f:\new api\web\src\pages\GEOManagement\index.jsx
- f:\new api\web\src\pages\ArticleManagement\index.jsx
- f:\new api\web\src\pages\ArticleEditor\index.jsx
- f:\new api\web\src\pages\ArticleDetail\index.jsx
- f:\new api\web\src\pages\PromptDetail\index.jsx
- f:\new api\web\src\pages\Setting\Operation\SettingsSEO.jsx

### 13.3 文档

- f:\new api\seo-center-overview.md
- f:\new api\docs\SEO_API.md
- f:\new api\docs\seo-sitemap-api.md
- f:\new api\docs\SEO_CLAUDESEO_INTEGRATION.md
- f:\new api\docs\seo-automation-plan.md
- f:\new api\docs\seo-kw-research-sop.md

---

## 14. 结论

1. 项目已具备完整的 SEO 基础设施：数据模型、AI 生成服务、多语言翻译、审计评分、站点地图、Google 索引、监控仪表盘。
2. 项目已存在 /console/seo-center 页面，集成方向应为扩展现有页面，而非从零新建。
3. 新增 SEO 中心模块时，前端优先在 f:\new api\web\src\pages\SEOCenter\index.jsx 增加 Tab；后端优先在 f:\new api\router\api-router.go 增加 /api/admin/seo/* 路由，并在 f:\new api\service 中封装业务逻辑。
4. 站点地图公开接口、SEO 组件、Schema.org 结构化数据均可直接复用。

---

报告结束
