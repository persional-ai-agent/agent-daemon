# ui-tui 易用性收口（Phase 13）

本轮一次性完成 ui-tui 易用性收口，覆盖审批体验、配置热更新、状态文件自修复与运维文档。

## 功能补齐

- 审批体验：
  - `/pending [n]`：支持查看最近 N 条待审批项
  - `/approve [id]` / `/deny [id]`：不传 id 时默认处理最近一条
- 配置热更新：
  - 新增 `/reload-config`，运行时重载 `config/config.ini` 的 `[ui-tui]` 参数
- 状态文件自修复：
  - `ui-tui-state.json` 解析失败时自动备份为 `ui-tui-state.json.corrupt.<timestamp>` 并重建
- 运维文档：
  - 新增 `docs/ui-tui-ops.md`，覆盖网络/超时/鉴权/审批/状态文件等排障流程

## 测试

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过（已纳入 reload-config 与状态修复回归）
