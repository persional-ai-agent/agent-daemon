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

