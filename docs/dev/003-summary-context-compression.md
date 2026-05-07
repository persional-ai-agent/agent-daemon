# 003 总结：Context Compression 补齐结果

## 已完成

- 新增 `internal/agent/compressor.go`，实现确定性上下文压缩器
- 在 `Engine.Run()` 每轮模型调用前接入预算检查与压缩
- 新增 `context_compacted` 结构化事件，输出压缩前后体积和裁剪量
- 新增配置项：
  - `AGENT_MAX_CONTEXT_CHARS`
  - `AGENT_COMPRESSION_TAIL_MESSAGES`
- 新增测试：
  - `compressor_test.go`
  - `loop_test.go` 压缩触发与事件测试

## 实现要点

- 预算估算：基于消息文本与 tool call 字段做字符级估算
- 保护策略：保留首条 system message 与最近 N 条消息
- 压缩策略：将中段历史转为一条 `assistant` 摘要消息，附带固定前缀
- 安全性：避免 tail 以孤立 `tool` 消息开头，减少上下文结构断裂风险

## 验证

- `go test ./...` 通过

## 当前边界

已具备最小可用上下文压缩能力，但仍未实现 Hermes 的高级能力：

- 辅助模型摘要
- 迭代摘要融合
- 多模态精细 token 预算
