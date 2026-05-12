# 172 总结：Gateway Yuanbao 审批快捷回复最小闭环

## 1. 背景

在 Telegram、Discord、Slack 都已有最小原生审批交互后，Yuanbao 侧仍只有文本 slash 命令：

- `/pending`
- `/approve`
- `/deny`

但 Yuanbao 适配器当前没有稳定的原生按钮/命令菜单 API 可复用，因此不能照搬 Telegram/Discord/Slack 的按钮流。

本轮目标不是伪造按钮，而是补一个更符合 Yuanbao 现状的最小可用闭环：审批快捷回复。

## 2. 本轮实现

### 2.1 Yuanbao 文本快捷回复归一

在 `internal/gateway/runner.go` 新增 `normalizeGatewayCommand()`：

- 仅对 `yuanbao` 平台生效
- 将常见中文快捷回复归一到现有 slash command：
  - `批准` / `同意` / `通过` -> `/approve`
  - `拒绝` / `驳回` -> `/deny`
  - `状态` -> `/status`
  - `待审批` -> `/pending`
  - `审批` -> `/approvals`
  - `帮助` -> `/help`

这样可以继续复用原有命令内核和审批工具，不额外引入平台分支状态机。

### 2.2 待审批提示补充快捷回复文案

当工具返回 `pending_approval` 时，如果当前平台是 Yuanbao：

- 在原有 `Pending approval: /approve <id> or /deny <id>` 基础上
- 追加 `Quick reply: 批准 / 拒绝`

### 2.3 `/pending` 与 `/help` 增强

在 Yuanbao 平台：

- `/pending` 输出会额外提示 `quick_reply: 批准 / 拒绝`
- `/help` 会列出中文快捷回复别名：
  - `状态`
  - `待审批`
  - `审批`
  - `批准`
  - `拒绝`
  - `帮助`

## 3. 结果

Yuanbao 现在具备最小审批快捷回复闭环：

- 用户仍可继续使用原始 slash 命令
- 也可直接回复中文短语完成常用审批动作

这让 Yuanbao 从“只有机械 slash 命令”收敛为“具备最小自然语言快捷交互”，适配了当前平台能力边界。

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

- Yuanbao 原生审批按钮 / 命令菜单
- 更完整原生 slash UI
- 更完整 token lock / 分布式协调
- 更多平台交互组件
