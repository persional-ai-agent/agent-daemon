# 040 总结：Provider 完整事件字典覆盖

## 变更摘要

补齐各 provider 流式事件中可主动提供的关键字段，使下游消费者可依赖统一字段。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/model/codex.go` | 从 `response.created` 事件提取 `responseID`，在 `message_done` 中添加 `message_id` |
| `internal/model/anthropic.go` | 当 `stop_reason=max_tokens` 时，在 `message_done` 中添加 `incomplete_reason=length` |
| `internal/model/openai.go` | 从流式 chunk 提取 `finish_reason`，当 `finish_reason=length` 时添加 `incomplete_reason=length` |
| `internal/model/anthropic_stream_test.go` | 新增 `TestAnthropicStreamingMaxTokensIncompleteReason` |
| `internal/model/codex_stream_test.go` | 新增 `TestCodexStreamingMessageDoneWithResponseID` |
| `internal/model/openai_stream_test.go` | 新增 `TestOpenAIStreamingLengthIncompleteReason` |

## 补齐后覆盖矩阵

### `message_done` 字段覆盖

| 字段 | OpenAI | Anthropic | Codex |
|------|--------|-----------|-------|
| `message_id` | ❌ 上游限制 | ✅ | ✅ 新增 |
| `finish_reason` | ✅ (含流式提取) | ✅ | ✅ |
| `stop_sequence` | ❌ 上游限制 | ✅ | ❌ 上游限制 |
| `incomplete_reason` | ✅ 新增 | ✅ 新增 | ✅ (已有) |

## 不可补齐的缺口

- OpenAI `message_id`：`chat/completions` 流式响应不提供消息 ID
- OpenAI `stop_sequence`：OpenAI 不支持自定义 stop sequence 的流式返回
- Codex `stop_sequence`：Codex API 不提供 stop sequence

## 测试结果

`go test ./...` 全部通过，新增 3 个测试用例覆盖本次变更。
