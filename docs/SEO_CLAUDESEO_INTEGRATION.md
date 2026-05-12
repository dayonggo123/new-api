# Claude-SEO 与现有 SEO 系统结合方案

> 项目现有 SEO 系统已覆盖：AI 生成、SSR、Sitemap、Schema.org、Meta 标签、Robots.txt
> Claude-SEO 可作为**能力扩展层**融入，补足缺失维度

---

## 一、现有系统能力盘点

| 维度 | 现状 | 评价 |
|---|---|---|
| **AI 内容生成** | 自动生成 keywords / intro / FAQ | ✅ 成熟 |
| **SSR 服务端渲染** | `/prompt/:id` 预注入 meta + JSON-LD | ✅ 成熟 |
| **Sitemap** | 动态 XML，含全部公开 prompt | ✅ 成熟 |
| **Schema.org** | CreativeWork + FAQPage + SoftwareApplication | ✅ 成熟 |
| **Robots.txt** | 静态文件，已支持 AI Crawler 管理 | ✅ 成熟 |
| **多语言 SEO** | `seo_i18n` 字段存在，**但未使用** | ⚠️ 半成品 |
| **技术 SEO 审计** | 无 | ❌ 缺失 |
| **Google APIs** | 无 | ❌ 缺失 |
| **SEO 报告** | 无 | ❌ 缺失 |
| **内容质量评分** | 无 | ❌ 缺失 |

---

## 二、Claude-SEO 可移植能力

```
claude-seo/
  skills/seo-technical/SKILL.md      → 技术 SEO 审计（9 类）
  skills/seo-content/SKILL.md        → E-E-A-T 内容质量评分
  skills/seo-google/SKILL.md         → Google APIs 集成
  skills/seo-schema/SKILL.md         → Schema.org 检测/生成（已有，可对照优化）
  skills/seo-sitemap/SKILL.md        → Sitemap 质量分析（已有，可对照优化）
  skills/seo-performance/SKILL.md    → Core Web Vitals 分析
  skills/seo-images/SKILL.md         → 图片 SEO 优化
  skills/seo-plan/SKILL.md           → SEO 战略规划
  agents/seo-*.md                    → 18 个子代理的系统提示词
  scripts/*.py                       → Google APIs 调用脚本
```

---

## 三、结合方案（分阶段）

### 阶段一：多语言 SEO 完善（1 天，立即收益）

**问题：** `seo_i18n` 字段已在数据库和 API 中存在，但 SSR 渲染和前端均未使用。

**改动：**

1. **后端** `controller/misc.go` — `GetPromptSEOPage` 增加语言识别：
   ```go
   lang := c.Query("lang")
   if lang == "" {
       // 尝试从 Cookie / Accept-Language 识别
       lang = parseLangFromRequest(c)
   }
   prompt.ApplyLanguage(lang)  // 复用已有的 ApplyLanguage 逻辑
   ```

2. **后端** `model/prompt.go` — `ApplyLanguage` 增加 SEO 字段翻译：
   ```go
   if t.SeoKeywords != "" { p.SeoKeywords = t.SeoKeywords }
   if t.Intro != "" { p.Intro = t.Intro }
   if t.Faq != "" { p.Faq = t.Faq }
   ```

3. **前端** `SEOManagement/index.jsx` — 添加多语言 Tab 编辑（参考 PresetPrompt 安全写法）
   - 12 语言 Tab
   - AI 一键翻译 keywords / intro / FAQ
   - 调用 `/api/translate/batch`

4. **前端** `PromptDetail/index.jsx` — `<SEO>` 组件传入当前语言
   ```jsx
   <SEO title={prompt.name} description={prompt.intro} lang={i18n.language} />
   ```

**收益：** 海外 SEO 立即生效，已有字段不再浪费。

---

### 阶段二：SEO 审计功能（2-3 天）

**目标：** 在 SEO 管理页面新增"SEO 审计"Tab，对一个 prompt 页面进行技术 + 内容审计。

**实现方式：** 不移植 Python 脚本，而是用 **AI Skill 调用**（复用现有 AI 基础设施）。

**后端：**

1. 新增 `POST /api/prompt/seo/:id/audit`
2. 收集页面信息：URL、HTML、已有 SEO 字段
3. 调用配置好的 AI 模型，使用 claude-seo 的提示词模板：
   - `seo-technical` 模板 → 技术审计（robots、sitemap、meta 完整性）
   - `seo-content` 模板 → E-E-A-T 评分
4. 返回审计结果 JSON（分数 + 问题列表 + 修复建议）

**前端：**

1. SEO 管理页面新增"审计"按钮
2. 审计报告展示：
   - 总分（0-100）
   - 技术 SEO 评分
   - 内容质量评分
   - 问题列表（Critical / High / Medium / Low）
   - 一键修复按钮（自动修改 SEO 字段）

**Claude-SEO 资产复用：**
- 提取 `skills/seo-technical/SKILL.md` 的系统提示词 → 作为 Skill 模板存入数据库
- 提取 `skills/seo-content/SKILL.md` 的 E-E-A-T 评分逻辑 → 作为 Skill 模板

---

### 阶段三：Google APIs 集成（3-5 天）

**目标：** 接入真实的 Google 搜索数据，替代纯 AI 推测。

**集成范围（按优先级）：**

| API | 用途 | 工作量 |
|---|---|---|
| **PageSpeed Insights v5** | 真实 Core Web Vitals 数据 | 低 |
| **CrUX** | 字段数据历史趋势 | 中 |
| **Search Console** | 搜索展现、点击、CTR、排名 | 高（需 OAuth） |
| **Indexing API** | 主动提交索引 | 低 |

**实现方式：**

1. **设置页面** `SettingsSEO.jsx` 新增 Google API 配置：
   - Service Account JSON 上传
   - Property 选择（sc-domain:xxx.com）
   - API Key（用于 PageSpeed/CrUX）

2. **后端** 新增 `controller/seo_google.go`：
   ```go
   GET /api/seo/google/pagespeed?url=xxx    → Lighthouse 分数 + CWV
   GET /api/seo/google/crux?url=xxx         → 真实用户 CWV 字段数据
   GET /api/seo/google/gsc/queries          → 搜索查询表现
   POST /api/seo/google/index               → 主动提交 URL 索引
   ```

3. **前端** SEO 管理页面新增"Google 数据"Tab：
   - PageSpeed 分数仪表盘（LCP / INP / CLS）
   - 搜索查询列表（关键词、展现、点击、CTR、排名）
   - 索引状态

**Claude-SEO 资产复用：**
- `skills/seo-google/SKILL.md` 的 API 调用逻辑 → 转化为 Go HTTP 客户端
- `scripts/pagespeed_check.py` → 参考 PSI v5 API 调用方式
- `scripts/gsc_query.py` → 参考 GSC API 调用方式
- `scripts/google_report.py` → 参考报告生成逻辑

---

### 阶段四：SEO 报告生成（2-3 天）

**目标：** 一键生成专业 SEO 报告（PDF / HTML）。

**实现方式：**

1. **后端** `service/seo_report.go`：
   - 聚合数据：AI 审计结果 + Google API 数据 + 现有 SEO 字段
   - 生成 Markdown 报告
   - 调用 AI 转化为专业排版 HTML

2. **前端** SEO 管理页面新增"生成报告"按钮：
   - 选择报告类型：单页审计 / 全站概览
   - 下载 PDF / 在线预览 HTML

**Claude-SEO 资产复用：**
- `scripts/google_report.py` 的报告结构 → 转化为 Go 模板
- `skills/seo-audit/SKILL.md` 的报告格式 → 作为报告模板

---

## 四、实施建议

### 推荐优先级

```
阶段一（多语言 SEO） → 立即实施，1 天完成，零风险
阶段二（SEO 审计）   → 次优先，2-3 天，提升管理能力
阶段三（Google APIs）→ 按需实施，3-5 天，需要 Google Cloud 账号
阶段四（SEO 报告）   → 最后实施，2-3 天，依赖阶段二/三数据
```

### 为什么不直接集成 claude-seo CLI？

| 方案 | 问题 |
|---|---|
| 直接集成 claude-seo Python 脚本 | 需要 Python 运行时 + 额外依赖（WeasyPrint、Playwright），与 Go 后端冲突 |
| 作为 Claude Code 插件使用 | 只适合开发环境，无法服务下游 Web/API 消费者 |
| **提取提示词 + 用现有 AI 接口调用** | ✅ 最符合项目架构，复用现有 Skill/AI 基础设施 |

### 资产复用清单

| Claude-SEO 资产 | 复用方式 |
|---|---|
| `skills/seo-*/SKILL.md` 系统提示词 | 提取为 Skill 模板，存入数据库 |
| `skills/seo-google/references/` API 文档 | 参考实现 Go 版 HTTP 客户端 |
| `scripts/*.py` | 参考 API 调用逻辑和参数格式 |
| `agents/seo-*.md` 子代理定义 | 转化为更细粒度的 Skill 模板 |

---

## 五、总结

> **不集成 claude-seo CLI，而是提取其知识资产（提示词模板、审计逻辑、API 集成方式），融入现有 SEO 系统。**

现有系统骨架完整，缺的只是：
1. **多语言内容填充**（字段已有，逻辑未接）
2. **AI 审计能力**（用 claude-seo 提示词模板）
3. **真实数据接入**（Google APIs）

建议先做**阶段一**（多语言 SEO），这是立即可见效的。需要我现在开始做吗？
