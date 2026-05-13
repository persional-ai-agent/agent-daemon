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
