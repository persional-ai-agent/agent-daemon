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
- `skills`：技能列表、详情编辑、新建、删除、搜索、同步与 reload。
- `agents`：delegate 会话、活动运行、历史记录、详情查询与中断。
- `cron`：创建定时任务，可设置结果投递目标与链式上下文模式，查看任务列表/详情，暂停、恢复、触发、删除任务，并查看运行记录。
- `models`：查看当前 provider/model/base_url、可用 provider，并写入新的模型选择。
- `plugins`：查看插件 dashboard slot 声明。
- `gateway`：查看网关状态与诊断信息，可启用/禁用并刷新。
- `voice`：查看语音状态，执行 voice/TTS/recording 控制。
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

`agentd tui` 默认 `-mode auto`：优先启动独立 `ui-tui`（完整命令面），若本机未安装 `ui-tui` 可执行文件则自动回退到内置 lite 模式。也可显式指定：

```bash
agentd tui -mode standalone
agentd tui -mode lite
agentd tui -mode standalone -fullscreen
```

在 `standalone/auto` 模式下，若无 `ui-tui` 二进制，`agentd tui` 会继续尝试源码回退（`go run ./ui-tui`，需在仓库根目录执行并有 Go 环境）。

使用独立 TUI 子工程：

```bash
cd ui-tui
go run .
```

常用命令：

- `/help`、`/commands`：命令列表
- `/status`、`/session`：当前会话状态
- `/new [session_id]`、`/reset [session_id]`：新建/重置并切换会话
- `/resume <session_id>`：加载并切换到已有会话
- `/retry`、`/undo`、`/compress [tail]`：重试、撤销、压缩当前上下文
- `/save [path]`：导出当前上下文为 JSON
- `/tools [list|show|schemas]`：工具清单、单工具 schema、全部 schema
- `/toolsets [list|show|resolve]`：工具集查看与解析
- `/sessions [n]`、`/stats [session_id]`、`/show [sid] [offset] [limit]`：会话列表、统计、分页查看
- `/history [n]`、`/todo`、`/memory [memory|user]`、`/model`：当前上下文、todo、记忆与模型状态
- `/reload`、`/clear`、`/tui`、`/quit`：重载、清空、TUI 状态和退出

`ui-tui` 额外命令（管理面）：

- `/api`、`/api <ws-url>`：查看/切换 WS 地址
- `/http`、`/http <http-url>`：查看/切换 HTTP 管理地址
- `/tool <name>`：查看工具 schema
- `/gateway status|enable|disable`：网关状态与启停
- `/config get`、`/config set <section.key> <value>`：配置查看与设置
- `/config tui`：查看 ui-tui 配置生效值与来源
- `/health`、`/cancel`：健康检查与当前会话中断
- `/history [n]`、`/rerun <index>`：本地历史查看与命令重放
- `/timeline [n]`：查看最近对话时间线（user/assistant/tool/result 摘要）
- `/events [n]`、`/events save <file>`：运行事件查看与导出
- `/events save <file> [json|ndjson] [since=<RFC3339>] [until=<RFC3339>]`：按格式和时间范围导出
- `/bookmark add|list|use`：会话配置书签
- `/workbench save|list|load|delete`：管理工作台配置方案（会话、endpoint、面板、刷新策略、视图）
- `/workflow save|list|run|delete`：管理命令编排方案（批量执行常用操作流）
- `/pending [n]`、`/approve [id]`、`/deny [id]`：终端内审批闭环（可默认处理最近一条）
- `/reload-config`：运行时重载 `[ui-tui]` 配置
- `/doctor`：后端能力预检（接口版本与连通性）
- `/actions`：打开快捷操作面板（编号选择常用动作）
- `/panel [name]`、`/panel next|prev`：切换全屏面板（overview/dashboard/sessions/tools/approvals/gateway/diag）
- `/panel list`：查看全屏面板清单
- `/panel status`、`/panel auto on|off`、`/panel interval <sec>`：管理全屏面板自动刷新策略
- `/refresh`：刷新当前全屏面板数据
- `/open <index>`：打开当前面板条目（会话切换/工具详情/审批执行）
- `/view human|json`：切换人类视图/JSON 视图
- `/fullscreen on|off`：运行时切换全屏看板
- `/version`：查看构建版本信息

稳定性与排障：

- 提示符会显示最近状态和错误码，例如 `tui[err/network]`。
- `/status` 可查看 `status/code/detail` 三元组，快速区分网络、超时、鉴权、请求参数或服务端错误。
- 长连接在短暂断线时会自动重连（同 session 恢复），并给出重连提示。
- 启动时会自动恢复上次会话与 endpoint。
- 若状态文件损坏，会自动备份并重建，避免启动失败。
- 运行烟测：`./ui-tui/e2e_smoke.sh`。
- 跳过启动自检：`go run ./ui-tui --no-doctor`。
