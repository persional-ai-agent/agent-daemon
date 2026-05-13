# ui-tui

独立 TUI 子工程（Node.js），通过 WebSocket 连接 `agentd`：

```bash
cd ui-tui
npm install
npm start
```

默认连接：

- `ws://127.0.0.1:8080/v1/chat/ws`

可选环境变量：

- `AGENT_API_BASE`：自定义 WS 地址
- `AGENT_SESSION_ID`：指定会话 ID（默认自动生成）

命令：

- 输入任意文本发送一轮对话
- `/help` 查看命令
- `/session` 查看当前会话 ID
- `/session <id>` 切换会话 ID
- `/api` 查看当前 WS 地址
- `/api <ws-url>` 切换 WS 地址
- `/quit` 或 `/exit` 退出
