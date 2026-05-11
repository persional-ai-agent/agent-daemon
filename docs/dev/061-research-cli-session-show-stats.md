# 061 Research：CLI 会话详情查看与统计

## 背景

Hermes 的 session store/CLI 通常支持：

- 列出最近会话
- 搜索历史
- 查看某个 session 的消息（分页）
- 查看会话统计（消息数、时间范围、工具相关计数等）

Go 版 `agent-daemon` 目前已有 `sessions list/search`，且 SQLite `messages` 表里保存了足够信息来做最小 `show/stats`，但缺少 CLI 入口。

## 目标

补齐最小可用的 CLI：

- `agentd sessions show <session_id>`：分页查看消息。
- `agentd sessions stats <session_id>`：输出统计信息，便于排障与外部 UI 取数。

## 范围与非目标

- 不做会话删除/导出（Hermes 有 JSON snapshot 能力，这里先不做）。
- 不做基于 LLM 的摘要与跨会话语义检索（仍保持关键词检索）。
- 默认输出 JSON（与现有 `config/model/tools/gateway` 命令保持一致）。

## 方案

- 复用 `internal/store.SessionStore`：
  - `LoadMessagesPage(sessionID, offset, limit)`
  - `SessionStats(sessionID)`
- `cmd/agentd` 增加子命令：
  - `sessions show [-offset N] [-limit N] session_id`
  - `sessions stats session_id`

