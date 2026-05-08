# 041 计划：审批状态持久化与细粒度审批策略

## 目标

1. 审批授权持久化到 SQLite，进程重启后可恢复
2. 支持按危险命令类别细粒度授权
3. 保持向后兼容

## 实施步骤

### 1. 扩展 SessionStore 新增 approvals 表

修改 `internal/store/session_store.go`：
- `init()` 新增 `approvals` 表创建
- 新增 `GrantApproval(sessionID, scope, pattern string, expiresAt time.Time) error`
- 新增 `RevokeApproval(sessionID string, scope string, pattern string) error`
- 新增 `ListApprovals(sessionID string) ([]ApprovalRecord, error)`
- 新增 `IsApproved(sessionID string, scope string, pattern string) (bool, error)`
- 新增 `CleanupExpiredApprovals() error`

验证：approval 表可正确创建和操作

### 2. 定义 ApprovalRecord 类型

在 `internal/store/session_store.go` 新增：
```go
type ApprovalRecord struct {
    ID        int64
    SessionID string
    Scope     string    // "session" or "pattern"
    Pattern   string    // 危险命令类别标识符
    GrantedAt time.Time
    ExpiresAt time.Time
}
```

### 3. 扩展 detectDangerousCommand 返回类别标识符

修改 `internal/tools/safety.go`：
- `detectDangerousCommand` 返回 `(category string, description string, dangerous bool)`
- `category` 为机器可读标识符（如 `recursive_delete`、`world_writable`、`root_ownership`、`remote_pipe_shell`、`service_lifecycle`）

验证：每个危险命令模式返回唯一类别标识符

### 4. 扩展 ApprovalStore 支持持久化

修改 `internal/tools/approval_store.go`：
- 新增 `PersistentApprovalStore`，组合内存缓存 + SQLite 持久化
- `Grant` 同时写内存和 SQLite
- `IsApproved` 先查内存，miss 时查 SQLite
- `Revoke` 同时删内存和 SQLite
- 新增 `LoadFromStore(sessionID string)` 从 SQLite 恢复到内存

验证：持久化 store 的 grant/revoke/status 行为与内存版一致

### 5. 扩展 approval 工具

修改 `internal/tools/builtin.go`：
- `grant` 新增可选 `scope`（默认 `session`）和 `pattern` 参数
- `status` 返回当前有效授权列表，包含 scope 和 pattern
- `revoke` 新增可选 `scope` 和 `pattern` 参数（不指定则撤销全部）

验证：`grant scope=pattern pattern=recursive_delete` 仅授权递归删除类命令

### 6. 修改 terminal 审批判定逻辑

修改 `internal/tools/builtin.go`：
- 危险命令判定时，先检查 session 级授权，再检查 pattern 级授权
- 使用 `detectDangerousCommand` 返回的 category 匹配 pattern

验证：pattern 级授权仅放行匹配类别的危险命令

### 7. 接入启动装配

修改 `cmd/agentd/main.go`：
- 使用 `PersistentApprovalStore` 替代内存版
- 启动时传入 SessionStore 用于持久化

验证：重启后审批授权仍有效

### 8. 增加测试并回归

- 持久化 store 测试
- 细粒度授权测试
- 重启恢复测试
- `go test ./...` 通过

## 模块影响

- `internal/store/session_store.go`
- `internal/tools/approval_store.go`
- `internal/tools/safety.go`
- `internal/tools/builtin.go`
- `internal/agent/loop.go`
- `cmd/agentd/main.go`

## 向后兼容

- `scope=session` 行为与当前完全一致
- 不指定 `scope`/`pattern` 时默认为 session 级授权
- 现有 `requires_approval` 单次放行逻辑不变
