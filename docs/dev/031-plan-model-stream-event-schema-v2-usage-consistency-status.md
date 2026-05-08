# 031 计划：`model_stream_event` v2 用量一致性状态字段补齐

## 目标

为 `usage` 增加统一一致性状态，降低客户端解析复杂度。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - 新增 `usage_consistency_status`
   - 状态规则：
     - `derived`
     - `adjusted`
     - `ok`
     - `source_only`
2. 更新标准化测试：
   - 补齐 `derived/adjusted/ok` 场景断言
3. 回归测试并同步文档。

## 验证标准

- `usage` 事件可稳定输出 `usage_consistency_status`
- 状态与 `total_tokens` 处理路径一致
- 测试通过
