# 039 计划：Provider 并行竞速与熔断

## 目标

在现有 `FallbackClient` 基础上增加熔断器与可选并行竞速能力，提升 Provider 层的故障隔离与延迟优化。

## 实施步骤

### 1. 实现熔断器核心状态机

新增 `internal/model/circuit.go`：

- `CircuitState` 枚举（Closed / Open / Half-Open）
- `ProviderCircuit` 结构体（失败计数、状态转换、时间窗口）
- `AllowRequest()` 判断是否允许发请求
- `RecordSuccess()` / `RecordFailure()` 更新状态
- `State()` 返回当前状态

验证：单元测试覆盖状态转换逻辑（Closed→Open→Half-Open→Closed/Open）

### 2. 改造 FallbackClient 集成熔断器

修改 `internal/model/fallback.go`：

- `FallbackClient` 增加 `PrimaryCircuit` / `FallbackCircuit` 字段
- `ChatCompletionWithEvents` 调用前检查熔断器状态
- 成功/失败后调用 `RecordSuccess()` / `RecordFailure()`
- 保持向后兼容：无熔断器配置时行为不变

验证：测试熔断器触发时跳过故障 provider

### 3. 实现并行竞速客户端

新增 `internal/model/race.go`：

- `RaceClient` 结构体（主备 provider + 熔断器 + 竞速开关）
- `ChatCompletionWithEvents` 实现并行发请求 + `select` 取最快
- 仅对未熔断的 provider 发请求
- 成功者重置状态，失败者记录失败
- 使用 `context.WithCancel` 取消慢请求

验证：测试竞速模式取最快响应、取消慢请求

### 4. 扩展配置项

修改 `internal/config/config.go`：

- `AGENT_MODEL_RACE_ENABLED`（默认 `false`）
- `AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD`（默认 `3`）
- `AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS`（默认 `60`）
- `AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS`（默认 `1`）

验证：配置可正确加载并传入客户端

### 5. 更新启动装配

修改 `cmd/agentd/main.go`：

- 按 `AGENT_MODEL_RACE_ENABLED` 选择 `RaceClient` 或 `FallbackClient`
- 传入熔断器配置

验证：启动日志显示当前模式（串行 fallback / 并行 race）

### 6. 增加测试并回归

- `circuit_test.go`：熔断器状态机全覆盖
- `race_test.go`：竞速模式 + 熔断器联动
- `fallback_test.go`：补充熔断器集成测试

验证：`go test ./...` 通过

## 模块影响

- `internal/model`：新增 `circuit.go`、`race.go`，修改 `fallback.go`
- `internal/config`：新增配置项
- `cmd/agentd`：启动装配更新

## 取舍

- 先实现双 provider 竞速，不扩展到多级级联
- 熔断器为进程内状态，不做跨进程持久化
- 并行竞速默认关闭，避免成本激增
- 不引入外部依赖，纯标准库实现
