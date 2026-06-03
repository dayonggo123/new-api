# OpenNana 提示词采集器使用说明

## 功能概述

自动采集 https://opennana.com/awesome-prompt-gallery 的全部提示词内容，支持：

- **无限滚动自动加载** — 自动滚动页面直到加载完全部 ~9227 条提示词
- **详情页深度采集** — 逐个访问详情页，提取完整提示词原文、标签、模型类型、生成参数等
- **断点续传** — 中途停止可从中断处继续，不会重复采集
- **增量更新** — 已采集的条目自动跳过
- **双格式输出** — JSON（完整结构） + CSV（表格形式）

---

## 采集字段

每条提示词包含以下字段：

| 字段 | 说明 |
|------|------|
| `id` | 提示词唯一标识 |
| `title` | 标题（中文描述） |
| `prompt_text` | 提示词原文（完整内容） |
| `prompt_text_en` | 英文提示词原文（如果是英文的） |
| `model` | AI 模型（Nano Banana Pro / ChatGPT / Grok / Seedance 2.0 等） |
| `media_type` | 媒体类型（image / video） |
| `tags` | 标签列表（如电影感、超写实、人像等） |
| `thumbnail_url` | 缩略图 URL |
| `image_urls` | 页面中的所有图片/视频 URL 列表 |
| `generation_params` | 生成参数（分辨率、比例等） |
| `author` | 作者信息 |
| `created_at` | 创建时间 |
| `source_url` | 原始页面链接 |
| `crawled_at` | 采集时间 |

---

## 安装依赖

```bash
# 1. 安装 Playwright Python 包
pip install playwright

# 2. 安装 Chromium 浏览器（只需执行一次）
playwright install chromium
```

---

## 使用方法

### 基础用法 — 采集全部提示词

```bash
python crawler_opennana.py
```

### 只采集图片类提示词

```bash
python crawler_opennana.py --media image
```

### 只采集视频类提示词

```bash
python crawler_opennana.py --media video
```

### 只采集前 100 条（测试用）

```bash
python crawler_opennana.py --max 100
```

### 显示浏览器窗口（调试用）

```bash
python crawler_opennana.py --visible
```

### 组合使用

```bash
# 只采图片、前 500 条、显示窗口看效果
python crawler_opennana.py --media image --max 500 --visible
```

---

## 输出文件

所有文件保存在 `./opennana_data/` 目录：

```
opennana_data/
├── prompts_list.json      # 列表页原始数据
├── prompts_detail.json    # 详情页完整数据（主文件）
├── prompts.csv            # CSV 格式（方便导入 Excel/数据库）
├── progress.json          # 进度记录（断点续信用）
└── crawler.log            # 运行日志
```

---

## 断点续传机制

采集过程中会自动保存进度到 `progress.json`：

- **正常中断**（Ctrl+C / 关闭窗口）：已采数据不会丢失，下次运行自动跳过已完成条目
- **异常中断**（网络错误 / 页面崩溃）：失败条目记录到 `failed_ids`，成功条目记录到 `completed_ids`
- **想重新采集某条**：手动编辑 `progress.json`，从 `completed_ids` 中移除对应 ID

---

## 同步到你的提示词库

采集完成后，你可以：

### 方式1：直接读取 JSON

```python
import json

with open("opennana_data/prompts_detail.json", "r", encoding="utf-8") as f:
    prompts = json.load(f)

for p in prompts:
    print(p["title"], p["prompt_text"], p["tags"])
    # 这里写入你的数据库...
```

### 方式2：导入数据库

CSV 文件可直接导入 MySQL / PostgreSQL：

```sql
-- MySQL 示例
LOAD DATA INFILE 'prompts.csv'
INTO TABLE your_prompt_table
CHARACTER SET utf8mb4
FIELDS TERMINATED BY ','
ENCLOSED BY '"'
LINES TERMINATED BY '\n'
IGNORE 1 ROWS;
```

### 方式3：调用你的 API

写一个适配脚本，读取 JSON 后调用你的提示词库 API：

```python
import json, requests

with open("opennana_data/prompts_detail.json") as f:
    prompts = json.load(f)

for p in prompts:
    requests.post("https://your-api.com/prompts", json={
        "title": p["title"],
        "content": p["prompt_text"],
        "tags": p.get("tags", []),
        "model": p.get("model", ""),
        "source": p["source_url"],
    })
```

---

## 注意事项

1. **采集速度**：9227 条全部采完预计需要 **2~4 小时**（取决于网络），建议先 `--max 100` 测试
2. **反爬策略**：脚本已设置合理的请求间隔（默认 0.5s），如遇到限制可适当调大 `DETAIL_DELAY`
3. **页面结构变更**：如果网站改版导致选择器失效，需要更新脚本中的 CSS 选择器
4. **数据版权**：采集的数据仅供个人学习使用，请遵守原站使用条款

---

## 常见问题

**Q: 运行报错 `playwright._impl._errors.Error: BrowserType.launch`?**
> A: 先执行 `playwright install chromium` 安装浏览器

**Q: 采集到一半断了，怎么继续？**
> A: 直接重新运行脚本即可，会自动从断点继续

**Q: 如何只更新新增的内容？**
> A: 脚本会自动跳过 `progress.json` 中记录的 `completed_ids`，新增内容会自动采集

**Q: 采集结果里有些字段是空的？**
> A: 详情页的字段依赖页面实际渲染的内容，有些页面可能缺少某些信息，属于正常情况
