# 168 总结：Gateway Telegram 原生审批按钮最小闭环

## 1. 背景

文本审批闭环已经具备：

- `/pending`
- `/approvals`
- `/grant` / `/revoke`
- `/approve` / `/deny`

但对齐 Hermes 时，Gateway 仍缺“原生平台审批按钮流”。本轮先补一个最小可用版本：仅对 Telegram 增加原生 Approve / Deny 按钮，不扩展到其他平台，也不引入新的审批状态机。

## 2. 本轮实现

### 2.1 可携带 metadata 的文本发送扩展

- 在 `internal/platform/adapter.go` 增加可选接口 `RichTextSender`
- `internal/gateway/runner.go` 的 `sendText()` 会优先调用该扩展接口
- 非扩展平台保持原有 `Send()` 行为，不受影响

这样可以只给支持原生控件的平台附加按钮，而不改动统一 Gateway 主流程。

### 2.2 Telegram 按钮发送

在 `internal/gateway/platforms/telegram.go`：

- 新增 `SendText(..., meta)` 实现
- 当 metadata 中包含 `approval_id` 时，自动附加 Telegram inline keyboard：
  - `Approve`
  - `Deny`

对应 callback data 直接复用现有文本命令：

- `/approve <id>`
- `/deny <id>`

因此无需新增审批协议，也无需修改 `approval` 工具。

### 2.3 Telegram callback 回流到统一命令处理

Telegram update loop 现在会消费 `CallbackQuery`：

- 提取 `callback.Data`
- 转成统一 `gateway.MessageEvent`
- 继续走现有 slash command 分支

这意味着按钮点击最终仍落到：

- `confirmApproval()`
- 既有授权校验
- 既有审批工具调用链

实现上是“原生 UI + 统一命令内核”，复杂度最低。

### 2.4 `/pending` 响应补充 `approval_id` metadata

`/pending` 返回最近待审批项时，如果存在 `approval_id`，也会附带 metadata，这样 Telegram 侧不仅能看到详情，还能直接点按钮处理。

## 3. 结果

Gateway 审批链路现在分两层：

- 跨平台保底：文本命令 `/pending`、`/approve`、`/deny`
- Telegram 增强：原生 Approve / Deny inline buttons

这样可以把“原生审批按钮流”从完全缺失收敛为“Telegram 最小落地，其他平台后补”。

## 4. 验证

- `go test ./...`

## 5. 文档同步

已更新：

- `README.md`
- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`

## 6. 仍未覆盖

本轮没有补：

- Discord / Slack / Yuanbao 原生审批按钮
- 原生 slash UI
- 更完整 token lock / 分布式协调
- Gateway richer delivery / 更多平台
