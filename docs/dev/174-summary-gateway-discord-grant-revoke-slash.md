# 174 总结：Gateway Discord `grant` / `revoke` 原生 slash 参数面

## 1. 背景

上一轮已经补了 Discord 原生 slash 命令最小闭环，但当时仍缺一块明显空白：

- `grant`
- `revoke`

这两个命令在文本模式下已经完整可用，但 Discord 原生 slash UI 只覆盖了状态、审批查看和 approve/deny，授权管理仍要退回手工输入。

## 2. 本轮实现

### 2.1 注册 `grant` / `revoke` 原生命令

在 `internal/gateway/platforms/discord.go` 的命令注册表中新增：

- `grant`
  - `pattern`：可选字符串
  - `ttl`：可选整数
- `revoke`
  - `pattern`：可选字符串

这样 Discord 用户可以通过原生命令面板直接选择授权管理动作，而不必记忆文本格式。

### 2.2 渲染回既有文本命令

新增 `renderDiscordGrantRevoke()`，把 slash 参数翻译回现有命令串：

- `grant(ttl=300)` -> `/grant 300`
- `grant(pattern=delete, ttl=300)` -> `/grant pattern delete 300`
- `revoke(pattern=delete)` -> `/revoke pattern delete`
- `revoke()` -> `/revoke`

这样继续复用：

- `parseApprovalManageCommand()`
- `grantApproval()`
- `revokeApproval()`

不增加新的审批协议或分叉逻辑。

## 3. 结果

Discord 原生 slash command 现在不再只覆盖“查询/单次审批”，也覆盖了“授权管理”。

这让 Discord 侧的原生命令闭环更接近文本命令能力面。

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

- Slack / Yuanbao 原生命令菜单
- 更完整原生表单 / modal 流
- 更完整 token lock / 分布式协调
