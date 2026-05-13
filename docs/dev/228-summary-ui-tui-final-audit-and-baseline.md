# ui-tui 最终审计与基线标记（Phase 14）

本轮完成了 ui-tui 的最终收尾动作：文档命令面审计、真实环境回归执行、发布基线标记。

## 执行结果

- 命令面与文档审计：
  - 已核对 `ui-tui/main.go` 的 `/help` 命令列表与 `ui-tui/README.md`、`docs/ui-tui-ops.md`、`docs/frontend-tui-user.md`。
  - 对齐结果：一致；补充了兼容性风险说明。
- 真实环境回归：
  - 执行命令链：`/reload-config`、`/health`、`/status`、`/pending 3`、`/approve`、`/deny`、`/events save ...`。
  - 结果：
    - 基础链路（reload/health/status/events）通过。
    - 发现两项环境兼容风险：
      1. `/pending` 在该环境返回 500（`converting NULL to int64`）。
      2. `/approve`/`/deny` 返回 404（后端未启用 `POST /v1/ui/approval/confirm`）。
  - 已将处理建议写入 `docs/ui-tui-ops.md`。
- 发布基线：
  - 创建里程碑标签：`ui-tui-parity-v1`

## 结论

ui-tui 当前代码基线已完成能力收口；剩余问题主要是“运行环境后端版本/历史数据兼容性”而非 ui-tui 端功能缺失。
