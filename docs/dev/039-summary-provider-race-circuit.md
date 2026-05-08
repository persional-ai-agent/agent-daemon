# 039 总结：Provider 并行竞速与熔断补齐结果

## 已完成

- 新增 `internal/model/circuit.go`：
  - `ProviderCircuit` 熔断器状态机（Closed / Open / Half-Open）
  - `AllowRequest()` 自动处理 Open → Half-Open 转换
  - `RecordSuccess()` / `RecordFailure()` 状态转换与计数
  - `State()` 返回当前状态（含超时推断 Half-Open）
  - `IncrementHalfOpenRequests()` 半开试探计数

- 改造 `internal/model/fallback.go`：
  - `FallbackClient` 增加 `PrimaryCircuit` / `FallbackCircuit` 字段
  - 新增 `NewFallbackClientWithCircuit()` 构造函数
  - `ChatCompletionWithEvents` 调用前检查熔断器状态
  - 熔断器 Open 时自动跳过故障 provider
  - 成功/失败后自动更新熔断器状态

- 新增 `internal/model/race.go`：
  - `RaceClient` 并行竞速客户端
  - 同时向未熔断的 provider 发请求
  - `select` 取最快成功响应
  - 自动取消慢请求
  - 成功者重置状态，失败者记录失败

- 扩展 `internal/config/config.go`：
  - `AGENT_MODEL_RACE_ENABLED`：是否开启并行竞速（默认 `false`）
  - `AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD`：连续失败阈值（默认 `3`）
  - `AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS`：熔断恢复超时（默认 `60`）
  - `AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS`：半开最大试探请求数（默认 `1`）

- 更新 `cmd/agentd/main.go`：
  - 按 `AGENT_MODEL_RACE_ENABLED` 选择 `RaceClient` 或 `FallbackClientWithCircuit`
  - 启动日志显示当前模式

- 新增测试：
  - `circuit_test.go`：熔断器状态机全覆盖（7 个用例）
  - `race_test.go`：竞速模式 + 熔断器联动（4 个用例）

## 新增配置

```bash
export AGENT_MODEL_RACE_ENABLED=true
export AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD=3
export AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS=60
export AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS=1
```

## 验证

- `go test ./...` 通过
- 熔断器状态转换测试通过
- 竞速模式取最快响应测试通过
- 熔断器联动 fallback 测试通过

## 当前边界

- 熔断器为进程内状态，不做跨进程持久化
- 并行竞速默认关闭，避免成本激增
- 仅支持双 provider 竞速，不扩展到多级级联
- 未实现 provider 健康探针（依赖请求失败触发）
