# ui-tui 体验增强总结（分页快捷翻页）

## 本阶段完成

1. 在 Go 版 `ui-tui` 中新增会话分页快捷命令：
   - `/next`
   - `/prev`
2. 基于最近一次 `/show [sid] [offset] [limit]` 记住会话与分页上下文，支持连续翻页。

## 结果

会话浏览从“每次手输 offset/limit”升级为“show 一次后快捷翻页”，交互效率提升。

## 验证

- `go test ./...` 通过。
