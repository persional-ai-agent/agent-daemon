# 0002 approval summary merged

## 模块

- `approval`

## 类型

- `summary`

## 合并来源

- `0001-approval-summary-merged.md`

## 合并内容

### 来源：`0001-approval-summary-merged.md`

# 0001 approval summary merged

## 模块

- `approval`

## 类型

- `summary`

## 合并来源

- `0003-approval-guardrails.md`
- `0040-approval-persistence.md`
- `0045-approval-interactive-confirm.md`

## 合并内容

### 来源：`0003-approval-guardrails.md`

# 004 总结：审批护栏补齐结果

## 已完成

- `internal/tools/safety.go` 新增危险命令模式识别（可审批）
- `terminal` 新增 `requires_approval` 门禁逻辑
- `terminal` schema 新增 `requires_approval` 字段
- hardline 规则保持最高优先级，不可审批放行
- 新增测试覆盖：
  - 危险命令未审批拒绝
  - 危险命令审批后放行
  - hardline 命令审批后仍拒绝

## 行为变化

- 命令命中危险模式但未设置 `requires_approval=true`：
  - 返回错误并拒绝执行
- 命中危险模式且设置 `requires_approval=true`：
  - 允许执行
- 命中 hardline 模式：
  - 永久阻断

## 验证

- `go test ./...` 通过

## 当前边界

本次已补齐“审批门禁核心逻辑”，但仍未实现：

- 交互式审批回调
- 审批持久化白名单
- 会话级审批状态管理

### 来源：`0040-approval-persistence.md`

# 041 总结：审批状态持久化与细粒度审批策略

## 变更摘要

1. 审批授权持久化到 SQLite，进程重启后可恢复
2. 支持按危险命令类别细粒度授权（pattern scope）
3. 保持向后兼容（session scope 行为与之前一致）

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/store/session_store.go` | 新增 `approvals` 表、`ApprovalRecord` 类型、`GrantApproval`/`RevokeApproval`/`ListApprovals`/`IsApproved`/`CleanupExpiredApprovals` |
| `internal/store/session_store_test.go` | 新增 6 个审批持久化测试 |
| `internal/tools/safety.go` | `dangerousCommandPatterns` 新增 `category` 字段；`detectDangerousCommand` 返回 `(category, description, dangerous)` |
| `internal/tools/approval_store.go` | 重写：新增 `patterns` 内存缓存、`persistent` 字段、`GrantPattern`/`RevokePattern`/`IsApprovedPattern`/`ListApprovals`/`LoadFromStore`；`NewPersistentApprovalStore` 构造函数 |
| `internal/tools/approval_store_test.go` | 新增 6 个细粒度审批测试 |
| `internal/tools/builtin.go` | `approval` 工具支持 `scope`/`pattern` 参数；`terminal` 审判判定增加 pattern 级检查；schema 新增 `scope`/`pattern` |
| `internal/tools/builtin_test.go` | 新增 5 个细粒度审批测试 |
| `cmd/agentd/main.go` | 使用 `NewPersistentApprovalStore` 替代 `NewApprovalStore` |

## 新增能力

### 审批持久化

- 授权记录写入 SQLite `approvals` 表
- 进程重启后，`IsApproved`/`IsApprovedPattern` 自动从 SQLite 恢复
- 过期记录通过 `CleanupExpiredApprovals` 惰性清理

### 细粒度审批

- `scope=session`（默认）：行为与之前一致，整个会话所有危险命令均可执行
- `scope=pattern`：仅授权指定类别的危险命令

危险命令类别标识符：

| 类别 | 描述 |
|------|------|
| `recursive_delete` | 递归删除命令 |
| `world_writable` | 世界可写权限修改 |
| `root_ownership` | 所有权变更为 root |
| `remote_pipe_shell` | 远程内容管道到 shell |
| `service_lifecycle` | 系统服务生命周期命令 |

### approval 工具新参数

- `scope`：`session`（默认）或 `pattern`
- `pattern`：当 `scope=pattern` 时必填，指定危险命令类别
- `status` 返回 `approvals` 列表，包含所有有效授权

## 审批判定优先级

1. hardline 命令 → 永久阻断
2. session 级授权 → 放行
3. pattern 级授权（匹配命令类别）→ 放行
4. `requires_approval=true` → 单次放行并自动 grant session 级
5. 无授权 → 拒绝

## 测试结果

`go test ./...` 全部通过，新增 17 个测试用例覆盖本次变更。

### 来源：`0045-approval-interactive-confirm.md`

# 046 总结：审批交互确认（pending + confirm）

## 变更摘要

1. 危险命令不再直接拒绝，而是返回 `pending_approval` 状态，携带 `approval_id`
2. `approval` 工具新增 `confirm` 动作，支持 approve/deny + 自动重新执行命令
3. 实现完整的交互式审批闭环：LLM 发现危险命令 → 请求用户确认 → 用户批准 → 执行

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/tools/builtin.go` | `BuiltinTools` 新增 `pendingApprovals` + `storePending`/`retrievePending`；`terminal` 危险命令返回 `pending_approval` 而非 error；`approval` 新增 `confirm` 动作（approve/deny + 自动执行命令）；schema 更新 |
| `internal/tools/builtin_test.go` | 更新 2 个旧测试、新增 4 个 confirm 测试 |

## 新的审批流程

```
Agent 调用 terminal("rm -rf /tmp/xxx")
  → 检测到危险命令，无预授权
  → 返回: {"success":false, "status":"pending_approval", "approval_id":"xxx", "command":"...", "reason":"..."}

LLM 收到 pending_approval 结果
  → 展示给用户: "需要运行: rm -rf /tmp/xxx。是否批准？"

用户说: "批准"
  → LLM 调用: approval(action=confirm, approval_id="xxx", approve=true)
  → 系统授予 session 级授权 + 自动重新执行命令
  → 返回命令执行结果

用户说: "拒绝"
  → LLM 调用: approval(action=confirm, approval_id="xxx", approve=false)
  → 清除 pending 记录，命令不执行
```

## `approval` 工具新增参数

| 参数 | 说明 |
|------|------|
| `action=confirm` | 确认 pending 审批 |
| `approval_id` | 来自 `terminal` 返回的 `approval_id` |
| `approve` | `true`=批准并执行，`false`=拒绝 |

## 测试结果

```
go test ./... -count=1 全部通过
go vet ./... 无警告
```

新增 4 个测试：
- `TestApprovalConfirmApproveAndExecute` — 确认批准后命令执行
- `TestApprovalConfirmDeny` — 拒绝不执行
- `TestApprovalConfirmMissingID` — 缺少 approval_id
- `TestApprovalConfirmUnknownID` — 无效 approval_id
