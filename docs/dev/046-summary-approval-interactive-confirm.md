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
  → 返回: {"status":"pending_approval", "approval_id":"xxx", "command":"...", "reason":"..."}

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
