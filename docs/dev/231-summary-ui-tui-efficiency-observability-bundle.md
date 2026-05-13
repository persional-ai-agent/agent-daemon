# ui-tui 一次性优化补全（Phase 17）

本轮一次性完成了交互效率、可读性、恢复建议、安全审计、测试矩阵、配置治理与发布体验补全。

## 本轮补全

- 交互效率：
  - `/sessions` 支持列出后立即交互选择切换会话
  - `/pending [n]` 支持交互选择并直接 approve/deny
  - `/show` 支持消息索引交互提示
- 输出可读性：
  - 新增 `/view human|json` 视图模式切换
- 错误恢复：
  - `/approve`、`/deny`、`/events save`、`/save`、`/cancel` 失败时输出 retry suggestion
- 安全与审计：
  - 关键操作审计日志 `~/.agent-daemon/ui-tui-audit.log`
  - 覆盖 `approve`/`deny`/`cancel`/`config set`
- 可测试性：
  - 扩展 mock 后端测试矩阵（含缺失 approval endpoint 的 doctor 场景）
- 配置治理：
  - 新增 `/config tui` 显示 ui-tui 生效配置与来源（env/config）
  - `[ui-tui]` 增加 `view_mode`、`auto_doctor`
- 发布体验：
  - 新增 `ui-tui/release.sh` 单文件构建脚本
  - 新增 `/version` 输出构建元信息

## 验证

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过
