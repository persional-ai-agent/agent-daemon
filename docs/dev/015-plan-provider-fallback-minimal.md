# 015 计划：Provider 故障切换最小补齐

## 目标

在不改动 Agent Loop 的前提下，新增主备 provider 自动切换能力，提升可用性。

## 实施步骤

1. 在 `internal/model` 新增 `FallbackClient`：
   - 主调用成功直接返回
   - 主调用失败且为可重试错误时调用备用 provider
2. 新增回退判定规则：
   - 状态码：`408/429/500/502/503/504`
   - 网络超时与常见连接错误
3. 新增配置项：`AGENT_MODEL_FALLBACK_PROVIDER`
4. 在 `cmd/agentd` 的模型装配中启用 fallback 包装。
5. 新增模型层与装配层测试。
6. 运行 `go test ./...` 回归。

## 验证标准

- 主 provider 成功时不触发 fallback
- 可重试错误时 fallback 生效
- 非可重试错误时直接返回主错误
- 全量测试通过
