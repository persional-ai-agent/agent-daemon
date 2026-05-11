# 102 Summary - Yuanbao 贴纸搜索工具实现（yb_search_sticker）

## 变更

- `yb_search_sticker`：基于内置贴纸目录（当前为子集）实现关键词搜索，返回 `sticker_id/name/description/package_id`
- `yb_send_dm` / `yb_send_sticker` / `yb_query_group_info` / `yb_query_group_members`：仍依赖 Yuanbao gateway adapter（Go 版暂未实现）

实现位置：

- `internal/tools/yuanbao_stickers.go`
- `internal/tools/yuanbao_tools.go`
- `internal/tools/builtin.go`：更新注册与 schema

