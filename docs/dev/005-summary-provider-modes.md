# 005 总结：Provider 多模式补齐结果

## 已完成

- 新增 `internal/model/anthropic.go`，实现 Anthropic Messages client
- 保留统一 `model.Client` 接口，`Engine` 无需改动
- 新增 provider 切换：
  - `AGENT_MODEL_PROVIDER=openai|anthropic`
- 新增 Anthropic 配置：
  - `ANTHROPIC_BASE_URL`
  - `ANTHROPIC_API_KEY`
  - `ANTHROPIC_MODEL`
- 新增 `internal/model/anthropic_test.go` 覆盖协议映射与解析

## 实现要点

- `system` 消息提取为 Anthropic `system` 字段
- `assistant.tool_calls` 映射为 `tool_use` block
- `tool` 消息映射为 `tool_result` block
- 响应中的 `tool_use` 反解为 `core.ToolCall`

## 验证

- `go test ./...` 通过

## 当前边界

已完成 OpenAI + Anthropic 双模式，但仍未覆盖：

- Codex Responses 模式
- provider 级高级重试/故障切换
- 各 provider 的流式差异抽象
