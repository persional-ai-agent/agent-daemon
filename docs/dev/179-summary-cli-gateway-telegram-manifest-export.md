# 179 - Summary - CLI gateway Telegram manifest 导出

## 本次变更

- 新增 `agentd gateway manifest -platform telegram [-json]`，导出 Telegram `setMyCommands` 命令清单与 BotFather 可用的命令列表。
- `internal/gateway/platforms/telegram.go` 导出 `TelegramCommands()`，CLI 与运行时命令注册复用同一份定义，避免两套清单漂移。
- README、产品文档、开发文档同步更新 Telegram manifest 导出入口。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway manifest -platform telegram -json`

## 结果

- Telegram 现在和 Slack / Discord 一样，具备最小命令清单导出能力。
- Gateway 平台对齐文档更新为“Telegram 已具备原生命令菜单 + 审批按钮 + manifest 导出”。
