# 068 计划：Hermes execute_code 最小对齐

## 目标（可验证）

- 新增工具 `execute_code`，可运行 python 片段并返回 stdout/stderr/exit code。
- 限制在 workdir 内执行，支持超时。
- 单测覆盖基础执行成功路径。

## 实施步骤

1. 新增 `internal/tools/execute_code.go` 并注册到 engine。
2. toolsets 增加 `code_execution`（默认不纳入 core）。
3. 更新 docs/dev 索引与工具清单。

