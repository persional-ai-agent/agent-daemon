# 102 Summary - Yuanbao 贴纸搜索工具实现（yb_search_sticker）

## 变更

- `yb_search_sticker`：基于内置贴纸目录（当前为子集）实现关键词搜索，返回 `sticker_id/name/description/package_id`
- `yb_send_dm` / `yb_send_sticker` / `yb_query_group_info` / `yb_query_group_members`：已接入 Go 版 Yuanbao gateway adapter（WebSocket + 手写 protobuf 编解码，最小可用）

实现位置：

- `internal/tools/yuanbao_stickers.go`
- `internal/tools/yuanbao_tools.go`
- `internal/tools/builtin.go`：更新注册与 schema
- `internal/gateway/platforms/yuanbao.go`：Yuanbao 平台适配器（当前偏 outbound/tool 驱动）
- `internal/yuanbao/*`：sign-token + WS protobuf 编解码 + 最小 WS client
