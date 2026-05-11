# 060 Research：CLI 会话列表与检索

## 背景

Hermes 有跨会话检索能力，且通常提供直接的 CLI 使用面。当前 Go 项目已有 SQLite 会话存储和 `session_search` 工具，但缺少直接的命令行入口查看/检索历史。

## 目标

提供最小 CLI：

- 列出最近 session（按最新消息排序）。
- 按关键词搜索历史消息内容（优先使用 FTS5，缺失时回退 LIKE）。

## 范围

- 不做 LLM 摘要。
- 不做会话导出/删除。
- 仅本地 `sessions.db`。

## 方案

- `internal/store.SessionStore` 增加 `ListRecentSessions(limit)`。
- `cmd/agentd` 增加 `sessions list` / `sessions search` 子命令，默认输出 JSON。
- `sessions search` 支持 `-exclude session_id` 排除当前会话。
