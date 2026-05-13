# ui-tui 配置统一到 config.ini（Phase 12）

本次将 ui-tui 关键运行参数统一接入 `config/config.ini`，并新增 `[ui-tui]` 配置段，确保终端端参数可集中管理。

## 变更内容

- 配置文件新增 `[ui-tui]` 示例项：
  - `ws_base`
  - `http_base`
  - `ws_read_timeout_seconds`
  - `ws_turn_timeout_seconds`
  - `ws_reconnect_max`
  - `history_max_lines`
  - `event_max_items`
- `ui-tui/main.go` 改为从 `internal/config.Load()` 读取上述配置（环境变量仍可覆盖）。
- 配置加载路径增强：新增 `../config/config.ini` 兜底，适配在 `ui-tui/` 子目录执行 `go run .` 的场景。

## 测试

- `internal/config/config_test.go` 新增：
  - `[ui-tui]` 配置读取验证
  - `../config/config.ini` 路径发现验证
- 全量验证：
  - `go test ./...`
  - `./ui-tui/e2e_smoke.sh`
