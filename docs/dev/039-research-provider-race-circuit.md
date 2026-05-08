# 039 调研：Provider 并行竞速与熔断

## 背景

当前 `FallbackClient` 采用串行降级策略：主 provider 失败后才尝试备用 provider。这在高可用场景下存在两个问题：

1. **延迟叠加**：主 provider 超时后才触发 fallback，总延迟 = 主超时 + 备用响应
2. **无状态感知**：连续失败后仍会尝试已故障的 provider，浪费请求配额

## 缺口分析

### 当前能力

- 串行 fallback：主失败 → 备用
- 基于错误码/超时判断是否 fallback
- 无 provider 健康状态记忆

### 缺失能力

- 并行竞速：同时向多个 provider 发请求，取最快响应
- 熔断器：连续失败后临时隔离 provider，避免无效请求
- 半开探测：熔断后自动试探性恢复
- 成本感知：优先使用低成本 provider，高成本仅作为竞速备选

## 方案对比

### 方案 A：纯并行竞速

**做法**：同时向 N 个 provider 发请求，`select` 取第一个成功响应，取消其余。

**优点**：
- 延迟最低（取最快者）
- 实现简单

**缺点**：
- 每次请求都消耗多个 provider 配额
- 成本高（N 倍 token 消耗）
- 无故障隔离，某个 provider 持续故障仍会被调用

**适用场景**：对延迟极度敏感、成本不敏感的场景

### 方案 B：熔断器 + 串行 fallback（当前方案的增强）

**做法**：在现有 `FallbackClient` 基础上增加熔断器状态机：

- **Closed**（正常）：正常调用，失败计数增加
- **Open**（熔断）：连续失败达到阈值后，跳过该 provider，直接 fallback
- **Half-Open**（半开）：熔断超时后，允许一次试探请求，成功则恢复 Closed，失败则继续 Open

**优点**：
- 成本可控（不并行发请求）
- 自动隔离故障 provider
- 实现复杂度适中

**缺点**：
- 延迟与当前 fallback 相同（串行）
- 需要维护状态（线程安全）

**适用场景**：成本敏感、可接受串行延迟的场景

### 方案 C：熔断器 + 可选并行竞速（推荐）

**做法**：融合方案 A 和 B：

- 默认使用熔断器 + 串行 fallback
- 可选开启并行竞速模式（通过配置 `AGENT_MODEL_RACE_ENABLED=true`）
- 并行竞速时，仅对未熔断的 provider 发请求
- 竞速失败会增加该 provider 的失败计数

**优点**：
- 兼顾成本与延迟（用户可选择）
- 故障隔离（熔断器保护）
- 灵活配置

**缺点**：
- 实现复杂度较高
- 需要管理并发状态

**适用场景**：通用场景，用户可根据需求选择模式

## 推荐方案

采用 **方案 C**，理由：

1. 向后兼容：默认行为与当前 fallback 一致
2. 渐进增强：用户可按需开启并行竞速
3. 故障隔离：熔断器保护所有模式
4. 可观测性：暴露 provider 健康状态，便于调试

## 核心设计

### 熔断器状态机

```
Closed ──(连续失败≥阈值)──> Open
  ↑                           │
  │                      (熔断超时)
  │                           ↓
  │                       Half-Open
  │                           │
  └──(试探成功)───────────────┘
  │
  └──(试探失败)──> Open
```

### 配置项

- `AGENT_MODEL_RACE_ENABLED`：是否开启并行竞速（默认 `false`）
- `AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD`：连续失败阈值（默认 `3`）
- `AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS`：熔断恢复超时（默认 `60`）
- `AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS`：半开状态最大试探请求数（默认 `1`）

### 数据结构

```go
type CircuitState int

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

type ProviderCircuit struct {
    mu                sync.RWMutex
    state             CircuitState
    failureCount      int
    successCount      int
    lastFailureTime   time.Time
    lastStateChange   time.Time
    threshold         int
    recoveryTimeout   time.Duration
    halfOpenMaxReqs   int
    halfOpenReqs      int
}

type RaceClient struct {
    Primary      Client
    PrimaryName  string
    Fallback     Client
    FallbackName string
    PrimaryCircuit *ProviderCircuit
    FallbackCircuit *ProviderCircuit
    RaceEnabled  bool
}
```

### 执行流程

**串行模式（默认）**：
1. 检查主 provider 熔断器状态
2. 若 Open，直接走 fallback
3. 若 Closed/Half-Open，调用主 provider
4. 成功 → 重置失败计数；失败 → 增加失败计数，可能触发熔断
5. 若主失败且可 fallback，重复 2-4 走备用 provider

**并行竞速模式**：
1. 检查所有 provider 熔断器状态
2. 过滤掉 Open 状态的 provider
3. 向剩余 provider 并发发请求
4. `select` 取第一个成功响应
5. 成功者重置失败计数，失败者增加失败计数
6. 取消其余进行中的请求

## 结论

该方案可在保持向后兼容的前提下，为 Provider 层增加故障隔离与延迟优化能力，属于 L3 级别需求（跨模块、影响关键链路）。
