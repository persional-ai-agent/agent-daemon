# ui-tui

独立 TUI 子工程（Go），通过 WebSocket 连接 `agentd`：

```bash
cd ui-tui
go run .
```

默认连接：

- `ws://127.0.0.1:8080/v1/chat/ws`

可选环境变量：

- `AGENT_API_BASE`：自定义 WS 地址
- `AGENT_HTTP_BASE`：自定义 HTTP 管理 API 地址（默认从 WS 地址自动推导）
- `AGENT_SESSION_ID`：指定会话 ID（默认自动生成）

命令：

- 输入任意文本发送一轮对话
- `/help` 查看命令
- `/session` 查看当前会话 ID
- `/session <id>` 切换会话 ID
- `/api` 查看当前 WS 地址
- `/api <ws-url>` 切换 WS 地址
- `/http` 查看当前 HTTP 地址
- `/http <http-url>` 切换 HTTP 地址
- `/tools` 列出工具
- `/tool <name>` 查看工具 schema
- `/sessions [n]` 查看最近会话
- `/pick <index>` 从最近一次 `/sessions` 结果中按序号切换会话
- `/show [sid] [offset] [limit]` 分页查看会话消息
- `/next` / `/prev` 基于最近一次 `/show` 做翻页
- `/stats [sid]` 查看会话统计
- `/gateway status|enable|disable` 网关状态与启停
- `/config get` 查看配置快照
- `/config set <section.key> <value>` 设置配置项
- `/pretty on|off` 开关 JSON 美化输出
- `/last` 查看最近一次 JSON 响应
- `/save <file>` 保存最近一次 JSON 响应到文件
- `/status` 查看最近一次命令状态
- `/quit` 或 `/exit` 退出

常用别名：

- `:q` / `quit` -> `/quit`
- `ls` -> `/tools`
- `show ...` -> `/show ...`
- `gw` / `gw ...` -> `/gateway status` / `/gateway ...`
- `cfg` / `cfg ...` -> `/config get` / `/config ...`
