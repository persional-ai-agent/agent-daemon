# 029 计划：`model_stream_event` v2 用量推理 token 字段补齐

## 目标

扩展 `usage` 字典，统一推理 token 的读取口径。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - `completion_tokens_details.reasoning_tokens -> reasoning_tokens`
   - `output_tokens_details.reasoning_tokens -> reasoning_tokens`
   - `reasoning_tokens_count -> reasoning_tokens`
2. 增加标准化测试覆盖：
   - OpenAI 风格嵌套字段
   - Codex 风格嵌套字段
3. 执行回归并同步文档。

## 验证标准

- `usage` 事件可稳定输出 `reasoning_tokens`
- 测试通过
