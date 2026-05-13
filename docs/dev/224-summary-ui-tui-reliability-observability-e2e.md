# ui-tui 稳定性与可观测性一次性补齐（Phase 11）

本阶段一次性补齐了 ui-tui 在长连接稳定性、诊断可观测性、长会话容量治理与回归自测上的缺口，全部基于 Go 实现，不依赖前端页面。

## 补齐内容

- 事件流健壮性：
  - WebSocket 断线自动重连（最多 2 次）
  - 重连保持同 `session_id`，并携带 `turn_id` 与 `resume` 标志
  - 45s 读超时提示（等待中），8m 单轮超时中断
- 长会话性能：
  - 本地历史命令文件滚动上限 2000 行
  - 内存事件日志滚动上限 2000 条
  - `/history`、`/events` 的读取请求自动受上限裁剪
- 错误可诊断性：
  - 统一错误分类：`network/timeout/auth/request/server/unknown`
  - 提示符显示 `状态/错误码`，`/status` 输出 `status/code/detail`
- 端到端回归：
  - 新增 `ui-tui/e2e_smoke.sh`，覆盖命令面烟测与可选后端健康联通路径
- 文档补齐：
  - `ui-tui/README.md`
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`

## 验证

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过（后端不可达时自动跳过 health 联通子场景）
