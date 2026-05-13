# ui-tui

独立 TUI 子工程（Go），通过 WebSocket 连接 `agentd`：

```bash
cd ui-tui
go run .
```

跳过启动自检：

```bash
go run . --no-doctor
```

全屏看板模式：

```bash
go run . --fullscreen
```

默认连接：

- `ws://127.0.0.1:8080/v1/chat/ws`

可选环境变量：

- `AGENT_API_BASE`：自定义 WS 地址
- `AGENT_HTTP_BASE`：自定义 HTTP 管理 API 地址（默认从 WS 地址自动推导）
- `AGENT_SESSION_ID`：指定会话 ID（默认自动生成）
- `AGENT_UI_TUI_BOOT_MESSAGE`：启动后自动发送首条消息（由 `agentd tui -message` 注入）
- `AGENT_UI_TUI_FULLSCREEN`：设为 `1/true` 可启用全屏看板模式

配置文件：

- 支持从 `config/config.ini` 的 `[ui-tui]` 读取运行参数（也支持 `../config/config.ini`，适配在 `ui-tui/` 子目录启动）。
- 环境变量优先级高于配置文件。

运行时行为（默认）：

- WS 读超时提示：45s 未收到事件会提示“等待服务端响应中”
- 单轮最长等待：8m（超时后返回 `timeout`）
- 自动重连：最多 2 次，重连时保持同一 `session_id`，并携带 `turn_id`/`resume`
- 历史命令上限：2000 行（滚动清理）
- 事件日志上限：2000 条（滚动清理）

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
- `/config tui` 查看 ui-tui 生效配置与来源（env/config）
- `/pretty on|off` 开关 JSON 美化输出
- `/view human|json` 切换人类视图/JSON 视图
- `/last` 查看最近一次 JSON 响应
- `/save <file>` 保存最近一次 JSON 响应到文件
- `/status` 查看最近一次命令状态
- `/fullscreen` 查看全屏模式状态
- `/fullscreen on|off` 运行时切换全屏看板
- `/health` 查看后端健康状态
- `/cancel` 取消当前会话中的运行任务
- `/history [n]` 查看本地命令历史
- `/timeline [n]` 查看最近对话时间线（user/assistant/tool/result 摘要）
- `/rerun <index>` 重新执行历史命令
- `/events [n]` 查看最近运行事件
- `/events save <file> [json|ndjson] [since=<RFC3339>] [until=<RFC3339>]` 保存事件日志到文件
- `/bookmark add <name> [sid]` 保存会话书签
- `/bookmark list` 查看会话书签
- `/bookmark use <name>` 切换到书签会话
- `/workbench save <name>` 保存当前工作台配置（session/api/panel/refresh/view）
- `/workbench list` 列出工作台配置
- `/workbench load <name>` 加载工作台配置
- `/workbench delete <name>` 删除工作台配置
- `/workflow save <name> <cmd1;cmd2;...>` 保存命令编排
- `/workflow list` 列出命令编排
- `/workflow run <name> [dry]` 执行/预演命令编排
- `/workflow delete <name>` 删除命令编排
- `/pending [n]` 查看最近待审批项（支持列表）
- `/approve [approval_id]` 同意待审批项（不传 id 默认最近一条）
- `/deny [approval_id]` 拒绝待审批项（不传 id 默认最近一条）
- `/reload-config` 运行时重载 `[ui-tui]` 配置
- `/doctor` 后端能力预检（health/sessions/approval/ws/config）
- `/actions` 打开快捷操作面板（编号选择常用管理动作）
- `/panel [name]` 切换全屏面板（overview/dashboard/sessions/tools/approvals/gateway/diag）
- `/panel list` 列出全屏面板
- `/panel next|prev` 循环切换全屏面板
- `/panel status` 查看面板运行状态（自动刷新、间隔、最近刷新时间）
- `/panel auto on|off` 开关面板自动刷新
- `/panel interval <sec>` 设置面板刷新间隔（1..300 秒）
- `/open <index>` 打开当前面板项（sessions/tools/approvals）
- `/refresh` 刷新当前全屏面板数据
- `/diag` 查看实时诊断（transport/reconnect/fallback/error）
- `/diag export <file>` 导出诊断包（含 recent_events）
- `/version` 查看 ui-tui 构建版本信息
- `/quit` 或 `/exit` 退出

状态诊断：

- 提示符：`tui[ok/ok]`、`tui[err/network]` 这类 `状态/错误码` 组合
- `/status` 输出：`status=<ok|err> code=<ok|network|timeout|auth|request|server|unknown> detail=<详情>`
- `/diag` 输出：`active_transport/reconnect_count/fallback_hint/last_error_code` 等实时字段
- 诊断包 schema：`diag.v1`（见 `docs/api/diagnostics.bundle.schema.json`，与 Web 导出对齐）
- 启动时自动恢复最近会话与 endpoint（`~/.agent-daemon/ui-tui-state.json`）
- 同步恢复全屏与面板偏好（`fullscreen/fullscreen_panel/panel_auto/panel_interval_seconds`）
- 若状态文件损坏，会自动备份为 `ui-tui-state.json.corrupt.<timestamp>` 并重建
- 默认启动会自动执行一次 doctor（可通过 `[ui-tui] auto_doctor=false` 或 `--no-doctor` 关闭）

审计日志：

- 关键操作（`approve`/`deny`/`cancel`/`config set`）会记录到 `~/.agent-daemon/ui-tui-audit.log`

常用别名：

- `:q` / `quit` -> `/quit`
- `ls` -> `/tools`
- `show ...` -> `/show ...`
- `gw` / `gw ...` -> `/gateway status` / `/gateway ...`
- `cfg` / `cfg ...` -> `/config get` / `/config ...`

烟测：

```bash
./ui-tui/e2e_smoke.sh
```

CI 中可通过 `ARTIFACTS_DIR` 导出诊断样本：

```bash
ARTIFACTS_DIR=./artifacts ./ui-tui/e2e_smoke.sh
```

发布单文件：

```bash
./ui-tui/release.sh v1.0.0
```
