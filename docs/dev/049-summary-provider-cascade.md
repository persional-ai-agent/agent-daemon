# 049 总结：Provider 多级级联与成本感知

## 变更摘要

1. 新增 `CascadeClient` 支持 N 级 provider 级联回退
2. 支持 `cost_aware` 模式按成本权重排序后依次尝试
3. 每级独立熔断器，跳过已开路的 provider
4. 通过 `AGENT_MODEL_CASCADE` 环境变量配置

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/model/cascade.go` | 新建：`CascadeClient` + `ProviderEntry` + `ParseCascadeProviders` |
| `internal/model/cascade_test.go` | 新建：6 个测试用例 |
| `internal/config/config.go` | 新增 `ModelCascade` + `ModelCostAware` |
| `cmd/agentd/main.go` | `buildModelClient` 优先检查 cascade 模式；新增 `buildCascadeClient` |

## 新增能力

### CascadeClient

```go
c := NewCascadeClientWithCircuit([]ProviderEntry{
    {Client: openai, Name: "openai", Cost: 1.0},
    {Client: anthropic, Name: "anthropic", Cost: 2.5},
    {Client: codex, Name: "codex", Cost: 3.0},
}, true, 3, 60*time.Second, 1)
```

- 顺序（cost_aware=false）：按列表顺序依次尝试
- 成本感知（cost_aware=true）：按 Cost 升序排列后依次尝试
- 每级独立熔断器，短路失败快速回退
- 跳过开路 provider，emit `circuit_state` 事件

### 环境变量

```bash
# 级联列表：provider:cost,provider:cost,...
AGENT_MODEL_CASCADE=openai:1.0,anthropic:2.5,codex:3.0

# 启用成本感知排序
AGENT_MODEL_COST_AWARE=true
```

### 与 FallbackClient 的关系

- `AGENT_MODEL_CASCADE` 未设置 → 沿用现有 fallback/race 模式
- `AGENT_MODEL_CASCADE` 已设置 → 优先使用级联模式
- `AGENT_MODEL_COST_AWARE` 控制排序

## 测试结果

`go build ./...` ✅ | `go test ./...` 全部通过 ✅ | `go vet ./...` 无警告 ✅
