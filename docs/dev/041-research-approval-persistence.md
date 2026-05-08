# 041 调研：审批状态持久化与细粒度审批策略

## 背景

当前审批系统已有：
- hardline 永久阻断（不可放行）
- `requires_approval` 单次显式放行
- 会话级审批状态（内存态 + TTL）
- `approval` 工具（`status`/`grant`/`revoke`）

但存在两个核心缺口：
1. **审批状态仅内存态**：进程重启后所有审批授权丢失，需要重新 grant
2. **审批粒度仅到 session 级**：grant 后该 session 所有危险命令均可执行，无法按命令模式或工具类型细粒度控制

## 当前实现分析

### ApprovalStore（内存态）

```go
type ApprovalStore struct {
    mu         sync.Mutex
    items      map[string]time.Time  // sessionID -> expiresAt
    defaultTTL time.Duration
}
```

- 按 `sessionID` 授权，粒度为整个会话
- 无持久化，进程重启丢失
- TTL 过期自动失效

### terminal 审批判定

```go
if reason, dangerous := detectDangerousCommand(command); dangerous {
    approved := tc.ApprovalStore != nil && tc.ApprovalStore.IsApproved(tc.SessionID)
    if !requiresApproval && !approved {
        return nil, fmt.Errorf("dangerous command requires approval: %s ...", reason)
    }
}
```

- 仅检查 session 级授权，不区分危险命令类型
- `requires_approval=true` 可单次放行并自动 grant session 级授权

## 缺口清单

1. **审批持久化**：授权记录应持久化到 SQLite，进程重启后可恢复
2. **命令模式级审批**：可按危险命令类别（如 `recursive_delete`、`world_writable`、`remote_pipe_shell`）授权，而非全量放行
3. **审批记录审计**：授权/撤销/使用记录可追溯

## 方案

### 1. 审批持久化

在现有 `sessions.db` 中新增 `approvals` 表：

```sql
CREATE TABLE IF NOT EXISTS approvals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'session',
    pattern TEXT NOT NULL DEFAULT '',
    granted_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
```

- `scope`：`session`（会话级）或 `pattern`（命令模式级）
- `pattern`：当 `scope=pattern` 时，存储危险命令类别（如 `recursive_delete`）
- `expires_at`：过期时间，过期记录在查询时惰性清理

### 2. 命令模式级审批

扩展 `approval` 工具：
- `grant` 新增可选 `scope` 和 `pattern` 参数
- `scope=session`（默认）：行为与当前一致
- `scope=pattern`：仅授权指定类别的危险命令

扩展 `detectDangerousCommand` 返回值，使其返回类别标识符（如 `recursive_delete`、`world_writable`）。

terminal 审批判定逻辑：
1. 先检查 session 级授权
2. 若无 session 级授权，检查 pattern 级授权（匹配命令类别）
3. 若均无，拒绝执行

### 3. 审批记录审计

`approval` 工具 `status` 返回当前有效授权列表，包含 scope 和 pattern。

## 取舍

- 不做用户侧交互确认 UI（保持工具级控制）
- 不做跨进程审批策略同步（单进程 SQLite 足够）
- 不做审批策略热重载（重启加载即可）
- 保持向后兼容：`scope=session` 行为与当前完全一致
