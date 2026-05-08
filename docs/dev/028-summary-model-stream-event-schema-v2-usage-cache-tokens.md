# 028 总结：`model_stream_event` v2 用量缓存 token 字段补齐

## 已完成

- `usage` 标准化新增缓存 token 字段：
  - `prompt_cache_write_tokens`
  - `prompt_cache_read_tokens`
- 兼容映射已补齐：
  - `cache_creation_input_tokens -> prompt_cache_write_tokens`
  - `cache_read_input_tokens -> prompt_cache_read_tokens`
  - `prompt_tokens_details.cached_tokens -> prompt_cache_read_tokens`
  - `input_tokens_details.cached_tokens -> prompt_cache_read_tokens`
- 新增标准化测试覆盖直接字段与嵌套字段。

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅补最小缓存 token 字段，未扩展 provider 全量计费维度
