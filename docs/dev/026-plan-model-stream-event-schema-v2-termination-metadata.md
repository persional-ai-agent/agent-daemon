# 026 计划：`model_stream_event` v2 终止元数据补齐

## 目标

统一 `message_done` 的终止元数据表达，降低前端/SDK 对 provider 差异的适配成本。

## 实施步骤

1. 扩展 `normalizeStreamEvent(message_done)`：
   - `stop -> stop_sequence`
   - `incomplete_details.reason` / `reason_detail -> incomplete_reason`
2. Anthropic 流式路径：
   - 透传 `message_delta.stop_sequence`
3. Codex completed envelope：
   - 透传 `response.incomplete_details.reason`
4. 更新模型层测试：
   - 标准化测试覆盖 `stop_sequence` 与 `incomplete_reason`
   - provider 流式测试覆盖透传字段
5. 回归 `go test ./...` 并同步文档。

## 验证标准

- `message_done` 可稳定输出 `stop_sequence`（若上游存在）
- `message_done` 可稳定输出 `incomplete_reason`（若上游存在）
- 回归测试通过
