# 032 计划：`model_stream_event` v2 用量异常状态补齐

## 目标

为 `usage_consistency_status` 增加 `invalid`，使客户端可直接识别脏数据场景。

## 实施步骤

1. 扩展 `normalizeStreamEvent(usage)`：
   - 识别 token 信号字段是否存在
   - 在未命中 `ok/derived/adjusted/source_only` 时，输出 `invalid`
2. 增加标准化测试：
   - 字符串数值可解析场景（应正常归一）
   - 非法字符串场景（应标记 `invalid`）
3. 回归 `go test ./...` 并同步文档。

## 验证标准

- 异常 token 输入时，`usage_consistency_status=invalid`
- 正常字符串数值输入仍可按既有规则归一
- 测试通过
