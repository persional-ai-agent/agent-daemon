# 032 总结：`model_stream_event` v2 用量异常状态补齐

## 已完成

- `usage_consistency_status` 新增状态：
  - `invalid`
- 标准化层新增 token 信号识别：
  - 在存在 token 字段但无法形成有效总量一致性路径时，标记为 `invalid`
- 新增测试覆盖：
  - 字符串数值输入可正常归一
  - 非法字符串输入标记 `invalid`

## 验证

- `go test ./internal/model` 通过
- `go test ./...` 通过

## 当前边界

- 仅提供单一异常状态，未细分非法来源类型
