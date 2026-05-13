# ui-tui 对齐总结（管理命令补齐）

## 本阶段完成

在 Go 版 `ui-tui` 中补齐管理面命令，使其可覆盖 Web 管理页的核心操作：

1. 工具管理：`/tools`、`/tool <name>`
2. 会话管理：`/sessions [n]`、`/show [sid] [offset] [limit]`、`/stats [sid]`
3. 网关管理：`/gateway status|enable|disable`
4. 配置管理：`/config get`、`/config set <section.key> <value>`
5. 连接管理：`/http`、`/http <http-url>`（配合已有 `/api`）

## 结果

`ui-tui` 不依赖前端界面即可完成对话与运维管理的闭环操作。

## 验证

- `go test ./...` 通过。
