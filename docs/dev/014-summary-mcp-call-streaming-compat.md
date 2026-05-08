# 014 总结：MCP `/call` 流式响应兼容结果

## 已完成

- MCP `/call` 增加 `text/event-stream` 分支处理
- 新增最小 SSE 解析器，支持：
  - `data:` 多行拼接
  - `[DONE]` 终止标记
  - `result` / `structuredContent` 结果提取
  - `error` 事件错误返回
- 保持原 JSON 响应处理逻辑不变
- 新增 SSE `/call` 测试覆盖

## 验证

- `go test ./internal/tools -run MCP -v` 通过
- `go test ./...` 通过

## 当前边界

- 未将流式中间事件透传到 Agent 事件总线
- 仍以“聚合后一次返回”方式对接现有工具调用接口
