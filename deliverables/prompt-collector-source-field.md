# 新增"来源平台"字段（Source）

## 修改明细

### 后端 (Go)

| 文件 | 变更 |
|------|------|
| `model/prompt.go` | +`Source string json:"source"` 字段 |
| `model/prompt.go` | `Update()` 的 `.Select()` 加入 `"source"` |
| `controller/prompt.go` | `UpdatePrompt()` 加入 `cleanPrompt.Source = prompt.Source` |

### 浏览器扩展

| 文件 | 变更 |
|------|------|
| `content-list.js` | +`extractSource(hostname)` 函数，data 中加入 `source` |
| `content-list.js` | 自动提交 payload 加入 `source` |
| `content-detail.js` | +`extractSource(hostname)` 函数，data 中加入 `source` |
| `popup.html` | 表单新增"来源平台"输入框 |
| `popup.js` | 渲染/提交时包含 `source` 字段 |
| `sidepanel.html` | 编辑面板新增"来源平台"输入框 |
| `sidepanel.js` | 提交/编辑时包含 `source` 字段 |

## 数据流

```
URL 域名 → extractSource() → data.source → storage + API payload → Prompt.Source
```

例：`https://opennana.com/awesome-prompt-gallery/xxx`
→ `extractSource("opennana.com")` → `"opennana"` → 入库时写入 `source` 列

## 验证

1. 扩展：采集一个提示词，确认弹窗/侧边栏中"来源平台"字段已被自动填充
2. 后端：查看数据库 `prompts` 表，`source` 列应有对应值
3. 手动编辑来源平台，再次提交应更新到数据库
