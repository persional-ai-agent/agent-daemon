# Frontend / TUI 使用文档

## 1. Web Dashboard

启动后端：

```bash
agentd serve
```

启动前端：

```bash
cd web
npm install
npm run dev
```

默认连接 `http://127.0.0.1:8080`。

页面说明：

- `chat`：支持普通请求与流式请求（SSE），可查看事件时间线。
- `sessions`：会话列表 + 详情分页查看。
- `tools`：工具列表 + schema 详情查看（支持筛选）。
- `gateway`：查看网关状态，可启用/禁用并刷新。
- `config`：查看配置快照，可写入单个 `section.key=value`。

## 2. CLI 类 TUI

进入交互式会话：

```bash
agentd chat
```

进入增强 TUI 模式（带实时事件轨迹）：

```bash
agentd tui
```

使用独立 TUI 子工程：

```bash
cd ui-tui
go run .
```

常用命令：

- `/help`：命令列表
- `/tools`：工具清单
- `/sessions [n]`：最近会话
- `/stats [session_id]`：会话统计
- `/show [sid] [offset] [limit]`：消息分页查看
- `/history [n]`：当前上下文预览
- `/reload`：从存储重载上下文
- `/clear`：清空当前进程内上下文
- `/quit`：退出

`ui-tui` 额外命令（管理面）：

- `/api`、`/api <ws-url>`：查看/切换 WS 地址
- `/http`、`/http <http-url>`：查看/切换 HTTP 管理地址
- `/tool <name>`：查看工具 schema
- `/gateway status|enable|disable`：网关状态与启停
- `/config get`、`/config set <section.key> <value>`：配置查看与设置
- `/health`、`/cancel`：健康检查与当前会话中断
- `/history [n]`、`/rerun <index>`：本地历史查看与命令重放
- `/events [n]`、`/events save <file>`：运行事件查看与导出
- `/events save <file> [json|ndjson] [since=<RFC3339>] [until=<RFC3339>]`：按格式和时间范围导出
- `/bookmark add|list|use`：会话配置书签
- `/pending`、`/approve <id>`、`/deny <id>`：终端内审批闭环

稳定性与排障：

- 提示符会显示最近状态和错误码，例如 `tui[err/network]`。
- `/status` 可查看 `status/code/detail` 三元组，快速区分网络、超时、鉴权、请求参数或服务端错误。
- 长连接在短暂断线时会自动重连（同 session 恢复），并给出重连提示。
- 启动时会自动恢复上次会话与 endpoint。
- 运行烟测：`./ui-tui/e2e_smoke.sh`。
