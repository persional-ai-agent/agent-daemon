# 015 总结：Provider 故障切换最小补齐结果

## 已完成

- 新增 `model.FallbackClient`，支持主备 provider 调用
- 回退触发条件：
  - 错误码：`408/429/500/502/503/504`
  - `timeout`、`context deadline exceeded`、连接拒绝/重置
- 新增配置项：`AGENT_MODEL_FALLBACK_PROVIDER`
- 启动装配支持自动包裹 fallback 客户端
- 新增测试：
  - `internal/model/fallback_test.go`
  - `cmd/agentd/main_test.go`

## 验证

- `go test ./cmd/agentd ./internal/model ./...` 通过

## 当前边界

- 未覆盖并行竞速与熔断策略
- 未做多级 fallback 链路
