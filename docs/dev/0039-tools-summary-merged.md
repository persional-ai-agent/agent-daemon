# 0039 tools summary merged

## 模块

- `tools`

## 类型

- `summary`

## 合并来源

- `0018-file-summary-merged.md`
- `0023-homeassistant-summary-merged.md`
- `0034-process-summary-merged.md`
- `0049-tool-summary-merged.md`

## 合并内容

### 来源：`0018-file-summary-merged.md`

# 0018 file summary merged

## 模块

- `file`

## 类型

- `summary`

## 合并来源

- `0080-file-tools-reject-non-regular.md`
- `0081-file-tools-reject-symlink-escape.md`
- `0083-file-staleness-warning.md`

## 合并内容

### 来源：`0080-file-tools-reject-non-regular.md`

# 081 Summary - 文件工具拒绝非普通文件（FIFO/Socket/Device）

## 背景

Hermes 的文件相关工具对“非普通文件”（如命名管道/FIFO、socket、设备文件等）通常会做防护，以避免：

- 读取 FIFO 导致阻塞（等待写端）；
- 误读/误写设备文件带来安全风险；
- 工具行为不可预期（例如读 socket、读特殊设备）。

Go 版 `agent-daemon` 之前仅做了 “路径必须在 workdir 内” 的约束，但未显式拒绝特殊文件类型。

## 变更

- `read_file`：在打开文件前拒绝非普通文件（FIFO/socket/device/目录）。
- `patch`：在读取与写入前拒绝非普通文件（避免 `os.ReadFile` 在 FIFO 上阻塞）。
- `write_file`：如果目标路径已存在，则拒绝非普通文件（不存在时正常创建）。

实现位置：

- `internal/tools/safety.go`：新增 `rejectNonRegularFile(path)`。
- `internal/tools/builtin.go`：在上述工具中接入校验逻辑。

## 验证

- 新增用例：`internal/tools/read_file_guardrails_test.go` 覆盖 FIFO 场景，确保 `read_file` 立即返回错误而不是阻塞。

## 边界与后续

- 本次仅对“非普通文件类型”做拒绝；未处理“符号链接指向 workdir 外”的额外隔离（如 Hermes 进一步要求，可继续对齐）。

### 来源：`0081-file-tools-reject-symlink-escape.md`

# 082 Summary - 文件工具拒绝 symlink 逃逸 workdir

## 背景

仅依赖“路径字符串是否在 workdir 内”无法阻止通过符号链接（symlink）实现的路径逃逸，例如：

- 在 workdir 内创建 `out -> /tmp/outside` 的目录 symlink；
- 再调用 `write_file path=out/a.txt`，实际会写入 workdir 外部路径。

Hermes 的文件工具通常会对 symlink/realpath 做更严格的约束，避免该类逃逸。

## 变更

- 增加 `rejectSymlinkEscape(workdir, resolvedPath)`：
  - 从 workdir 开始逐级 `lstat` 检查已存在的路径组件；
  - 发现任一组件为 symlink 即拒绝，防止读/写/补丁操作穿透到 workdir 外。
- `read_file` / `patch` / `write_file` 在执行前接入该校验：
  - `write_file` 在 `MkdirAll` 前校验，避免通过目录 symlink 在 workdir 外创建文件。

实现位置：

- `internal/tools/safety.go`：新增 `rejectSymlinkEscape`
- `internal/tools/builtin.go`：接入 `read_file` / `patch` / `write_file`

## 验证

- `internal/tools/read_file_guardrails_test.go`：新增用例，`read_file` 读取指向 workdir 外的 symlink 文件应拒绝。
- `internal/tools/builtin_test.go`：新增用例，`write_file` 写入 symlink 目录路径应拒绝，且不会写到 workdir 外。

## 边界与后续

- 当前策略是“路径组件中出现 symlink 即拒绝”，更严格但更安全；如需要支持 workdir 内部 symlink（且仍不逃逸），可后续演进为 realpath 归一化 + 根前缀校验策略。

### 来源：`0083-file-staleness-warning.md`

# 084 Summary - write_file/patch 增加“文件已变更”告警（staleness warning）

## 背景

Hermes 的文件工具会在 `read_file` 后记录文件 mtime，并在后续 `write_file`/`patch` 时检查该文件是否在“上次读取之后被外部修改”，以提示模型：

- 你基于的内容可能已经过期；
- 需要考虑先重新读取确认再写入，避免覆盖他人修改或并发 agent 的写入。

Go 版 `agent-daemon` 之前缺少该类告警。

## 变更

- `read_file`：在同一 `session_id` 内记录该文件的 mtime（`ModTime().UnixNano()`）。
- `write_file` / `patch`：
  - 如果目标文件存在且记录过 read mtime，并且当前 mtime 与记录不一致，则在返回中加入 `_warning` 字段（不阻塞写入）。
  - 写入成功后刷新记录的 mtime，避免同一会话的连续写入误报。

实现位置：

- `internal/tools/builtin.go`：新增 `readStamp`（上限 1000，超限清空），并在 `read_file`/`write_file`/`patch` 接入。

## 验证

- `internal/tools/builtin_test.go`：新增 `write_file` staleness warning 用例。
- `internal/tools/patch_test.go`：新增 `patch` staleness warning 用例。

### 来源：`0023-homeassistant-summary-merged.md`

# 0023 homeassistant summary merged

## 模块

- `homeassistant`

## 类型

- `summary`

## 合并来源

- `0088-homeassistant-kanban-tools.md`

## 合并内容

### 来源：`0088-homeassistant-kanban-tools.md`

# 089 Summary - Home Assistant 与 Kanban core tools 实现

## 背景

Hermes 的 `_HERMES_CORE_TOOLS` 中包含：

- Home Assistant：`ha_list_entities` / `ha_get_state` / `ha_list_services` / `ha_call_service`
- Kanban：`kanban_show/complete/block/heartbeat/comment/create/link`

Go 版 `agent-daemon` 之前缺失上述工具。

## 变更

### Home Assistant

新增 `ha_*` 工具，基于 Home Assistant REST API：

- `HASS_URL`（或 `HOME_ASSISTANT_URL`）与 `HASS_TOKEN` 未配置时，返回 `success=false` 且 `available=false`
- API：
  - GET `/api/states`
  - GET `/api/states/{entity_id}`
  - GET `/api/services`
  - POST `/api/services/{domain}/{service}`

实现位置：

- `internal/tools/homeassistant.go`
- `internal/tools/builtin.go`（注册 + schema）

### Kanban

新增 `kanban_*` 工具，使用 workdir 内本地持久化文件作为最小实现：

- 存储：`{workdir}/.agent-daemon/kanban.json`
- 支持：
  - `kanban_show` 查看 board
  - `kanban_create` 创建 task（可自定义 id）
  - `kanban_comment` 写入备注
  - `kanban_complete` / `kanban_block` 更新状态
  - `kanban_heartbeat` 记录 worker 心跳
  - `kanban_link` 建立任务链接关系

实现位置：

- `internal/tools/kanban.go`
- `internal/tools/builtin.go`（注册 + schema）

### Toolsets

- 新增 toolset：`homeassistant` / `kanban`
- `core` toolset include 增加 `homeassistant` / `kanban`

实现位置：

- `internal/tools/toolsets.go`

### 来源：`0034-process-summary-merged.md`

# 0034 process summary merged

## 模块

- `process`

## 类型

- `summary`

## 合并来源

- `0075-process-list-action.md`
- `0104-process-tool-actions.md`
- `0107-process-write-and-terminate.md`

## 合并内容

### 来源：`0075-process-list-action.md`

# 076 总结：process 工具补齐 list 动作（对齐 Hermes 体验）

## 变更

`process` 工具新增 `action=list`，用于列出当前进程内跟踪的后台任务（terminal background）。

返回字段包含：

- `session_id`、`command`、`started_at`、`status`、`exit_code`、`output_file`、`error`

## 边界

只覆盖 agent-daemon 自己启动并跟踪的后台进程，不做系统级进程枚举。

### 来源：`0104-process-tool-actions.md`

# 105 Summary - process 工具补齐 poll/log/wait/kill/write 动作（Hermes 体验对齐）

## 背景

Hermes 的 `process` 工具支持对后台进程进行更丰富的管理（poll/log/wait/kill/write 等）。Go 版此前仅支持 `list/status/stop`，导致模型在长任务场景下无法按 Hermes 习惯读取增量日志或等待完成。

## 变更

- `process` 新增动作：
  - `poll`：返回自上次 poll 以来的新输出（按 session 记忆 offset）
  - `log`：按 byte offset 分页读取输出文件
  - `wait`：阻塞等待进程结束或超时，并返回一次最终 poll
  - `kill`：`stop` 别名
  - `write`：返回 not supported（Go 版未跟踪 stdin/pty）
- `process` schema 更新，暴露 `offset/max_chars/timeout_seconds`

## 修改文件

- `internal/tools/builtin.go`

### 来源：`0107-process-write-and-terminate.md`

# 108 Summary - process write/terminate 行为补齐（Hermes 体验对齐）

## 变更

- `process(action="write")`：支持向后台进程 stdin 写入内容（best-effort；非 PTY）。
- `process(action="stop")`：从硬 kill 调整为优先 `SIGTERM`，超时后 `SIGKILL`（保留 `kill` 显式硬 kill）。

说明：

- `stop_process`（独立工具）仍保持历史行为（硬 kill），避免破坏既有调用习惯；推荐新代码使用 `process(action="stop")`。

## 修改文件

- `internal/tools/process.go`
- `internal/tools/builtin.go`

### 来源：`0049-tool-summary-merged.md`

# 0049 tool summary merged

## 模块

- `tool`

## 类型

- `summary`

## 合并来源

- `0058-tool-disable-config.md`
- `0069-tool-result-success-field.md`
- `0072-tool-success-web-session-search.md`
- `0073-tool-success-todo-memory.md`
- `0074-tool-success-terminal-approval-skills.md`
- `0253-tool-capability-closure-media-backends.md`

## 合并内容

### 来源：`0058-tool-disable-config.md`

# 059 总结：工具禁用配置最小对齐

## 变更摘要

新增工具禁用配置，补齐 Hermes 工具管理中的最小启停能力。

## 新增能力

```bash
agentd tools disable terminal
agentd tools disabled
agentd tools enable terminal
```

配置来源：

- `AGENT_DISABLED_TOOLS=terminal,web_fetch`
- `[tools] disabled = terminal,web_fetch`

禁用工具会从 registry 中移除，因此不会出现在 `tools list` / `tools schemas` 中，dispatch 时也会成为 unknown tool。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/config/config.go` | 新增 `DisabledTools` 配置 |
| `internal/config/config_test.go` | 增加 disabled tools 配置读取测试 |
| `internal/tools/registry.go` | 新增 `Disable` |
| `cmd/agentd/main.go` | 增加 `tools disabled|disable|enable` 与运行时过滤 |
| `cmd/agentd/main_test.go` | 增加列表解析和 registry 过滤测试 |
| `README.md` | 增加工具启停示例 |
| `docs/overview-product.md` | 更新 CLI 工具管理说明 |
| `docs/overview-product-dev.md` | 更新工具禁用设计说明 |
| `docs/dev/README.md` | 增加 059 文档索引 |

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config ./cmd/agentd`：通过
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`：通过
- 手动验证：临时配置文件执行 `tools disable -file ... terminal` 后，`tools disabled -file ...` 输出 `terminal`
- 手动验证：使用 `AGENT_CONFIG_FILE=... tools list` 时，`terminal` 从工具列表消失，`read_file` 仍保留
- 手动验证：`tools enable -file ... terminal` 后禁用列表清空

### 来源：`0069-tool-result-success-field.md`

# 070 总结：补齐工具结果 `success` 字段（兼容性增强）

## 背景

Hermes 多数工具返回 JSON 时会包含 `success` / `error` 等标准字段，便于上层统一处理。

## 本次变更

- `read_file` / `write_file` / `search_files` 的返回结果补齐 `success=true`（不移除原有字段）。

## 边界

- 其他工具仍可能返回不同风格字段；后续可按需逐步统一，不做一次性重构。

### 来源：`0072-tool-success-web-session-search.md`

# 073 总结：web 与 session_search 工具返回补齐 success

为对齐 Hermes 常见返回格式并便于上层统一处理，本次为以下工具返回增加 `success=true`：

- `session_search`
- `web_fetch`
- `web_search`
- `web_extract`

不移除原字段，保持兼容。

### 来源：`0073-tool-success-todo-memory.md`

# 074 总结：todo/memory 工具返回补齐 success

为对齐 Hermes 常见返回格式并便于上层统一处理，本次为以下工具返回增加 `success=true`：

- `todo`
- `memory`

不移除原字段，保持兼容。

### 来源：`0074-tool-success-terminal-approval-skills.md`

# 075 总结：terminal/approval/skills 工具返回补齐 success

为对齐 Hermes 常见返回格式并便于上层统一处理，本次为以下工具返回增加 `success` 字段：

- `terminal`（前台：`success = (error==nil && exit_code==0)`；后台：`success=true`；pending approval：`success=false`）
- `process_status` / `stop_process`（兼容工具）
- `approval`
- `skill_list`/`skills_list`、`skill_view`、`skill_search`

不移除原字段，保持兼容。

### 来源：`0253-tool-capability-closure-media-backends.md`

# 工具能力收口（第 4 项）：media/best-effort 实后端完善

本轮针对“工具能力级差距”做了完整收口，重点完善 `vision_analyze / image_generate / text_to_speech` 的实后端兼容与回退链路。

## 主要改动

- `internal/tools/media_tools.go`
  - `vision_analyze`：
    - OpenAI 响应结构异常时不再直接失败，改为回退到元数据模式，保证工具可用性。
  - `image_generate`：
    - OpenAI 图片接口在非 `b64_json` 返回时，新增 URL 下载兜底（`tryWriteImageFromURL`）。
    - 仍保留本地 placeholder 回退，形成“实后端优先 + 可降级”链路。
  - `text_to_speech`：
    - 新增参数级覆盖：`model`、`voice`（优先于环境变量）。
    - 继续保留 OpenAI 失败后的 WAV 占位回退。
- `internal/tools/builtin.go`
  - `text_to_speech` schema 补齐 `model` 与 `voice` 字段。

## 测试

- 新增 `internal/tools/media_tools_test.go`
  - `tryWriteImageFromURL` 下载写盘测试
  - `text_to_speech` schema 字段覆盖测试（`model/voice`）

## 验证

- `go test ./internal/tools ./internal/api ./cmd/agentd`
- `make contract-check`
- `go test ./...`
