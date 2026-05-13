# 262-summary-cli-tui-source-fallback-and-boot-message

本轮继续收口 CLI/TUI 差异项：提升 `agentd tui` 在“独立 ui-tui 不可执行”场景下的可用性，并补齐首条消息透传能力。

## 变更

- `cmd/agentd/main.go`
  - `agentd tui` 独立模式启动链路增强：
    - 先找二进制：`AGENT_UI_TUI_BIN` -> `PATH(ui-tui)` -> `./ui-tui/tui.run`
    - 若都不可用，新增源码回退：`go run ./ui-tui`（仓库根目录）
  - `agentd tui -message` 在独立 `ui-tui` 路径下生效：
    - 通过环境变量 `AGENT_UI_TUI_BOOT_MESSAGE` 透传给 `ui-tui`
  - 新增函数：
    - `buildUITUICommand`
    - `resolveUITUISourceDir`

- `ui-tui/main.go`
  - 启动时识别 `AGENT_UI_TUI_BOOT_MESSAGE`，自动发送首条对话。

- 测试
  - `cmd/agentd/main_tui_test.go`
    - 新增 `resolveUITUISourceDir` 分支测试。

- 文档
  - `docs/frontend-tui-user.md`
  - `docs/frontend-tui-dev.md`
  - `ui-tui/README.md`

## 验证

- `go test ./cmd/agentd -count=1`
- `go test ./ui-tui -count=1`
- `go test ./...`

