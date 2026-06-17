# SEO 内容营销团队工作流 × SEO 中心 集成方案

> 基于代码审计（`reports/code-audit-seo-assets-2026-06-17.md`）和现有 SEO 中心实现（`web/src/pages/SEOCenter/index.jsx`）制定。
> 目标：把 SEO 内容营销团队的 5 阶段工作流（研究 → 创作 → 优化联审 → CRO → 发布）无缝嵌入到 HeHarseCloud 后台的 SEO 中心。

---

## 一、现状盘点

### 1.1 SEO 中心已具备的能力

| 模块 | 当前实现 | 对应代码 |
|---|---|---|
| 关键词研究 | 输入种子词 → AI 返回高 ROI 词、主题簇、内容缺口 | `web/src/pages/SEOCenter/index.jsx` KeywordResearchTab |
| 内容生成 | 输入关键词 → 自动生成文章/Prompt → 可选自动 SEO/GEO/翻译/发布 | `service/content_generator.go` |
| 监控仪表盘 | 流量、索引、排名、CTR、健康评分、Top 关键词 | `service/seo_monitor.go` |
| SEO 字段模型 | `seo_title`、`seo_description`、`seo_keywords`、`intro`、`faq`、`seo_i18n`、`slug` 完整支持 | `model/article.go`、`model/prompt.go` |
| SEO 审计 | 文章/Prompt SEO 评分、历史记录、低分内容筛选 | `service/seo_audit.go`、`controller/seo_audit.go` |
| 内链建议 | 自动内链推荐 | `service/auto_internal_link.go` |
| 批量翻译 | SEO 字段 12 语言批量翻译任务队列 | `service/seo_batch_translate.go` |
| Google 索引 | URL 批量提交索引 | `service/google_indexing.go` |

### 1.2 当前缺口

- **关键词研究**只支持 AI 生成，缺少「站内机会词、竞品反查、SERP 深挖、社群挖掘」等精准关键词获取方法
- **内容生成**只有一次性生成，缺少 Phase 3 的「SEO 优化联审」流程（seo-optimizer、content-editor、link-strategist 三审）
- **监控仪表盘**与内容生成/优化没有形成闭环：发现低 CTR 词后不能一键回流到优化队列
- 没有**发布前质量门禁**：内容评分 ≥70 分才能发布

---

## 二、5 阶段工作流映射到代码

```
SEO 内容营销团队 SOP          SEO 中心代码落地
─────────────────────────────────────────────────────────────
Phase 1: 关键词研究     ──→   SEO 中心「关键词研究」tab 扩展
Phase 2: 内容创作       ──→   SEO 中心「内容生成」tab（已存在）
Phase 3: 优化联审       ──→   新增「内容优化」tab / 在生成后增加审核面板
Phase 4: CRO 优化       ──→   内容编辑器内嵌 CTA 建议 + 落地页模板
Phase 5: 发布与监控     ──→   监控仪表盘 + 低 CTR 自动回流优化
```

---

## 三、具体集成方案

### 3.1 Phase 1：关键词研究 tab 扩展

**目标**：把「7 个获取精准关键词的方法」产品化，不仅依赖 AI 生成。

#### 新增研究模式

在 `SEOCenter/index.jsx` 的 KeywordResearchTab 增加「研究模式」选择器：

```jsx
<Form.Select field="research_mode" initValue="ai_explore">
  <Option value="ai_explore">AI 主题探索</Option>
  <Option value="site_opportunity">站内机会词（GSC）</Option>
  <Option value="competitor_reverse">竞品反查</Option>
  <Option value="serp_deep">SERP 深挖</Option>
  <Option value="community">社群/论坛挖掘</Option>
</Form.Select>
```

#### 后端接口扩展

在 `controller/seo_research.go` 增加：

```go
POST /api/admin/seo/research          // 现有：AI 主题探索
POST /api/admin/seo/research/site-opportunity   // 站内机会词
POST /api/admin/seo/research/competitor         // 竞品反查
POST /api/admin/seo/research/serp-deep          // SERP 深挖
POST /api/admin/seo/research/community          // 社群挖掘
```

每个接口复用现有的 `service/seo_research.go` 或新增：

- `service/seo_research_site.go`
- `service/seo_research_competitor.go`
- `service/seo_research_serp.go`
- `service/seo_research_community.go`

#### 快速模板升级

`model.GetSEOQuickTemplates()` 返回的模板增加 `research_mode` 字段：

```go
type SEOQuickTemplate struct {
    ID           int    `json:"id"`
    Name         string `json:"name"`
    SeedKeyword  string `json:"seed_keyword"`
    Description  string `json:"description"`
    ResearchMode string `json:"research_mode"` // 新增
    Language     string `json:"language"`
}
```

### 3.2 Phase 2：内容生成 tab 增强

**目标**：让关键词一键进入内容生产，并保留人工编辑入口。

当前 `ContentGeneratorTab` 已支持：

```jsx
API.post('/api/admin/content/generate', {
  type: 'article' | 'prompt' | 'tutorial',
  keywords: [...],
  language: 'en',
  auto_seo: true,
  auto_geo: true,
  auto_translate: true,
  auto_publish: false,
})
```

#### 需要增强的点

1. **从关键词研究自动带入 brief**
   不仅要带 `keywords`，还要带：
   - 主关键词
   - 搜索意图
   - 推荐内容类型
   - 竞品差距主题
   - 建议 FAQ

2. **生成前支持人工编辑 brief**
   在 ContentGeneratorTab 增加一个「编辑 Brief」折叠面板，允许运营调整关键词、意图、目标字数、CTA 目标。

### 3.3 Phase 3：新增「内容优化」tab

**目标**：把 seo-optimizer、content-editor、link-strategist 的联审流程产品化。

#### 新增 Tab

```jsx
<TabPane tab="内容优化" itemKey="optimize">
  <ContentOptimizeTab />
</TabPane>
```

#### 后端接口

复用并聚合已有接口：

```go
// SEO 审计（已存在）
POST /api/admin/seo/audit
GET  /api/admin/seo/internal-links

// 需要新增：内容优化联审总接口
POST /api/admin/content/optimize
```

`ContentOptimizeRequest` 设计：

```go
type ContentOptimizeRequest struct {
    RecordID    int    `json:"record_id"`    // 文章/Prompt ID
    ContentType string `json:"content_type"` // article / prompt
    Language    string `json:"language"`
}

type ContentOptimizeResult struct {
    SEOScore       int                    `json:"seo_score"`
    HumanScore     int                    `json:"human_score"`
    Readability    map[string]interface{} `json:"readability"`
    KeywordHeatmap []KeywordPosition      `json:"keyword_heatmap"`
    MetaOptions    MetaOptions            `json:"meta_options"`
    InternalLinks  []InternalLinkSuggest  `json:"internal_links"`
    AIHints        []string               `json:"ai_hints"`
    PublishCheck   []CheckItem            `json:"publish_check"`
}
```

#### 实现位置

- 前端：`web/src/pages/SEOCenter/ContentOptimizeTab.jsx`
- 后端：`service/content_optimizer.go` 聚合调用：
  - `service/seo_audit.go`
  - `service/auto_internal_link.go`
  - `service/ai_helper.go`（AI 可读性/人性化评分）

### 3.4 Phase 4：CRO 优化嵌入编辑器

**目标**：在文章/Prompt 编辑页面给出 CTA、转化漏斗、心理学优化建议。

#### 最小改动点

在 `ArticleEditor/index.jsx` 和 Prompt 编辑页增加「CRO 建议」侧边栏：

```jsx
// 调用新增接口
POST /api/admin/content/cro-analysis
{
  "record_id": 123,
  "content_type": "article",
  "language": "en"
}
```

返回：CTA 位置建议、情感触发词、异议处理 FAQ、转化漏斗评分。

### 3.5 Phase 5：发布与监控闭环

**目标**：发布前有质量门禁，发布后有数据回流。

#### 发布前门禁

在内容发布流程中增加检查：

```go
func CanPublish(recordID int, contentType string) (bool, []string) {
    optimizeResult, _ := RunContentOptimize(recordID, contentType)
    if optimizeResult.SEOScore < 70 || optimizeResult.HumanScore < 70 {
        return false, []string{"SEO 评分或人性化评分低于 70 分"}
    }
    return true, nil
}
```

#### 监控仪表盘增强

当前 MonitorTab 已展示 Top 关键词和 CTR。增加：

1. **低 CTR 高曝光词列表** → 一键进入优化队列
2. **排名下降预警** → 自动提示需要更新内容
3. **内容健康度详情** → 跳转到内容优化 tab

新增接口：

```go
GET /api/admin/seo/monitor/low-ctr       // 低 CTR 机会词
GET /api/admin/seo/monitor/ranking-drop  // 排名下降内容
POST /api/admin/seo/monitor/queue-fix    // 加入优化队列
```

---

## 四、文件改动清单

### 4.1 前端

| 文件 | 改动 |
|---|---|
| `web/src/pages/SEOCenter/index.jsx` | 增加「内容优化」tab、关键词研究模式选择器 |
| `web/src/pages/SEOCenter/ContentOptimizeTab.jsx` | 新增优化联审面板 |
| `web/src/pages/SEOCenter/KeywordResearchTab.jsx` | 拆分或增强现有 KeywordResearchTab，支持多模式 |
| `web/src/pages/ArticleEditor/index.jsx` | 增加 CRO 建议侧边栏入口 |
| `web/src/components/seo/SEO.jsx` | 无需改动，直接复用 |

### 4.2 后端

| 文件 | 改动 |
|---|---|
| `router/api-router.go` | 新增 `/api/admin/seo/research/*`、`/api/admin/content/optimize`、`/api/admin/content/cro-analysis`、`/api/admin/seo/monitor/low-ctr` 等路由 |
| `controller/seo_research.go` | 新增多种研究模式控制器 |
| `service/seo_research_*.go` | 新增站内机会、竞品反查、SERP 深挖、社群挖掘服务 |
| `service/content_optimizer.go` | 新增内容优化联审聚合服务 |
| `service/content_cro.go` | 新增 CRO 分析服务 |
| `model/seo_research.go` | 扩展研究模板、研究结果模型 |

### 4.3 数据库

| 表/字段 | 改动 |
|---|---|
| `seo_research_history` | 增加 `research_mode` 字段 |
| `articles` / `prompts` | 可选增加 `seo_score`、`human_score`、`last_optimize_time` 字段 |

---

## 五、MVP 落地建议

如果想最小成本验证，按这个顺序做：

### P0（2 周内）

1. **关键词研究 tab 增加 SERP 深挖模式**
   - 用户输入种子词，后端调用 Google 搜索解析 PAA/Related/Featured Snippet
   - 前端展示「长尾词 + FAQ 问题」两列
   - 接入现有 `onAddKeyword` 流到内容生成

2. **内容生成结果接入人工编辑入口**
   - 生成完成后，「查看」按钮跳转到文章编辑器
   - 编辑器中预填生成的 SEO 字段

### P1（1 个月内）

3. **新增「内容优化」tab**
   - 输入文章/Prompt ID，返回 SEO 评分 + 可读性评分 + 内链建议
   - 评分低于 70 分红字提示

4. **监控仪表盘增加低 CTR 回流**
   - 展示高曝光低 CTR 关键词
   - 提供「去优化」按钮，跳转到内容优化 tab

### P2（2 个月内）

5. **竞品反查、社群挖掘模式**
6. **CRO 分析嵌入编辑器**
7. **发布前质量门禁强制化**

---

## 六、与现有接口的复用关系

| 新功能 | 复用的现有接口/服务 |
|---|---|
| 站内机会词 | 需接入 GSC API 或搜索日志；展示复用 MonitorTab |
| 竞品反查 | 可复用 `service/seo_research.go` 的 AI 分析，替换输入为竞品 URL |
| SERP 深挖 | 新增爬虫服务，结果格式复用 `model.SEOKeywordResearchResult` |
| 内容优化 | 聚合 `seo_audit.go` + `auto_internal_link.go` + AI 评分 |
| 监控回流 | 复用 `/api/admin/seo/monitor/*` |
| 发布门禁 | 在文章/Prompt 发布 controller 中调用内容优化服务 |

---

## 七、结论

SEO 中心不是从零开始，而是**在现有 3 个 tab 基础上做工作流深化**：

- 关键词研究：从单一 AI 生成 → 多源精准关键词获取
- 内容生成：从关键词 → 生成 → 保存 → 增加 brief 编辑和人工审核
- 内容优化：**新增 tab**，聚合已有 SEO 审计 + 内链 + AI 评分能力
- 监控仪表盘：从看数据 → 数据驱动回流优化

这样就把 SEO 内容营销团队的 5 阶段 SOP 完全嵌入了产品。
