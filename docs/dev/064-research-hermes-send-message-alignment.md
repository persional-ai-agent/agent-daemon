# 064 调研：Hermes send_message 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 的 `send_message`（`tools/send_message_tool.py`）是跨平台投递工具：

- 可 `action=list` 展示可投递目标（channel directory）。
- `action=send` 支持平台 target 解析、频道名解析、媒体附件抽取等。
- 依赖 gateway platform adapters（Telegram/Discord/Slack…）与配置。

## 当前项目差异

本项目已有最小 Gateway（Telegram/Discord/Slack）用于“接收消息 -> 运行 agent -> 回复”，但缺少：

- agent 主循环可调用的跨平台投递工具。
- gateway adapters 的运行时注册/查询机制（供工具调用）。

## 最小对齐目标（本次）

- 提供 `send_message` 工具：
  - `action=list` 返回当前已连接的 adapter 平台名列表。
  - `action=send` 通过运行时 adapter 直接发送文本到指定 `platform + chat_id`。
- 将 adapter 接口从 gateway 包中解耦，避免与 tools/agent 的 import cycle。

## 不在本次范围

- 频道目录（按名称解析 target）、媒体附件、线程/话题路由、重试策略与错误脱敏等。

