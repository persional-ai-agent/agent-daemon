# 0031 send summary merged

## 模块

- `send`

## 类型

- `summary`

## 合并来源

- `0040-send-summary-merged.md`

## 合并内容

### 来源：`0040-send-summary-merged.md`

# 0040 send summary merged

## 模块

- `send`

## 类型

- `summary`

## 合并来源

- `0110-send-message-home-channel.md`

## 合并内容

### 来源：`0110-send-message-home-channel.md`

# 111 Summary - send_message 支持 HOME_CHANNEL 默认目标（Hermes 体验对齐）

## 变更

- `send_message(action="send")`：当未提供 `chat_id`，且不在 gateway 上下文中时，会尝试读取默认目标：
  - `TELEGRAM_HOME_CHANNEL`
  - `DISCORD_HOME_CHANNEL`
  - `SLACK_HOME_CHANNEL`
  - `YUANBAO_HOME_CHANNEL`

用于 CLI/API 场景下的“默认投递目标”对齐 Hermes 体验。

## 修改文件

- `internal/tools/send_message.go`
