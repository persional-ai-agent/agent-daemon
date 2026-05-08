# 027 计划：`model_stream_event` v2 终止原因一致性补齐

## 目标

增强 `message_done.finish_reason` 与 `message_done.incomplete_reason` 的一致性，减少客户端推断逻辑。

## 实施步骤

1. 扩展 `normalizeStreamEvent(message_done)`：
   - `incomplete_reason` 归一（`max_tokens/max_output_tokens -> length`）
   - 当 `finish_reason=length` 且 `incomplete_reason` 缺失时，自动补 `incomplete_reason=length`
2. 更新标准化测试：
   - 覆盖自动补全场景
   - 覆盖别名归一场景
3. 回归测试并同步文档。

## 验证标准

- `finish_reason=length` 时，`incomplete_reason` 必定存在且为 `length`
- 历史别名可统一收敛
- 测试通过
