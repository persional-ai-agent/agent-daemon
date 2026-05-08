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
