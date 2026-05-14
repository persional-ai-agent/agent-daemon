# 274 总结：CLI/TUI 状态化命令面补齐

## 背景

用户要求一次性完成 Hermes 差异清单中的第 1 项：CLI/TUI 体验补齐。当前不重写 Hermes 全量 TUI 前端，而是在现有 Go 入口内完成可验证的 CLI/TUI 核心命令闭环。

## 实现结果

- `internal/cli/chat.go` 引入 `chatState`，slash 命令可真实切换 `session_id`、重置/加载上下文，并保持 system prompt 与历史状态一致。
- 新增会话命令：`/new`、`/reset`、`/resume`、`/retry`、`/undo`、`/compress`、`/save`。
- 扩展管理命令：`/tools list|show|schemas`、`/toolsets list|show|resolve`、`/todo`、`/memory`、`/model`、`/status`、`/commands`。
- `internal/cli/tui.go` 的 lite TUI 事件输出扩展到 user、turn、model stream、assistant、tool、MCP、delegate、context compact、completed/error 等事件。
- 补充 CLI 单元测试覆盖新建/恢复、撤销/压缩、工具 schema 查看、重试等核心路径。

## 边界

- `/undo` 和 `/compress` 作用于当前进程内上下文，不回写删除 SQLite 历史，避免引入会话存储破坏性语义。
- `/retry` 会基于上一条 user 消息重新运行，并将新的上下文用于后续交互；持久化层会追加新的运行记录。
- 仍未复刻 Hermes 的完整高级 TUI 前端、模型选择器 UI、插件 slash 命令生态和全部快捷键。

## 验证

```bash
GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./...
```

结果：通过。
