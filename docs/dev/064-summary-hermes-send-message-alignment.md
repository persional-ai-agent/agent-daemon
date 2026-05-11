# 064 总结：Hermes send_message 最小对齐

## 完成情况

- 新增 `internal/platform`：adapter 接口 + 运行时 registry。
- Gateway runner 在 adapter connect/disconnect 时 register/unregister。
- 新增工具 `send_message`：
  - `list`：返回已连接平台列表
  - `send`：按 `platform + chat_id`（或 `target=platform:chat_id`）投递文本
- toolsets 增加 `messaging`，并纳入 `core` includes。

## 边界

- 无 channel directory / 目标名称解析。
- 无媒体附件路由、线程/话题、重试与错误脱敏对齐。
