# 0017 integration summary merged

## 模块

- `integration`

## 类型

- `summary`

## 合并来源

- `0025-integration-summary-merged.md`

## 合并内容

### 来源：`0025-integration-summary-merged.md`

# 0025 integration summary merged

## 模块

- `integration`

## 类型

- `summary`

## 合并来源

- `0097-integration-tool-placeholders.md`

## 合并内容

### 来源：`0097-integration-tool-placeholders.md`

# 098 Summary - 补齐 Hermes 额外集成工具名（占位实现 + 凭证 gate）

## 背景

Hermes 的 TOOLSETS 中包含若干第三方集成工具（Discord admin、飞书文档/云盘、Spotify、Yuanbao、RL training 等）。Go 版 `agent-daemon` 之前缺失这些工具名，导致 toolset 解析与“工具名对齐”不完整。

## 变更

新增以下工具的“占位实现”：

- Discord：admin：`discord_admin`（需要 `DISCORD_BOT_TOKEN`）
- Feishu/Lark：`feishu_doc_read`、`feishu_drive_list_comments`、`feishu_drive_list_comment_replies`、`feishu_drive_add_comment`、`feishu_drive_reply_comment`（需要 `FEISHU_APP_ID/FEISHU_APP_SECRET`）
- Spotify：`spotify_search`、`spotify_devices`、`spotify_playback`、`spotify_queue`、`spotify_playlists`、`spotify_albums`、`spotify_library`（需要 `SPOTIFY_ACCESS_TOKEN`）
- Yuanbao：`yb_send_dm`、`yb_send_sticker`、`yb_search_sticker`、`yb_query_group_info`、`yb_query_group_members`（需要 `YUANBAO_TOKEN`）
- RL：`rl_*`（占位）

行为：

- 未配置凭证时：返回 `success=false` 且 `available=false`，并提示缺失的 env
- 配置凭证后：当前仍返回 `not implemented`（后续可逐个实现真实 API）

实现位置：

- `internal/tools/integration_tools.go`
- `internal/tools/builtin.go`：注册 + schema
- `internal/tools/toolsets.go`：新增 toolset `discord_admin/feishu/spotify/rl/yuanbao`
