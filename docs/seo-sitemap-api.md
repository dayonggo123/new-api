# SEO Sitemap API 接口文档

> 专为 SEO 站点地图（sitemap.xml）动态生成而设计的轻量公开接口。
> 只返回生成 sitemap 所需的核心字段，数据量极小，支持大页码分页。

---

## 一、文章站点地图接口

### `GET /api/public/articles/sitemap`

**认证：** 无需认证

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 500，最大 5000 |

**响应示例：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "items": [
      {
        "id": 1,
        "slug": "ru-he-shi-yong-chatgpt",
        "updated_time": 1759300000,
        "created_time": 1759200000
      },
      {
        "id": 2,
        "slug": "ai-xie-zuo-ji-qiao",
        "updated_time": 1759100000,
        "created_time": 1759000000
      }
    ],
    "total": 156,
    "page": 1,
    "page_size": 500,
    "total_page": 1
  }
}
```

**字段说明：**

| 字段 | 说明 |
|------|------|
| id | 文章 ID |
| slug | 文章 URL 路径（如 `/articles/ru-he-shi-yong-chatgpt`） |
| updated_time | 最后更新时间（秒级时间戳），用于 sitemap `<lastmod>` |
| created_time | 创建时间（秒级时间戳） |

**文章 URL 构造：**

```
https://heharse.cloud/articles/{slug}
```

---

## 二、提示词站点地图接口

### `GET /api/public/prompts/sitemap`

**认证：** 无需认证

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 500，最大 5000 |

**响应示例：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "items": [
      {
        "id": 101,
        "title": "小红书文案生成器",
        "updated_time": 1759300000,
        "created_time": 1759200000
      },
      {
        "id": 102,
        "title": "AI 绘图提示词优化",
        "updated_time": 1759100000,
        "created_time": 1759000000
      }
    ],
    "total": 823,
    "page": 1,
    "page_size": 500,
    "total_page": 2
  }
}
```

**字段说明：**

| 字段 | 说明 |
|------|------|
| id | 提示词 ID |
| title | 提示词标题（可用于生成 URL slug） |
| updated_time | 最后更新时间（秒级时间戳），用于 sitemap `<lastmod>` |
| created_time | 创建时间（秒级时间戳） |

**提示词 URL 构造：**

```
https://heharse.cloud/prompts/{id}
```

> 提示词详情页目前使用 `id` 作为路径参数（如 `/prompts/101`），没有单独的 slug 字段。
> 如需 slug 形式的 URL（如 `/prompts/xiao-hong-shu-wen-an`），后续可给 Prompt 模型增加 slug 字段。

---

## 三、分页与性能

| 项目 | 说明 |
|------|------|
| 默认 page_size | 500 条 |
| 最大 page_size | 5000 条 |
| 排序 | 按 `updated_time desc` 倒序 |
| 查询字段 | 只 select SEO 需要的 4 个字段，不加载 content 等大字段 |
| 事务 | 不使用事务，纯 SELECT 查询 |

**建议的分页策略：**

- 文章：如果总量 < 5000，建议 `page_size=5000` 一次拉完
- 提示词：如果总量 800+，建议 `page_size=5000` 一次拉完；未来扩展到数万条时，分多次请求

**示例：拉取全部提示词（假设 800+ 条）：**

```bash
curl "https://heharse.cloud/api/public/prompts/sitemap?page=1&page_size=5000"
```

**示例：分页拉取（适合数万条场景）：**

```bash
# 第 1 页
curl "https://heharse.cloud/api/public/prompts/sitemap?page=1&page_size=5000"

# 第 2 页
curl "https://heharse.cloud/api/public/prompts/sitemap?page=2&page_size=5000"
```

---

## 四、生成 sitemap.xml 的示例代码

**Python 示例：**

```python
import requests
import xml.etree.ElementTree as ET
from datetime import datetime, timezone

BASE_URL = "https://heharse.cloud"
API_BASE = f"{BASE_URL}/api"

def fetch_all(endpoint):
    """分页拉取全部数据"""
    items = []
    page = 1
    while True:
        res = requests.get(f"{API_BASE}{endpoint}", params={"page": page, "page_size": 5000})
        data = res.json()["data"]
        items.extend(data["items"])
        if page >= data["total_page"]:
            break
        page += 1
    return items

def build_sitemap():
    urlset = ET.Element("urlset", xmlns="http://www.sitemaps.org/schemas/sitemap/0.9")

    # 文章
    articles = fetch_all("/public/articles/sitemap")
    for a in articles:
        url = ET.SubElement(urlset, "url")
        ET.SubElement(url, "loc").text = f"{BASE_URL}/articles/{a['slug']}"
        lastmod = datetime.fromtimestamp(a["updated_time"], tz=timezone.utc).strftime("%Y-%m-%d")
        ET.SubElement(url, "lastmod").text = lastmod
        ET.SubElement(url, "changefreq").text = "weekly"
        ET.SubElement(url, "priority").text = "0.8"

    # 提示词
    prompts = fetch_all("/public/prompts/sitemap")
    for p in prompts:
        url = ET.SubElement(urlset, "url")
        ET.SubElement(url, "loc").text = f"{BASE_URL}/prompts/{p['id']}"
        lastmod = datetime.fromtimestamp(p["updated_time"], tz=timezone.utc).strftime("%Y-%m-%d")
        ET.SubElement(url, "lastmod").text = lastmod
        ET.SubElement(url, "changefreq").text = "weekly"
        ET.SubElement(url, "priority").text = "0.6"

    tree = ET.ElementTree(urlset)
    tree.write("sitemap.xml", encoding="utf-8", xml_declaration=True)
    print(f"Generated sitemap with {len(articles)} articles and {len(prompts)} prompts")

build_sitemap()
```

---

## 五、与现有接口对比

| 特性 | 现有列表接口 | Sitemap 专用接口 |
|------|-------------|-----------------|
| 路径 | `/api/public/articles` | `/api/public/articles/sitemap` |
| 返回字段 | 完整 Article（含 content、summary 等） | 只返回 id、slug、updated_time、created_time |
| 默认 page_size | 10 | 500 |
| 最大 page_size | 100 | 5000 |
| 使用事务 | 是（Begin/Commit） | 否 |
| 适用场景 | 前端列表展示 | SEO sitemap 生成 |

---

## 六、后续扩展建议

1. **给提示词增加 slug 字段**
   - 在 `Prompt` 模型中增加 `slug string gorm:"uniqueIndex;size:255"`
   - 创建/更新时自动从 title 生成 slug
   - 详情页路由改为 `/prompts/{slug}`，SEO 更友好

2. **sitemap 索引文件**
   - 如果总量超过 50,000 条，需要拆分为多个 sitemap 文件，并用 `sitemapindex.xml` 索引

3. **定时生成**
   - 建议设置定时任务（如每天凌晨）自动拉取数据生成 sitemap.xml，并推送到 CDN
