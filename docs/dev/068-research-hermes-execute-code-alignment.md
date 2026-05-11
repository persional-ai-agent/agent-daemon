# 068 调研：Hermes execute_code 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 的 `execute_code` 允许用脚本方式编排工具调用（减少多轮 LLM 往返）。

## 当前项目差异

Go 版此前没有 `execute_code`，复杂 pipeline 只能通过多轮 tool calls 完成。

## 最小对齐目标（本次）

- 新增 `execute_code`：执行短 Python 代码片段，返回 stdout/stderr/exit_code。
- 执行目录受 `AGENT_WORKDIR` 限制，并支持 `timeout_seconds`。

## 边界

- 当前 `execute_code` 仅做“本地脚本执行”，不具备 Hermes 那种脚本内部调用 tools 的 RPC 编排能力。

