# 0032 session plan merged

## 模块

- `session`

## 类型

- `plan`

## 合并来源

- `0041-session-plan-merged.md`

## 合并内容

### 来源：`0041-session-plan-merged.md`

# 0041 session plan merged

## 模块

- `session`

## 类型

- `plan`

## 合并来源

- `0008-session-approval-state.md`

## 合并内容

### 来源：`0008-session-approval-state.md`

# 009 计划：会话级审批状态补齐

## 目标

补齐会话级危险命令审批状态，实现 `grant -> 多命令复用 -> revoke/过期` 的最小闭环。

## 实施步骤

1. 新增审批状态存储
验证：支持按 `session_id` 授权、撤销、过期判断

2. 新增 `approval` 工具
验证：支持 `status` / `grant` / `revoke`

3. 接入 terminal 审批判定
验证：危险命令在会话已授权时可执行；hardline 仍不可放行

4. 增加配置项
验证：支持默认授权 TTL 配置

5. 增加测试并回归
验证：审批状态与 terminal 联动测试通过，`go test ./...` 通过
