# 030 计划：`model_stream_event` v2 用量总量一致性补齐

## 目标

增强 `usage.total_tokens` 的可用性与一致性，减少客户端重复兜底逻辑。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - 当 `total_tokens` 缺失且 `prompt_tokens/completion_tokens` 可用时自动补齐
   - 当 `total_tokens < prompt_tokens + completion_tokens` 时自动校正
   - 输出 `total_tokens_adjusted=true` 标记校正行为
2. 增加标准化测试：
   - 缺失总量自动补齐
   - 总量偏小自动校正
3. 回归 `go test ./...` 并同步文档。

## 验证标准

- `usage.total_tokens` 在主路径可稳定使用
- 校正场景可通过 `total_tokens_adjusted` 识别
- 测试通过
