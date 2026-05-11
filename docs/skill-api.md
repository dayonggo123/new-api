# Skill API 接口文档

## 概述
画布 Skill 配置接口，用于前端画布节点动态渲染技能按钮。

---

## 1. 获取启用的 Skill 列表（公开）

**接口**：`GET /api/skills`

**鉴权**：`TryUserAuth`（未登录也可访问）

**说明**：返回所有 `status = 1` 的 Skill 配置。前端画布根据此列表渲染技能按钮。

**响应示例**：
```json
{
  "success": true,
  "data": [
    {
      "id": "prompt-translate",
      "name": "翻译提示词",
      "nameEn": "Translate Prompt",
      "icon": "languages",
      "cost": 0,
      "supportedNodeTypes": [
        "imageEditNode",
        "videoGenNode",
        "llmAgentNode",
        "textAnnotationNode"
      ],
      "description": "将提示词翻译成英文，提升 AI 生成效果",
      "execution": {
        "type": "llm",
        "systemPromptTemplate": "You are a professional translator. Translate the user's prompt into natural, fluent English optimized for AI generation. Output ONLY the translated text, no explanations.",
        "userPromptTemplate": "Translate the following into English:\n\n\"\"\"\n{{prompt}}\n\"\"\""
      },
      "overrideLocal": false
    }
  ]
}
```

---

## 2. 获取全部 Skill 列表（含禁用）

**接口**：`GET /api/skills/all`

**鉴权**：`AdminAuth`（仅管理员）

**说明**：返回所有 Skill，包括 `status = 2`（禁用）的项。用于后台管理。

**响应格式**：同上

---

## 3. 创建 Skill

**接口**：`POST /api/skills`

**鉴权**：`AdminAuth`

**请求体**：
```json
{
  "id": "prompt-translate",
  "name": "翻译提示词",
  "nameEn": "Translate Prompt",
  "icon": "languages",
  "cost": 0,
  "supportedNodeTypes": [
    "imageEditNode",
    "videoGenNode",
    "llmAgentNode",
    "textAnnotationNode"
  ],
  "description": "将提示词翻译成英文，提升 AI 生成效果",
  "execution": {
    "type": "llm",
    "systemPromptTemplate": "You are a professional translator...",
    "userPromptTemplate": "Translate:\n\"\"\"\n{{prompt}}\n\"\"\""
  },
  "overrideLocal": false,
  "status": 1
}
```

**字段说明**：

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `id` | ✅ | string | Skill 唯一标识，64 字符以内 |
| `name` | ✅ | string | 中文显示名 |
| `nameEn` | ❌ | string | 英文显示名 |
| `icon` | ✅ | string | Lucide 图标名，如 `languages`、`wand2`、`sparkles` |
| `cost` | ✅ | int | 积分消耗（0 = 免费） |
| `supportedNodeTypes` | ✅ | string[] | 支持的节点类型枚举 |
| `description` | ❌ | string | Skill 描述 |
| `execution.type` | ✅ | string | 固定为 `llm` |
| `execution.systemPromptTemplate` | ❌ | string | LLM system prompt 模板 |
| `execution.userPromptTemplate` | ❌ | string | LLM user prompt 模板 |
| `overrideLocal` | ❌ | bool | 是否覆盖前端同名内置 skill |
| `status` | ❌ | int | 1=启用（默认），2=禁用 |

**模板变量**（前端自动替换）：
- `{{prompt}}` — 用户节点上的提示词内容
- `{{language}}` — 当前界面语言（zh / en / ja 等）
- `{{nodeType}}` — 节点类型

**响应示例**：
```json
{
  "success": true,
  "data": {
    "id": "prompt-translate",
    "name": "翻译提示词",
    ...
  }
}
```

---

## 4. 更新 Skill

**接口**：`PUT /api/skills/:id`

**鉴权**：`AdminAuth`

**说明**：`id` 从 URL 路径获取，请求体格式同创建接口。注意：`id` 字段在请求体中会被忽略，以 URL 中的 `:id` 为准。

**请求示例**：
```bash
PUT /api/skills/prompt-translate
Content-Type: application/json

{
  "name": "翻译提示词（新版）",
  "cost": 10,
  "status": 1
}
```

---

## 5. 删除 Skill

**接口**：`DELETE /api/skills/:id`

**鉴权**：`AdminAuth`

**说明**：物理删除，不可恢复。

**请求示例**：
```bash
DELETE /api/skills/prompt-translate
```

**响应示例**：
```json
{
  "success": true,
  "data": null
}
```

---

## 节点类型枚举

```
uploadImageNode
imageEditNode
videoGenNode
exportImageNode
llmAgentNode
textAnnotationNode
storyboardGenNode
storyboardSplitNode
groupNode
productEditNode
```

---

## 前端调用链路

```
前端画布 → Tauri 命令 fetch_skills
  → 请求 GET /api/skills（带用户 Cookie/Token）
  → 返回 Skill 配置列表
  → 前端根据 supportedNodeTypes 过滤当前节点支持的 Skill
  → 动态渲染按钮
  → 点击后根据 execution 配置构造 LLM 请求
```

**降级处理**：接口返回 404 时前端静默失败，不影响现有功能。
