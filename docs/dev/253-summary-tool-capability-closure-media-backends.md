# 工具能力收口（第 4 项）：media/best-effort 实后端完善

本轮针对“工具能力级差距”做了完整收口，重点完善 `vision_analyze / image_generate / text_to_speech` 的实后端兼容与回退链路。

## 主要改动

- `internal/tools/media_tools.go`
  - `vision_analyze`：
    - OpenAI 响应结构异常时不再直接失败，改为回退到元数据模式，保证工具可用性。
  - `image_generate`：
    - OpenAI 图片接口在非 `b64_json` 返回时，新增 URL 下载兜底（`tryWriteImageFromURL`）。
    - 仍保留本地 placeholder 回退，形成“实后端优先 + 可降级”链路。
  - `text_to_speech`：
    - 新增参数级覆盖：`model`、`voice`（优先于环境变量）。
    - 继续保留 OpenAI 失败后的 WAV 占位回退。
- `internal/tools/builtin.go`
  - `text_to_speech` schema 补齐 `model` 与 `voice` 字段。

## 测试

- 新增 `internal/tools/media_tools_test.go`
  - `tryWriteImageFromURL` 下载写盘测试
  - `text_to_speech` schema 字段覆盖测试（`model/voice`）

## 验证

- `go test ./internal/tools ./internal/api ./cmd/agentd`
- `make contract-check`
- `go test ./...`
