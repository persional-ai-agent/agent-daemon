# 004 计划：审批护栏补齐

## 目标

在 terminal 工具中补齐“危险命令审批门禁”，形成 hardline 与危险命令分层控制。

## 实施步骤

1. 新增危险命令模式库
验证：可识别递归删除、高风险权限修改、远程脚本直管道执行等命令

2. 接入 terminal 门禁
验证：危险命令未设置 `requires_approval=true` 时拒绝执行

3. 保持 hardline 优先级
验证：hardline 命令即使设置 `requires_approval=true` 仍被阻断

4. 补充 schema 与返回字段
验证：`terminal` schema 包含 `requires_approval`，返回中保留该字段

5. 增加测试并回归
验证：新增审批门禁测试通过，`go test ./...` 通过
