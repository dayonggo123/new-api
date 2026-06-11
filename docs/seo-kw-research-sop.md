# SEO Keyword Research SOP（海外市场版）

## v2.0 | Target: Global/Overseas Market | Platform: harse.tv (New-API)

> **适用场景：** harse.tv 海外 SEO 关键词研究、内容规划、竞品对标
> **目标搜索引擎：** Google Global（非百度/国内）
> **核心差异化：** Node Canvas（节点画布）+ Multi-language i18n（12 语种）+ GEO Structured Content

---

## 一、SOP 概述

### 1.1 研究对象：harse.tv

| 维度 | 现状 | SEO 资产 |
|------|------|----------|
| **产品定位** | HarseTV 分镜助手 — 节点画布 AI 创意工作台 | Title: "HarseTV 分镜助手 — 节点画布 AI 创意工作台" |
| **核心技术** | Node-based canvas（类似 ComfyUI）AI video creation | 差异化蓝海词：`node canvas video`, `AI workflow builder` |
| **内容系统** | New-API 提示词库 + 文章教程系统 | 已有 `/prompts`, `/articles` 页面，支持 slug、SEO 字段 |
| **多语言** | 12 语种 i18n（en/ja/ko/de/fr/es/pt/ru/ar/hi/tr/id） | **海外市场核心优势** — 竞品几乎全英文单语 |
| **结构化内容** | GEO blocks（Prompt 3 区块 + 文章 5 语义块） | Schema: Article, FAQPage, WebPage |
| **提示词库** | 支持按模型/分类组织，公开 API | 可对标 VideoPrompt.app / OpenPromptLib |
| **当前流量** | ≈ 0 organic traffic（仅工具页，零内容资产） | **最大机会 = 内容填充** |

### 1.2 目标市场：Global Overseas（Google 生态）

| 市场层级 | 语言 | 优先级 | 说明 |
|----------|------|--------|------|
| **Tier-1 核心市场** | English (US/UK/CA/AU) | P0 | 最大搜索量，最高 CPC，竞争最激烈 |
| **Tier-2 增长市场** | Japanese, Korean, German, French | P1 | 高付费意愿，竞争低于英语圈 |
| **Tier-3 长尾市场** | Spanish, Portuguese, Russian, Arabic, Hindi, Turkish, Indonesian | P2 | 低竞争蓝海，i18n 护城河 |

### 1.3 海外竞品地图（已确认）

#### A. 提示词库类（Direct Competitors — Prompt Library）

| 竞品 | URL | 规模 | 模型覆盖 | 多语言 | 内容策略 | 弱点 |
|------|-----|------|----------|--------|----------|------|
| **VideoPrompt.app** | videoprompt.app | 500+ prompts | 10+ models | ❌ 仅 EN | 模型页+分类页+博客+FAQ | 无 Schema；标题写 50+ 实际只有部分；无多语言 |
| **OpenPromptLib** | openpromptlib.com | 4,379 video prompts | Sora/Runway/Pika/Kling/Seedance | ❌ 仅 EN | 画廊模式+每周精选+API | Seedance 偏重；缺 Runway Gen-3/Pika/Sora；无主题分类 |
| **PromptBase** | promptbase.com | 6,600+ video prompts | Sora/Veo/+ | ❌ 仅 EN | 付费市集模式 | 每条收费；免费用户只能浏览 |
| **CinePrompt** | cineprompt.pro | 1,000+ prompts | Sora2/Veo3/Kling/Seedance/Pika/Runway | ❌ 仅 EN | 免费浏览+生成器 | 新站，DR 低 |
| **Tona.AI** | tonaai.io | 17 prompts (文章) | Kling3/Veo/Sora | ❌ 仅 EN | 博客文章+工具平台 | 标题承诺 50+ 实际只有 17；无 H3；零外链 |
| **ImageToVideo** | imaginetovideo.com | 1,000+ prompts | Sora2/Veo3/Kling3 | ❌ 仅 EN | 提示词画廊 | 较新 |

#### B. AI 视频工具类（Indirect Competitors — Tool Comparison)

| 竞品 | URL | 定位 | SEO 策略 |
|------|-----|------|----------|
| **RHTV** | rhtv.runninghub.cn | 节点画布视频创作 | PR 曝光强；灵感库 UGC；百度百科词条；❌ 仅中文 |
| **LibTV** | liblib.tv / tv.liblib.art | AI 视频（背靠 liblib.art 社区） | Creator+Agent 双入口；价格内卷 76%；❌ 仅中文 |
| **HappyHorse** | happy-horse.tv | AI Video Generator | 分镜助手定位；提示词指南；提示词实验室 |
| **Alignify** | alignify.co | 节点式工具评测 | Canvas Video 专题页；深度评测文章 |

### 1.4 harse.tv 的 SEO 护城河分析

| 能力 | harse.tv | RHTV | LibTV | VideoPrompt.app | OpenPromptLib |
|------|----------|------|-------|-----------------|---------------|
| Schema 标记 | ✅ Article+FAQ+WebPage | ❌ | ❌ | ❌ | ❌ |
| 多语言 i18n | ✅ 12 语种 | ❌ ZH only | ❌ ZH only | ❌ EN only | ❌ EN only |
| 文章/教程系统 | ✅ CMS + GEO blocks | ❌ | ❌ | ✅ Blog | ✅ Weekly |
| 提示词库 | ✅ 公开 API | ⚠️ 封装内部 | ❌ 无 | ✅ 500+ | ✅ 4,379 |
| 节点画布 | ✅ 核心产品 | ✅ | ❌ | ❌ | ❌ |
| **结论** | **技术 SEO 领先** | 产品强 | 社区强 | 内容量强 | 数据量强 |

---

## 二、标准流程（8 步法 — 海外优化版）

## Step 1：种子词收集（Seed Keywords）

### 1.1 从 harse.tv 产品能力提取

基于实际产品功能，种子词分为 **4 大维度**：

```
┌─────────────────────────────────────────────────────────┐
│              harse.tv 种子词体系                          │
├─────────────┬───────────────────┬───────────────────────┤
│ 维度 A: 品类 │ 维度 B: 技术      │ 维度 C: 场景          │
│ (What)       │ (How)             │ (Use Case)            │
├─────────────┼───────────────────┼───────────────────────┤
│ AI video     │ Node canvas       │ YouTube shorts        │
│ creation     │ Node-based        │ Marketing video       │
│ AI video     │ editor           │ Product demo          │
│ generator    │ ComfyUI           │ Social media content   │
│ Text to      │ Workflow          │ Storyboard /          │
│ video        │ Visual AI         │ 分镜                    │
│ AI video     │ pipeline          │ Film pre-visualization │
│ maker        │ AI storyboard     │ Explainer video       │
│              │ maker            │                       │
├─────────────┴───────────────────┴───────────────────────┤
│ 维度 D: 模型（Model-Specific Keywords — 最高转化意图）   │
├─────────────────────────────────────────────────────────┤
│ Sora prompt / Sora 2 / OpenAI Sora                      │
│ Kling prompt / Kling 3.0 / Kling AI                     │
│ Veo prompt / Veo 3 / Veo 3.1 / Google Veo              │
│ Runway prompt / Runway Gen-4 / Runway ML               │
│ Pika prompt / Pika 2.0                                 │
│ Seedance / Jimeng 即梦                                  │
│ Luma / Dream Machine / Wan / Hailuo                   │
└─────────────────────────────────────────────────────────┘
```

### 1.2 从竞品 Title/Meta/H1 提取的实际关键词

通过 WebFetch 竞品页面提取到的**真实在用关键词**：

| 来源 | 提取的关键词 |
|------|-------------|
| **VideoPrompt.app** | AI Video Prompt, Video Prompt Library, Sora Prompt Generator, Sora2 Video Generator, Veo3 Prompt, Prompt Generator, AI Video Generation Prompt Collection, curated prompts |
| **OpenPromptLib** | AI Video Prompts, curated prompts for AI video generation, Seedance 2.0, Kling 3.0, PixVerse |
| **Tona.AI** | Best AI Video Prompts, Cinematic landscape prompts, Product showcase, Human portrait, Nature wildlife, Abstract artistic, Architecture urban, Prompt engineering tips |
| **CinePrompt** | Free AI Video Prompts, AI Video Prompt Generator, cinematic prompts |
| **HappyHorse** | AI Video Generator, AI Video Prompt Guide, AI Video Prompt Lab, Video Prompt Generator |
| **harse.tv 首页** | HarseTV 分镜助手, 节点画布, AI 创意工作台 |

### 1.3 种子词清单输出模板

| 种子词 | 维度 | 语言 | 来源 | 搜索引擎 | 备注 |
|--------|------|------|------|----------|------|
| AI video prompt library | A品类 | EN | VideoPrompt.app title | Google Global | 核心大词 |
| Sora prompt template | D模型 | EN | 竞品聚类 | Google | 高转化 |
| node canvas video editor | B技术 | EN | harse.tv 产品 | Google | **蓝海差异化** |
| AI storyboard creator | C场景 | EN | 产品功能 | Google | 分镜=storyboard |
| Kling 3.0 prompt examples | D模型 | EN | Tona.AI 文章 | Google | 长尾高意图 |

### 检查点
- [ ] 种子词 ≥ 15 个，覆盖 A/B/C/D 四个维度
- [ ] 至少包含 5 个模型专属词（Dimension D）
- [ ] 至少包含 3 个节点画布/工作流相关词（Dimension B — 差异化）
- [ ] 全部使用**英文**（海外市场为主）

---

## Step 2：关键词扩展（Keyword Expansion）

### 2.1 扩展方法矩阵

| 方法 | 操作 | 适用场景 | 预期产出 |
|------|------|----------|----------|
| **A. 工具扩展** | Ahrefs Keywords Explorer / SEMrush Magic Tool | 批量获取搜索量和 KD | 200-500 个词 |
| **B. SERP 挖掘** | Google Related Searches + People Also Ask + Autocomplete | 发现真实用户搜索行为 | 50-100 个长尾词 |
| **C. 竞品逆向** | Ahrefs Site Explorer → Top Pages of 竞品 | 窃取竞品已验证有效的词 | 100-200 个词 |
| **D. 模型词矩阵** | [Model] × [Action] × [Format] 笛卡尔积 | 系统化覆盖所有模型组合 | 300+ 个词 |

### 2.2 方法 D：模型词矩阵（harse.tv 专用 — 最重要）

这是 harse.tv 最大的内容机会。每个模型 × 每个动作 × 每个格式 = 一篇独立内容：

```
[MODEL] × [ACTION] × [FORMAT] = CONTENT PIECE

Models (10):  Sora, Sora2, Kling, Kling3, Veo, Veo3, Runway, Pika, Seedance, Luma
Actions (8):   prompt, template, guide, tutorial, examples, best practices, tips, comparison
Formats (6):   for beginners, for marketing, cinematic, short-form, product video, anime/style

→ 10 × 8 × 6 = 480 potential content pieces (long-tail keywords)
```

**实际高优先级示例（P0）：**

| 关键词 | 模型 | 意图 | 预估搜索量 | 竞争度 | 推荐内容类型 |
|--------|------|------|------------|--------|-------------|
| `Sora 2 prompt template` | Sora | Commercial | 2K-5K/月 | 中 | Prompt 详情页 |
| `Kling 3.0 prompt examples` | Kling | Informational | 3K-8K/月 | 中 | 提示词集合页 |
| `best Veo 3 prompts for marketing` | Veo | Commercial | 500-2K/月 | 低 | 专题列表页 |
| `Runway Gen-4 prompt guide for beginners` | Runway | Informational | 1K-3K/月 | 中 | 教程文章 |
| `cinematic AI video prompts 2026` | All | Informational | 5K-15K/月 | 中高 | Pillar 文章 |
| `AI video prompt for YouTube shorts` | All | Informational | 2K-5K/月 | 低 | 场景化列表 |
| `product showcase AI video prompt template` | All | Commercial | 1K-3K/月 | 低 | 模板详情页 |
| `node-based AI video creation tutorial` | Tech | Informational | 500-2K/月 | **极低** | 教程（差异化） |

### 2.3 Google SERP 挖掘实操

对每个种子词，执行以下操作并记录：

```bash
# 1. Google 搜索 + 记录底部 Related Searches
# 例：搜索 "AI video prompt library" 底部出现：
# - "ai video prompt generator free"
# - "best ai video prompts for sora"
# - "how to write ai video prompts"
# - "ai video prompt examples cinematic"

# 2. 展开 People Also Ask（PAA）
# 例："What is the best prompt for AI video generation?"
#     "How do I prompt AI to create a video?"
#     "Which AI video generator has the best prompts?"

# 3. 查看 Autocomplete 建议
# 在搜索框输入种子词，记录下拉建议的前 10 个
```

### 2.4 输出：扩展关键词主表

| # | Keyword | Monthly Vol | KD | CPC | Intent | Model Tag | Category | Source | ROI Score |
|---|---------|-------------|----|-----|--------|-----------|----------|--------|-----------|
| 1 | AI video prompt library | 5K-10K | 45 | $2.80 | Commercial | Multi | Library | Tool(A) | 85 |
| 2 | Sora 2 prompt examples | 3K-8K | 35 | $1.50 | Info | Sora | Template | Matrix(D) | 92 |
| 3 | Kling 3.0 prompt guide | 2K-5K | 28 | $1.20 | Info | Kling | Tutorial | Matrix(D) | 90 |
| 4 | best AI video prompts 2026 | 8K-15K | 55 | $2.20 | Commercial | Multi | List | SERP(B) | 82 |
| 5 | node canvas AI video tool | 500-2K | **12** | $1.80 | Commercial | Tech | Product | Seed(A) | **88** |
| ... | ... | ... | ... | ... | ... | ... | ... | ... | ... |

### 检查点
- [ ] 总扩展词 ≥ 300 个
- [ ] 模型词矩阵覆盖 ≥ 6 个主流模型
- [ ] 包含 ≥ 20 个「节点画布/工作流」相关词（差异化）
- [ ] 每个词有搜索量和 KD 数据来源标注

---

## Step 3：搜索意图分类（Search Intent Classification）

### 3.1 海外市场意图分类标准

| Intent Type | 标识 | 典型模式 | Content Format | Conversion Value | Content Example |
|-------------|------|----------|----------------|------------------|-----------------|
| **Informational** | 📖 学习 | `what is`, `how to`, `guide`, `tutorial`, `for beginners` | Blog post, Tutorial, How-to guide | Low-Med (top of funnel) | "How to write Sora prompts" |
| **Commercial Investigation** | 🔍 比较 | `best`, `vs`, `comparison`, `review`, `top 10`, `alternatives` | Comparison, Review, Listicle | High (decision stage) | "Sora vs Kling vs Veo 2026" |
| **Transactional** | 💡 行动 | `generator`, `free`, `template`, `download`, `prompt examples` | Tool page, Template gallery, Prompt detail | Very High (ready to act) | "Free Sora 2 prompt templates" |
| **Navigational** | 🎯 导航 | brand name + feature, `login`, `official` | Homepage, Feature page | Medium (brand intent) | "harse.tv prompt library" |

### 3.2 AI Video 领域特有意图细分

本领域有一个重要的**混合意图**现象需要特别注意：

| 关键词示例 | 主意图 | 次意图 | 策略建议 |
|------------|--------|--------|----------|
| `free AI video prompts` | Transactional（要免费资源） | Informational（想学习怎么写） | **先给提示词（交易），再教技巧（信息）** → Prompt Detail Page 内嵌 Geo Blocks |
| `best AI video generator 2026` | Commercial Investigation（比工具） | Navigational（找具体网站） | 写对比文，自然引入 harse.tv 作为推荐之一 |
| `Sora prompt template download` | Transactional（要下载） | Commercial（可能付费） | 免费提供模板 → 收集邮箱 → 推送新模板通知 |
| `node canvas vs traditional editing` | Commercial Investigation | Informational | 教程型对比文 → 展示 harse.tv 节点画布优势 |

### 3.3 意图分布目标

对于 harse.tv 的内容策略，理想的意图分布应该是：

```
Informational:     40%  ← 流量入口（教程/指南/什么是...）
Commercial Inv.:   30%  ← 决策影响（对比/评测/最佳...）
Transactional:     25%  ← 直接转化（模板/生成器/免费下载）
Navigational:       5%  ← 品牌词（harse.tv 相关）
```

### 检查点
- [ ] 每个 keyword 有意图标注
- [ ] Top 50 高搜索量词全部经过 Google SERP 手动核查
- [ ] 意图分布接近 40/30/25/5 目标比例

---

## Step 4：难度与商业价值评分（Difficulty & Business Value Scoring）

### 4.1 难度评估（海外版）

| 维度 | 数据来源 | 权重 | 说明 |
|------|----------|------|------|
| **KD (Keyword Difficulty)** | Ahrefs KD / Semrush % Difficulty | 40% | 核心指标 |
| **SERP Domain Authority** | Moz DA / Ahrefs DR of top 10 | 30% | 前 10 平均 DR > 50 = 难 |
| **Content Depth** | Manual review of top 10 results | 20% | 前 10 平均字数 < 1500 = 机会 |
| **Schema/Technical SEO** | Check if top 10 use structured data | 10% | 竞品不用 Schema = 我们有机会用 Schema 超车 |

**关键发现：AI Video Prompt 领域的竞品几乎都不用 Schema！**

这意味着：
- 即使 DR 较低，**高质量 + Schema + GEO 结构化内容可以快速超越竞品**
- harse.tv 已经有 Article + FAQPage + WebPage Schema → **技术 SEO 天然领先**

### 4.2 商业价值评分（harse.tv 定制版）

| 维度 | 评分标准 | 权重 | harse.tv 特殊考量 |
|------|----------|------|-------------------|
| **Product Match** | 1-5 (5=描述 harse.tv 核心功能) | 30% | 「节点画布」「分镜」相关词 = 5 分 |
| **Intent Strength** | Trans(5) > Comm(4) > Info(3) > Nav(2) | 25% | Template/Generator 类词分数更高 |
| **CPC** | >$3 = 5, $1-3 = 3, <$1 = 2 | 15% | AI video 领域 CPC 普遍偏高 |
| **Multi-language Potential** | 该词在其他 11 种语言中是否有搜索量 | 15% | **harse.tv 独有优势 — i18n 可以放大 10x 流量** |
| **Node Canvas Relevance** | 是否与「节点/画布/工作流/ComfyUI」相关 | 15% | **差异化词加分** — 竞品几乎不覆盖 |

### 4.3 ROI 评分公式（v2.0 海外增强版）

```
ROI Score = (
  Search_Volume_Norm × 0.20 +
  Business_Value × 0.35 +
  MultiLang_Bonus × 0.15 +
  NodeCanvas_Diff_Bonus × 0.10 -
  Difficulty_Penalty × 0.20
) × 100

其中：
- MultiLang_Bonus: 如果该词在 ≥ 5 个 Tier-1/2 语言中有对应翻译词 = +15 分
- NodeCanvas_Diff_Bonus: 如果是「节点/画布/ComfyUI/工作流」相关词 = +10 分
- Difficulty_Penalty: KD > 60 时额外扣分
```

### 4.4 简化优先级矩阵

| Search Volume | Difficulty | Business Value | Multi-lang | Node Canvas | ROI Grade | Action |
|---------------|------------|---------------|------------|-------------|-----------|--------|
| High (>5K/mo) | Low (<30) | High | ✅ | ✅ | ⭐⭐⭐⭐⭐ | **Do NOW** |
| High (>5K/mo) | Med (30-50) | High | ✅ | ✅ | ⭐⭐⭐⭐ | Batch 1 |
| Med (1K-5K) | Low (<30) | Med-High | ✅ | ✅ | ⭐⭐⭐⭐ | Batch 1 |
| Med (1K-5K) | Med | High | ❌ | ❌ | ⭐⭐⭐ | Batch 2 |
| Low (<1K) | Low | Any | ✅ | ✅ | ⭐⭐⭐ | Batch 2 (Long-tail farm) |
| Any | High (>60) | Low | ❌ | ❌ | ⭐ | Skip or Long-term |

### 检查点
- [ ] 高 ROI 词（⭐⭐⭐⭐+）≥ 25 个
- [ ] 至少 10 个词获得 Multi-language Bonus
- [ ] 至少 5 个词获得 Node Canvas Differentiation Bonus
- [ ] 所有评分数据有来源可追溯

---

## Step 5：主题聚类策略（Topic Clustering — Pillar & Cluster）

### 5.1 harse.tv 主题簇架构（基于实际能力设计）

```
┌──────────────────────────────────────────────────────────────┐
│                  harse.tv Topic Cluster Map                   │
│                  （海外市场 + 多语言版本）                      │
└──────────────────────────────────────────────────────────────┘

📦 PILLAR A: AI Video Prompt Library（提示词库 — 流量核心）
│  Pillar Keyword: "AI Video Prompt Library" (Vol: 5K-10K/mo)
│  Page: /prompts (已有页面，需填充内容 + Schema + i18n)
│
├── Cluster A1: Sora Prompts & Templates (Vol: 3K-8K)
│   ├── /prompts/sora          → Sora prompt list (model page)
│   ├── /prompts/sora/beginner → Sora prompts for beginners
│   ├── /prompts/sora/cinematic → Cinematic Sora prompt templates
│   └── /prompts/sora/marketing → Sora marketing video prompts
│
├── Cluster A2: Kling Prompts & Guides (Vol: 2K-5K)
│   ├── /prompts/kling
│   ├── /prompts/kling-3        → Kling 3.0 specific (NEW MODEL BOOST)
│   └── /prompts/kling/tips     → Kling prompt engineering tips
│
├── Cluster A3: Veo / Veo3 Prompts (Vol: 1K-3K) 🆕
│   ├── /prompts/veo3           → Ride the "Veo 3" new model wave
│   └── /prompts/google-veo-vs-sora → Cross-model comparison
│
├── Cluster A4: Runway / Pika / Others (Vol: 2K-4K combined)
│   ├── /prompts/runway
│   ├── /prompts/pika
│   └── /prompts/seedance       → Chinese model going global
│
├── Cluster A5: Prompt by Use Case (Vol: 3K-7K combined) 🔥
│   ├── /prompts/youtube-shorts → HUGE search volume
│   ├── /prompts/product-demo   → High commercial value
│   ├── /prompts/social-media   → TikTok/Instagram/YouTube
│   └── /prompts/anime-style    → Viral niche
│
└── Cluster A6: AI Video Prompt Generator (Tool page)
    └── /tools/prompt-generator → Interactive tool (transactional)

📦 PILLAR B: Node Canvas AI Video Creation（节点画布 — 差异化核心）
│  Pillar Keyword: "Node-Based AI Video Creation Tool" (Vol: 500-2K, LOW KD!)
│  Page: /guide/node-canvas-video (需新建 — 蓝海 Pillar Page)
│
├── Cluster B1: What is Node Canvas Video? (Info)
│   → "what is node-based video editing"
│   → "node canvas vs timeline editing"
│
├── Cluster B2: Node Canvas Tutorials (Info)
│   → "node canvas AI video tutorial for beginners"
│   → "ComfyUI workflow for video creation"
│   → "how to build AI video pipeline"
│
├── Cluster B3: Comparisons (Commercial)
│   → "node canvas video tools comparison 2026"
│   → "harse vs RHTV vs Comfy"  🎯 用竞品品牌词截流
│
└── Cluster B4: Workflows & Templates (Transactional)
    → "AI video workflow templates"
    → "storyboard AI workflow examples"

📦 PILLAR C: AI Video Tools Comparison（工具横评 — 截流利器）
│  Pillar Keyword: "Best AI Video Generators 2026" (Vol: 8K-15K)
│  Page: /article/best-ai-video-generators-2026 (需新建)
│
├── Cluster C1: Full Comparison Articles
│   → "Sora vs Kling vs Veo vs Runway 2026"
│   → "best AI video generators for YouTube"
│   → "free AI video generators comparison"
│
├── Cluster C2: Model Deep-Dives
│   → "Sora 2 review: what's new"
│   → "Kling 3.0 vs Kling 2: what changed"
│   → "Veo 3.1 complete guide"
│
└── Cluster C3: By Use Case
    → "best AI video tool for marketing videos"
    → "best AI video tool for beginners"
    → "best AI video tool for animation"

📦 PILLAR D: AI Video Prompt Engineering（提示词工程 — 权威性建设）
│  Pillar Keyword: "AI Video Prompt Engineering Guide" (Vol: 2K-5K)
│  Page: /article/ai-video-prompt-engineering-guide
│
├── Cluster D1: Prompt Writing Techniques
│   → "how to write AI video prompts"
│   → "AI video prompt structure formula"
│   → "camera movement terms for AI video prompts"
│
├── Cluster D2: Model-Specific Prompting
│   → "Sora prompting techniques"
│   → "Kling prompting best practices"
│   → "Runway prompt syntax guide"
│
└── Cluster D3: Advanced Techniques
    → "AI video prompt engineering for professionals"
    → "consistency in AI video generation"
    → "negative prompts for AI video"
```

### 5.2 内链策略（Internal Linking Strategy）

```
Pillar Page (Harse.tv 特色)
    │
    ├─→ Cluster Pages (模型/分类/场景)
    │       │
    │       ├─→ Prompt Detail Pages (/prompt/{slug})
    │       │       │
    │       │       ├─→ Geo Blocks: 适用场景 / 使用步骤 / 使用技巧
    │       │       ├─→ Related Prompts (同模型/同分类)
    │       │       └─→ CTA: Try on harse.tv
    │       │
    │       └─→ Article Pages (/article/{slug})
    │               │
    │               ├─→ Geo Blocks: 是什么 / 为什么 / 如何做 / 总结
    │               ├─→ FAQ Schema (auto from faq field)
    │               └─→ Related Articles
    │
    └─→ Cross-pillar links (Pillar A ↔ B ↔ C ↔ D)
            (e.g., Sora prompt page links to "Sora vs others" comparison)
```

### 5.3 多语言集群策略（i18n Topic Clustering）

harse.tv 的 12 语种 i18n 是核心护城河。每个主题簇都需要多语言版本：

```
English Pillar:  /prompts  →  /en/prompts (default)
Japanese:         /ja/prompts  →  "AI動画プロンプトライブラリ"
Korean:           /ko/prompts  →  "AI 동영상 프롬프트 라이브러리"
German:           /de/prompts  →  "AI-Videoprompt-Bibliothek"
French:           /fr/prompts  →  "Bibliothèque de prompts vidéo IA"
Spanish:          /es/prompts  →  "Biblioteca de prompts de video IA"
Portuguese:       /pt/prompts  →  "Biblioteca de prompts de vídeo IA"
Russian:          /ru/prompts  →  "Библиотека промптов для ИИ видео"
Arabic:           /ar/prompts  →  "مكتبة أوامر الفيديو بالذكاء الاصطناعي"
Hindi:            /hi/prompts  →  "AI वीडियो प्रॉम्प्ट लाइब्रेरी"
Turkish:          /tr/prompts  →  "AI video prompt kütüphanesi"
Indonesian:       /id/prompts  →  "Perpustakaan prompt video AI"

hreflang 标签实现:
<link rel="alternate" hreflang="en" href="https://harse.tv/prompts" />
<link rel="alternate" hreflang="ja" href="https://harse.tv/ja/prompts" />
... (12 variants total)
```

**注意：** 非英语市场的 AI video prompt 关键词竞争远低于英语市场。日语/韩语/德语的「AI 動画プロンプト」或 «KI-Videogenerierung» 类型关键词，KD 通常只有英语市场的 1/3 到 1/2。

### 检查点
- [ ] 4 个 Pillar Page 关键词确认，每个有明确 URL 规划
- [ ] 每个 Pillar 下有 ≥ 4 个 Cluster Page 规划
- [ ] 内链架构图完成（Pillar ↔ Cluster ↔ Detail 三层）
- [ ] 多语言 URL 规划覆盖 ≥ 8 个主要语言
- [ ] hreflang 实现方案已确认

---

## Step 6：竞品分析（Competitor Analysis — 海外深度版）

### 6.1 竞品 SEO 能力雷达图

```
                    Schema Markup
                        ▲
                   VideoPrompt  ●
                         ╲
          Content Volume  ╲  ● OpenPromptLib
                 ▲         ╲
                │    ●PromptBase
                │         ╲
  Multi-language │          ╲____● harse.tv (WE ARE HERE)
                │               
  ◄─────────────┼──────────────► Node Canvas
     Tech Innovation              Differentiation
     
     (harse.tv 在右上象限：技术差异化 + 多语言 + Schema，
      但需要大幅提升内容数量来扩大面积)
```

### 6.2 竞品关键词窃取矩阵（Keyword Gap Analysis）

| 竞品域名 | 其排名词示例 | 搜索量 | 我们的覆盖？ | 行动 |
|----------|-------------|--------|-------------|------|
| **videoprompt.app** | `AI video prompt library` | 5K-10K | ⚠️ 有页面但空 | **立即填充内容** |
| **videoprompt.app** | `Sora prompt generator` | 2K-5K | ❌ 无 | 建生成器页面 |
| **videoprompt.app** | `Veo3 prompt examples` | 1K-3K | ❌ 无 | 补 Veo3 提示词 |
| **openpromptlib.com** | `Seedance 2.0 prompts` | 1K-2K | ❌ 无 | 可选（中国模型） |
| **openpromptlib.com** | `curated AI video prompts` | 2K-4K | ⚠️ 有但无内容 | 填充 + 加 curated 策略说明 |
| **tonaai.io** | `best AI video prompts 2026` | 5K-10K | ❌ 无 | **写 Pillar 文章（他们只写了 17 条，我们可以写 50+）** |
| **cineprompt.pro** | `free AI video prompt generator` | 2K-4K | ❌ 无 | 建工具页 |
| **promptbase.com** | `Sora video prompts buy` | 1K-3K | ❌ 无 | **我们免费提供 → 截流付费用户** |
| **rhtv.runninghub.cn** | `节点画布视频` (ZH) | 3K-5K (Baidu) | ⚠️ 有产品无内容 | **英文版节点画布内容 = 蓝海** |
| **happy-horse.tv** | `AI video prompt guide` | 2K-4K | ❌ 无 | 写更好的 guide |

### 6.3 竞品内容弱点利用表

| 竞品 | 弱点 | harse.tv 如何超越 |
|------|------|------------------|
| **Tona.AI** | 标题说 50+ 实际 17 条；无 H3；零外链 | 做**真正的 50+** 提示词合集 + 完整 H2/H3 + 内链 + Schema |
| **VideoPrompt.app** | 无 Schema；无多语言；标题虚高 | **全页面 Schema + 12 语种 i18n + 真实内容量** |
| **OpenPromptLib** | 缺少分类导航；偏重 Seedance | **完善分类系统 + 覆盖全部主流模型** |
| **PromptBase** | 每条收费 | **同等质量完全免费** → 以此作为 USP 营销 |
| **RHTV / LibTV** | 仅中文；零 SEO 内容资产 | **英文 + Schema + 内容系统** → 同类产品全球 SEO 第一人 |
| **All 竞品** | 都没有「节点画布」相关内容 | **占领这个蓝海关键词家族** |

### 检查点
- [ ] 分析 ≥ 6 个竞品（至少 3 个直接竞品 + 3 个间接竞品）
- [ ] 竞品关键词 Gap 清单 ≥ 20 个词
- [ ] 每个竞品的「可利用弱点」已识别
- [ ] harse.tv 的差异化切入点已明确

---

## Step 7：内容缺口分析（Content Gap Analysis）

### 7.1 内容缺口优先级矩阵（基于 harse.tv 实际情况）

#### P0 — 本月必须做（高搜索量 + 低难度 + 我们能做好）

| # | 关键词缺口 | 搜索量 | KD | 为什么我们能赢 | 建议内容 | 预计工时 |
|---|-----------|--------|----|---------------|----------|----------|
| P0-1 | `AI video prompt library` | 5K-10K | 45 | 已有页面+Schema+i18n，只需填内容 | 填充 200+ 提示词到 /prompts | 3 天 |
| P0-2 | `best AI video prompts 2026` | 5K-15K | 55 | 竞品只有 17 条，我们可以做 50+ 真实测试过的 | Pillar 文章 + Geo Blocks | 2 天 |
| P0-3 | `node canvas AI video tool` | 500-2K | **~12** | **全球几乎无人做 SEO 内容** | Pillar 文章 + 教程系列 | 2 天 |
| P0-4 | `Sora 2 prompt examples` | 3K-8K | 35 | 有 i18n+Schema 优势 | 提示词详情页批量创建 | 2 天 |
| P0-5 | `free AI video prompt generator` | 2K-4K | 30 | 我们有技术能力自建工具 | 工具页面 (/tools/generator) | 3 天 |

#### P1 — 下个月要做（中等搜索量 + 中等难度）

| # | 关键词缺口 | 搜索量 | KD | 建议内容 | 预计工时 |
|---|-----------|--------|----|----------|----------|
| P1-1 | `Kling 3.0 prompt guide` | 2K-5K | 28 | 深度教程 + 提示词模板 | 2 天 |
| P1-2 | `Veo 3 prompt examples` | 1K-3K | 25 | 新模型红利窗口期 | 1 天 |
| P1-3 | `AI video tools comparison 2026` | 8K-15K | 58 | 含 harse.tv 的公正横评 | 3 天 |
| P1-4 | `AI video prompts for YouTube shorts` | 2K-5K | 32 | 场景化提示词合集 | 1 天 |
| P1-5 | `ComfyUI video workflow tutorial` | 3K-6K | 35 | 节点画布教程（关联到 harse.tv） | 2 天 |
| P1-6 | `how to write AI video prompts` | 2K-4K | 38 | Prompt Engineering 101 | 2 天 |

#### P2 — 持续填充（长尾词农场 + 多语言扩展）

| # | 关键词方向 | 数量 | 策略 |
|---|-----------|------|------|
| P2-1 | 模型×场景 长尾组合 | ~100 个 | 批量生成 Prompt Detail Pages |
| P2-2 | 非 English 内容 | 11 语言 × 核心页面 | 翻译 P0/P1 内容到日/韩/德/法/西等 |
| P2-3 | 节点画布专业术语 | ~20 个 | `workflow builder` / `visual programming` / `node editor` 等 |

### 7.2 harse.tv 内容缺口总览图

```
                    竞品已覆盖的内容区域
                    ████████████████░░
                    █ VideoPrompt █░░░ (500+ prompts, no schema)
                    ████████████████░░
                    ██ OpenPromptLib ░░ (4379 prompts, no categories)
                    ██████████████░░░░
                    ███ Tona.AI ░░░░░░░ (17 prompts only!)
                    ████████████████░░
                    
     harse.tv 当前:  ░░░░░░░░░░░░░░░░░  (≈ 0 content assets)
     
     目标状态 (3 months):
     ████████████████████████  ← 匹配竞品内容量
     ██████████░░░░░░░░░░░░░░░  ← + Schema (竞品没有)
     ██████████░░░░░░░░░░░░░░░  ← + 12-Language i18n (竞品没有)
     ██████████░░░░░░░░░░░░░░░  ← + Node Canvas content (竞品没有)
     ██████████░░░░░░░░░░░░░░░  ← + GEO Structured Blocks (竞品没有)
```

### 检查点
- [ ] P0 缺口 ≥ 5 个，P1 ≥ 6 个，P2 方向明确
- [ ] 每个缺口有「为什么我们能赢」的理由
- [ ] 每个缺口有预计工时和负责角色
- [ ] 内容缺口总览图可视化完成

---

## Step 8：研究报告输出（Research Report Output）

### 8.1 报告结构

```
reports/seo-kw-research-{project}-{date}.md

1. Executive Summary (1 page)
   ├── 3 key findings
   ├── Top 5 action items with ROI scores
   ├── Traffic potential forecast (3/6/12 month projections)
   └── Resource requirements

2. High-ROI Keyword Master List (CSV attachment)
   ├── 300+ keywords with full data
   ├── Sorted by ROI Score DESC
   └── Columns: Keyword, Vol, KD, CPC, Intent, Model, Category, ROI, Priority

3. Topic Cluster Architecture
   ├── 4 Pillar Pages with URLs
   ├── 24+ Cluster Pages mapped
   ├── Internal linking diagram
   └── i18n rollout plan (12 languages)

4. Competitive Intelligence Brief
   ├── 6+ competitors analyzed
   ├── Keyword gap matrix (their ranking → our gaps)
   ├── Weakness exploitation plan
   └── SWOT analysis per competitor

5. Content Gap Action Plan
   ├── P0: 5 items (this month)
   ├── P1: 6 items (next month)
   ├── P2: long-term roadmap
   └── Content brief template for writers

6. Technical SEO Recommendations
   ├── Schema implementation status
   ├── i18n hreflang checklist
   ├── Page speed optimization
   └── Sitemap update plan
```

### 8.2 交付物清单

| 交付物 | 格式 | 用途 |
|--------|------|------|
| **研究报告** | Markdown (.md) | 决策参考 + 团队对齐 |
| **关键词主表** | CSV / Excel | 内容团队直接使用 |
| **主题簇地图** | Mermaid 图 / 图片 | 内容架构可视化 |
| **内容缺口清单** | CSV / Excel (按优先级) | 排期执行 |
| **竞品情报卡** | Markdown (每竞品 1 页) | 快速参考 |
| **Content Brief 模板** | Markdown | 给写手的任务书 |
| **SOP 本文件** | Markdown | 下次研究复用 |

### 8.3 成功指标（KPIs）

| KPI | Baseline (Month 0) | Month 3 Goal | Month 6 Goal | Month 12 Goal |
|-----|--------------------|--------------|--------------|---------------|
| **Organic Traffic** | ~0/mo | 2K-5K/mo | 10K-25K/mo | 50K-100K/mo |
| **Indexed Pages** | <10 (only homepage/tool) | 50+ | 200+ | 500+ |
| **Ranking Keywords (Top 100)** | 0 | 20-50 | 100-300 | 500-1000 |
| **Featured Snippets** | 0 | 1-3 | 5-10 | 15-20 |
| **Backlinks (Referring Domains)** | 0 | 5-10 | 20-50 | 50-100 |
| **i18n Pages Indexed** | 0 | 20+ (EN+JA+KO+DE) | 100+ (8 langs) | 400+ (12 langs) |
| **Domain Rating (Ahrefs DR)** | 0 (new) | 10-15 | 20-30 | 35-45 |

---

## 三、质量控制检查清单（QC Checklist）

### 数据质量
- [ ] 搜索量数据来自 ≥ 1 个可信工具（Ahrefs/SEMrush/GSC）
- [ ] KD / DA 数据有 SERP 手动核查验证
- [ ] 商业价值经过产品团队确认
- [ ] 所有竞品数据来自**实际页面抓取**（非猜测）

### 覆盖完整性
- [ ] 种子词覆盖 A/B/C/D 四个维度
- [ ] 扩展词 ≥ 300 个
- [ ] 模型词矩阵覆盖 ≥ 6 个主流 AI 视频模型
- [ ] 「节点画布」差异化词 ≥ 10 个
- [ ] 意图分布接近 40/30/25/5

### 海外市场特性
- [ ] **所有核心关键词以英文为主**
- [ ] 目标搜索引擎 = **Google**（不是百度）
- [ ] 多语言机会已评估（哪些词值得翻译到哪些语言）
- [ ] CPC 数据以 USD 为基准
- [ ] 竞品分析包含**海外竞品**（不只是国内 RHTV/LibTV）

### 竞品深度
- [ ] 分析 ≥ 6 个竞品（含海外直接竞品）
- [ ] 每个竞品有**实际页面内容提取**（Title/H1/H2/Schema/内容量）
- [ ] 竞品关键词 Gap ≥ 20 个
- [ ] 竞品「可利用弱点」已列出

### 可执行性
- [ ] 关键词按 ROI 降序排列
- [ ] P0/P1/P2 三级优先级清晰
- [ ] 每个关键词有建议的**内容类型和 URL**
- [ ] 有工时估算和排期建议
- [ ] 有明确的 KPI 和时间节点预期

---

## 四、海外 SEO 特别注意事项

### 4.1 Google vs 百度的核心差异

| 维度 | Google（我们的目标） | 百度（不适用） |
|------|---------------------|---------------|
| **内容偏好** | 原创、深度、E-E-A-T | 数量、更新频率 |
| **技术权重** | Schema 标记很重要 | 几乎无影响 |
| **多语言** | hreflang 必须正确 | 单语言即可 |
| **速度要求** | Core Web Vitals 影响排名 | 影响较小 |
| **外链质量** | 权重极高 | 权重低（更看重域名年龄）|
| **移动优先** | Mobile-first indexing | PC 优先 |
| **AI 内容** | 允许但有质量门槛 | 相对宽松 |

### 4.2 harse.tv 的 Google SEO 优势清单

| 优势 | 状态 | 利用方式 |
|------|------|----------|
| Schema (Article + FAQPage + WebPage) | ✅ 已实现 | 每个 Prompt/Article 页面自动注入 |
| 12-Language i18n + hreflang | ✅ 已实现 | 占领非英语市场的长尾词 |
| Slug-based clean URLs | ✅ 已实现 | SEO 友好的 URL 结构 (`/prompt/{slug}`, `/article/{slug}`) |
| GEO Structured Content (3-block/5-block) | ✅ 已实现 | 增加 page depth → Google 偏好深度内容 |
| Sitemap auto-generation | ✅ 已实现 | 加速新页面索引 |
| Public API for prompts/articles | ✅ 已实现 | 支持第三方集成 → 自然获取外链 |
| Auto-translate queue (12 lang) | ✅ 已实现 | 批量翻译降低多语言成本 |
| `is_translated` status tracking | ✅ 已实现 | 运营效率提升 |

### 4.3 海外 SEO 常见陷阱

| 陷阱 | 说明 | 避免 |
|------|------|------|
| **Machine translation without editing** | Google 能识别机翻内容并降权 | 机翻后必须人工审校 |
| **Ignoring E-E-A-T** | YMYL（Your Money Your Life）领域需要权威性 | 添加作者信息、更新日期、引用来源 |
| **Keyword stuffing in i18n** | 直接翻译可能导致关键词不自然 | 每个语言市场单独做关键词适配 |
| **Ignoring mobile experience** | Google 移动优先索引 | 确保 SPA 在移动端正常渲染 |
| **No backlink strategy** | 再好的内容没外链接也难排名 | 提交到工具目录（Product Hunt / AlternativeTo / AI Tool Directory） |
| **Forgetting hreflang** | 多语言页面没有 hreflang → 重复内容问题 | 每个 i18n 页面必须有完整的 hreflang 标签组 |

---

## 五、工具速查表（Quick Reference）

### 推荐工具栈（海外 SEO）

| 用途 | 免费工具 | 付费工具（推荐） | 备注 |
|------|----------|-----------------|------|
| **关键词研究** | Google Keyword Planner, Google Trends, Ubersuggest (limited) | **Ahrefs** (Keywords Explorer), SEMrush (Keyword Magic) | Ahrefs 对海外关键词最准 |
| **SERP 分析** | 手动 Google 搜索 | Ahrefs SERP Checker, SEMrush SEO Content Template | 必须手动验证 KD 准确性 |
| **竞品分析** | SimilarWeb (basic), Open Page Explorer (Moz free tier) | **Ahrefs** Site Explorer, SEMrush Domain Overview | 导出竞品 Top Pages + Top Keywords |
| **内容缺口** | 手动 Google 对比 | Ahrefs Content Gap, SEMrush Keyword Gap | 输入你域 + 3 竞品域 |
| **技术 SEO** | Google Search Console (FREE, essential!), PageSpeed Insights, Rich Results Test | Screaming Frog, Deepcrawl | GSC 必须第一时间安装 |
| **排名追踪** | Google Search Console (average position) | Ahrefs Rank Tracker, SERPWatcher | 追踪目标关键词的排名变化 |
| **Schema 验证** | Google Rich Results Test (FREE) | Schema.org Validator (free) | 每个 Prompt/Article 发布前必测 |
| **外链分析** | Ahrefs Free Backlink Checker (limited) | Ahrefs Site Explorer | 监控竞品外链策略 |

### Google Search Console 配置（必须做）

```
1. 验证 harse.tv 域名所有权
2. 提交 Sitemap:
   - https://harse.tv/sitemap.xml (if exists)
   - 或 https://harse.tv/sitemap.xml.gz
3. 监控 Core Web Vitals:
   - LCP (Largest Contentful Paint) < 2.5s
   - FID (First Input Delay) < 100ms  
   - CLS (Cumulative Layout Shift) < 0.1
4. 设置国际 targeting:
   - Search Appearance → International Targeting
   - 确认 Country → Unlisted (global target)
   - Language: hreflang tags verified
5. 创建自定义仪表盘监控:
   - Top queries by impressions/CTR
   - Average position trend
   - Indexed pages count
   - Core Web Vitals status
```

---

## 六、版本记录

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| v1.0 | 2026-06-11 | Initial version (China market focus) | SEO Content Team |
| **v2.0** | **2026-06-11** | **Major overhaul: overseas/global market focus; real competitor intelligence (VideoPrompt.app/OpenPromptLib/Tona.AI/PromptBase/CinePrompt/RHTV/LibTV/HappyHorse); harse.tv capability-based clustering; model-keyword matrix method; multi-language i18n strategy; node canvas blue ocean differentiation; Google-first SEO approach; QC checklist enhanced for international SEO** | **SEO Content Team (revised based on live competitive analysis)** |

---

*This SOP is maintained by the SEO Content Team (Lead: 搜尔文)*  
*Before using this SOP, ensure authorization from the team lead*  
*Improvement suggestions: team-lead@seo-content-team*
