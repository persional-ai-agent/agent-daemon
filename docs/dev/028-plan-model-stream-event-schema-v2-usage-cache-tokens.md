# 028 计划：`model_stream_event` v2 用量缓存 token 字段补齐

## 目标

扩展 `usage` 统一字典，使缓存 token 可跨 provider 统一消费。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - `cache_creation_input_tokens -> prompt_cache_write_tokens`
   - `cache_read_input_tokens -> prompt_cache_read_tokens`
   - `prompt_tokens_details.cached_tokens -> prompt_cache_read_tokens`
   - `input_tokens_details.cached_tokens -> prompt_cache_read_tokens`
2. 增加标准化测试覆盖：
   - 直接字段映射
   - 嵌套字段映射
3. 回归测试并同步文档。

## 验证标准

- `usage` 事件可稳定输出 `prompt_cache_write_tokens/prompt_cache_read_tokens`
- 测试通过
