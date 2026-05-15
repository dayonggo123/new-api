# Prompt Collector - 提示词采集器

从 opennana.com 等第三方提示词平台一键采集提示词，自动提交到 new-api 提示词库。

## 功能

- 🔍 **智能提取**：在 opennana.com 列表页为每个提示词卡片注入"采集到库"按钮
- 📋 **自动抓取**：自动进入详情页提取 prompt 标题、正文、封面图、模型类型等信息
- ✏️ **弹窗编辑**：在插件弹窗中预览和编辑提取的内容，确认后一键入库
- 🔐 **安全认证**：Admin JWT Token 本地存储，仅用于调用 new-api 管理接口

## 安装步骤

### 1. 加载未打包的扩展

1. 打开 Chrome/Edge 浏览器，进入扩展管理页面：
   - Chrome: `chrome://extensions/`
   - Edge: `edge://extensions/`

2. 开启右上角「开发者模式」

3. 点击「加载已解压的扩展程序」，选择本文件夹 (`browser-extension/`)

4. 扩展图标会出现在浏览器工具栏中

### 2. 配置 API

1. 点击扩展图标打开弹窗
2. 展开「⚙️ API 配置」面板
3. 填写：
   - **API Base URL**: 你的 new-api 域名，如 `https://heharse.cloud`
   - **Admin JWT Token**: 从 new-api 管理后台获取的管理员 Token（格式：`Bearer eyJhbG...`）
   - **默认分类 ID**: 采集的提示词默认归入的分类（可选）
4. 点击「保存配置」

> 如何获取 Admin JWT Token？
> 1. 登录 new-api 管理后台
> 2. 打开浏览器开发者工具 (F12) → Application/应用 → Local Storage
> 3. 找到 `token` 字段的值，前面加上 `Bearer ` 即可

### 3. 开始采集

#### 方式一：列表页一键采集

1. 打开 `https://opennana.com/awesome-prompt-gallery`
2. 每个提示词卡片右上角会出现「采集到库」按钮
3. 点击按钮，插件会自动打开详情页并提取内容
4. 点击扩展图标，在弹窗中预览和编辑提取的内容
5. 点击「提交到提示词库」完成入库

#### 方式二：详情页直接采集

1. 在 opennana.com 点击任意提示词进入详情页
2. 直接点击扩展图标
3. 插件会自动提取当前详情页的 prompt 内容
4. 编辑后提交即可

## 提取字段映射

| 来源字段 | new-api 字段 | 说明 |
|---------|-------------|------|
| 页面标题 | `title` | 提示词标题 |
| 详情页 prompt 文本 | `content` | 提示词正文 |
| 页面描述/标题 | `description` | 简介 |
| OG 图片 / 页面图片 | `cover_image_url` | 封面图 |
| 检测到的模型名 | `model` | 如 GPT-4o、Nano Banana Pro |
| URL 参数/页面内容 | `media_type` | image 或 video |
| 页面标签 | `tags` | 标签列表 JSON |
| 固定值 | `status` | 1（启用） |
| 固定值 | `author` | "采集导入" |

## 注意事项

1. **Next.js SPA**: opennana.com 是单页应用，列表页滚动加载更多内容时，新卡片会自动注入采集按钮
2. **详情页适配**: 首次使用时建议检查提取的 prompt 内容是否准确，不同详情页布局可能影响提取效果
3. **CORS 限制**: 插件通过 background service worker 代理跨域请求到 new-api，无需服务端改动
4. **Token 安全**: Admin JWT 仅存储在浏览器本地，不会上传到任何第三方

## 文件结构

```
browser-extension/
├── manifest.json          # Chrome Extension V3 配置
├── background.js          # Service Worker（跨域请求代理）
├── content-list.js        # 列表页脚本（注入采集按钮）
├── content-detail.js      # 详情页脚本（提取 prompt 内容）
├── popup.html             # 弹窗 UI
├── popup.js               # 弹窗逻辑（编辑 + API 提交）
├── popup.css              # 弹窗样式
├── icons/                 # 扩展图标
│   ├── icon16.png
│   ├── icon32.png
│   ├── icon48.png
│   └── icon128.png
└── README.md              # 本文件
```

## 后续扩展

如需适配其他提示词平台，修改以下文件：

- `manifest.json` → `content_scripts.matches` 添加新域名
- `content-list.js` → 添加新平台的卡片选择器
- `content-detail.js` → 添加新平台的 prompt 提取策略

## 技术支持

如有问题，请检查：
1. API Base URL 是否正确（不要带末尾斜杠 `/`）
2. Admin JWT Token 是否有效且未过期
3. new-api 的 `/api/prompt/` 和 `/api/prompt-category/all` 接口是否可访问
