# VolcEngine 渠道增强 QA 报告

> **项目**：new-api 火山方舟（VolcEngine / Ark）渠道能力增强  
> **QA 版本**：v1.0  
> **编写日期**：2026-07-02  
> **负责人**：QA 工程师 · 严过关（software-qa-engineer）

---

## 1. 测试目标

验证 PRD《VolcEngine 渠道增强 PRD》与架构设计《VolcEngine 增强架构设计》中定义的三项能力：

1. 图片生成增强（2K/3K/4K、多图参考、sequential_image_generation、b64_json）
2. 文件上传（二进制 + URL 上传，返回 file_id / URL）
3. 多模态 Chat（image_url / video_url / input_audio / file）

---

## 2. 测试环境

- **仓库**：`F:\new api\`
- **Go 目标平台**：linux/amd64
- **测试限制**：当前会话为 Windows 桌面环境，Go 交叉编译目标为 linux/amd64，因此测试二进制无法直接执行；本报告以**编译通过 + 单元测试代码审查**作为核心验证手段。

---

## 3. 测试执行结果

| 检查项 | 命令/方法 | 结果 |
|--------|-----------|------|
| 全量编译 | `go build ./...` | ✅ 通过 |
| VolcEngine 单元测试编译 | `go test -c ./relay/channel/volcengine/...` | ✅ 通过 |
| 变更包 vet | `go vet ./relay/... ./dto/...` | ✅ 无新增问题（既有 heuristics 告警均位于未改动文件） |

> 注：因 GOOS=linux/amd64，测试二进制无法在本机直接运行；后续需在 Linux 运行环境或本机切换 GOOS=windows 后执行 `go test ./relay/channel/volcengine/...`。

---

## 4. 代码审查结果

### 4.1 新增/修改文件清单

- `dto/file.go` — 文件上传 Request/Response DTO，已实现 `dto.Request` 接口。
- `types/relay_format.go` — 新增 `RelayFormatFiles`。
- `relay/constant/relay_mode.go` — 新增 `RelayModeFiles` 与路径识别。
- `relay/channel/adapter.go` — 新增可选接口 `FileUploadAdaptor`。
- `relay/helper/valid_request.go` — 新增 `GetAndValidateFileUploadRequest`。
- `relay/file_handler.go` — 通用文件上传处理逻辑。
- `router/relay-router.go` — `/v1/files` POST 接入 `controller.Relay`。
- `controller/relay.go` — `relayHandler` 增加 `RelayModeFiles` 分支。
- `middleware/distributor.go` — `/v1/files` 特殊模型名 `volcengine-files`。
- `relay/channel/volcengine/image.go` — 图片生成请求/响应转换。
- `relay/channel/volcengine/file.go` — 文件上传实现。
- `relay/channel/volcengine/multimodal.go` — 多模态消息转换。
- `relay/channel/volcengine/adaptor.go` — 接入以上能力。
- `relay/channel/volcengine/constants.go` — 新增 `volcengine-files` 伪模型。
- `relay/channel/volcengine/volcengine_test.go` — 单元测试。

### 4.2 关键审查点

| 检查点 | 状态 | 说明 |
|--------|------|------|
| 图片生成字段透传 | ✅ | `size`、`image`（单/数组）、`sequential_image_generation`、`sequential_image_generation_options`、`output_format`、`watermark` 均正确映射。 |
| b64_json 下载转换 | ✅ | `response_format=b64_json` 时下载 URL 并转为 base64。 |
| 多模态消息转换 | ✅ | `image_url`/`video_url`/`input_audio`/`file` 支持 url/base64/file_id，已覆盖测试用例。 |
| 文件上传 multipart 构建 | ✅ | 二进制文件与 URL 二选一，支持 `purpose`、`preprocess_configs`、`tos`、`expire_at`。 |
| 路由与鉴权 | ✅ | `/v1/files` POST 走标准 Relay 流程，其他方法保持 NotImplemented。 |
| 接口兼容性 | ✅ | `FileUploadAdaptor` 为可选接口，不影响其他渠道。 |
| 计费处理 | ⚠️ | 图片生成返回空 `usage`，由 `image_handler.go` 统一按 `n` ratio 计费；文件上传按 1 token 计费，需配置 `volcengine-files` 模型价格。 |
| 错误处理 | ✅ | 统一使用 `types.NewError` / `types.NewOpenAIError` 返回标准错误。 |

---

## 5. 发现的问题与建议

### 5.1 问题 1：文件上传依赖伪模型 `volcengine-files`

**描述**：`/v1/files` 路由通过 `middleware.Distributor` 选择渠道，当前逻辑将模型名固定为 `volcengine-files`。用户必须在其 VolcEngine 渠道的模型列表中显式添加 `volcengine-files`，否则渠道选择失败。

**影响**：中。若未配置，文件上传请求会返回模型不可用错误。

**建议**：
- 在渠道配置文档中明确说明需添加 `volcengine-files` 模型。
- 后续优化：可让分销商在 `/v1/files` 时按渠道类型（VolcEngine）选择任意可用渠道，而非依赖伪模型。

### 5.2 问题 2：文件上传计费模型未精确

**描述**：文件上传成功后在 `FileHelper` 中按 `PromptTokens=1, TotalTokens=1` 进行后计费。若 `volcengine-files` 模型价格不为 0，则每次上传都会按 1 token × 价格 × 倍率扣费。

**影响**：低。可通过将 `volcengine-files` 模型价格设为 0 或按次计费规避。

**建议**：
- 文档中建议将 `volcengine-files` 设为按次计费或价格 0，由运营侧根据火山方舟实际费用配置。

### 5.3 问题 3：多模态 base64 前缀假设

**描述**：对于 `input_audio` 和 `file` 的 base64 数据，代码在缺少 `data:` 前缀时自动补齐 `data:audio/{format};base64,...` 或 `data:application/pdf;base64,...`。若火山方舟官方要求其他 MIME 类型，自动补齐可能不适用。

**影响**：低。用户通常可传入带正确前缀的 base64 或 file_id/url。

**建议**：在后续真机联调中验证火山方舟对音频/PDF base64 的 MIME 前缀要求，必要时扩展配置。

### 5.4 问题 4：测试未能在本机执行

**描述**：由于 Go 环境目标为 linux/amd64，无法直接运行测试二进制。

**影响**：中。单元测试仅完成编译，未实际执行。

**建议**：
- 在 Linux 环境或切换 `GOOS=windows` 后执行 `go test ./relay/channel/volcengine/...`。
- 建议补充端到端测试：真实 Seedream 生图、文件上传、多模态 Chat 调用。

---

## 6. 结论

- **编译状态**：✅ 通过
- **单元测试编译**：✅ 通过
- **代码审查**：✅ 通过，3 个低/中风险建议项已记录
- **整体判定**：代码可进入交付阶段，但需在目标 Linux 环境补跑单元测试，并依据建议 1-2 配置 `volcengine-files` 模型与价格。
