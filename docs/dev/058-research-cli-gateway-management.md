# 058 Research：CLI 网关管理最小对齐

## 背景

Hermes 提供 `hermes gateway` 入口管理消息网关。当前 Go 项目已有 Telegram、Discord、Slack 网关适配器和 `AGENT_GATEWAY_ENABLED` / `gateway.enabled` 配置，但缺少专用 CLI 管理入口。

## 目标

补齐最小网关管理面：

- 查看网关是否启用。
- 查看已配置 token 的平台。
- 列出支持的平台。
- 写入 `gateway.enabled=true/false`。

## 范围

- 不启动或停止运行中的进程。
- 不写入平台 token，避免专用命令处理 secret；token 继续通过 `agentd config set gateway.telegram.bot_token ...` 等方式配置。
- 不实现 Hermes 的 pairing、setup wizard、token lock 或平台级状态探测。

## 推荐方案

在 `cmd/agentd` 增加 `gateway status|platforms|enable|disable`。`status` 默认输出文本，支持 `-json`，并可通过 `-file` 指定配置文件。

## 三角色审视

- 高级产品：提供用户最需要的网关开关和状态查看，不伪装成完整 Gateway setup。
- 高级架构师：复用现有配置系统和 Gateway 支持平台，不启动外部连接。
- 高级工程师：helper 测试覆盖支持平台与已配置平台判断。
