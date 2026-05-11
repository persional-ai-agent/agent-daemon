# 100 Summary - Spotify 工具实现（Web API）

## 变更

将 Spotify 相关工具从占位升级为可用实现（需要 `SPOTIFY_ACCESS_TOKEN`）：

- `spotify_search`：`/v1/search`
- `spotify_devices`：`/v1/me/player/devices`
- `spotify_playback`：`action=get|play|pause`（`/v1/me/player`、`/play`、`/pause`）
- `spotify_queue`：`action=get|add`（`/v1/me/player/queue`）
- `spotify_playlists`：`/v1/me/playlists`
- `spotify_albums`：`/v1/me/albums`
- `spotify_library`：`/v1/me/tracks`

实现位置：

- `internal/tools/spotify.go`
- `internal/tools/builtin.go`：注册与 schema 更新

