# 006 总结：Codex Responses 模式补齐结果

## 已完成

- 新增 `internal/model/codex.go`，实现 Responses API client
- 新增 `provider=codex` 运行时切换
- 新增配置项：
  - `CODEX_BASE_URL`
  - `CODEX_API_KEY`
  - `CODEX_MODEL`
- 实现核心映射：
  - `function_call` -> `core.ToolCall`
  - `tool` 消息 -> `function_call_output` 输入项
- 新增 `internal/model/codex_test.go` 测试覆盖请求映射与响应解析

## 验证

- `go test ./...` 通过

## 当前边界

本次补齐了 Codex 基础模式，但仍未覆盖：

- Responses API 的高级项（reasoning 项、延续对话 item_id 管理等）
- provider 级流式统一抽象
