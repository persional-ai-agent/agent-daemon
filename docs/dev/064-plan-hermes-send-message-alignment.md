# 064 计划：Hermes send_message 最小对齐

## 目标（可验证）

- `send_message(action='list')` 返回当前进程已连接的 gateway adapters。
- `send_message(action='send', platform, chat_id, message)` 可投递文本消息。
- Gateway runner 会在 adapter connect/disconnect 时注册/注销 adapter。
- 文档对齐矩阵更新：Gateway/toolsets 标记调整，补 docs/dev 索引。

## 实施步骤

1. 解耦 adapter 接口到 `internal/platform`。
2. 新增运行时 adapter registry。
3. Gateway runner hook：connect 后 register，退出前 unregister。
4. 新增工具 `send_message` 并注册到 engine。
5. 更新 toolsets `messaging` + `core` includes。
6. 补单测与文档。

