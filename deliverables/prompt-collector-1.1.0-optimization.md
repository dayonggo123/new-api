# Prompt Collector 1.1.0 优化报告

## 修改文件清单

| 文件 | 修改内容 |
|------|----------|
| `manifest.json` | 无实质性改动（URL 匹配改由脚本自身处理） |
| `background.js` | +`fetchWithTimeout` 超时封装、消息参数校验、错误分类 |
| `content-list.js` | +URL 安全检查、+300ms 防抖 MutationObserver、+精确容器监听、+错误日志 |
| `content-detail.js` | +URL 安全检查、+`appendBatchData` 批量通道、+自动打开侧边栏 |
| `sidepanel.js` | +`loadCategories()` 错误日志 |

## 优化明细

### 🛡️ 稳定性（3 项）

1. **content_scripts 互斥** — 两个脚本通过 `window.location.pathname` 自检，
   避免在对方页面上同时运行导致重复操作或 DOM 冲突。

2. **API 请求超时** — `background.js` 新增 `fetchWithTimeout(url, opts, 15s)`，
   使用 `AbortController` 实现。网络故障或 API 不可达时不再永久挂起。

3. **错误分类与日志** — 区分 AbortError（请求超时）与其他运行时错误，
   所有空 catch 块补全了有意义的日志输出（`console.warn` / `console.debug`）。

### ⚡ 性能（1 项）

4. **MutationObserver 防抖** — 300ms 合并窗口，避免 SPA 滚动加载时
   短时间内密集触发 `scanCards()`。同时优先定位卡片容器缩小观察范围。

### 🔄 数据流（1 项）

5. **详情页采集统一走批量通道** — `content-detail.js` 从只写 `EXTRACTED_KEY`
   改为同时写 `BATCH_KEY`，采集的数据在 sidepanel 中可见，不再只局限于 popup。

---

## 验证方式

1. 在 Chrome 扩展管理页刷新 Prompt Collector
2. 打开 opennana.com 列表页，确认采集按钮正常出现
3. 打开 opennana.com 详情页，确认没有采集按钮干扰
4. 侧边栏自动弹出，采集的数据出现在列表中
5. 配置错误的 API URL 时，15s 后显示超时提示而非一直转圈
