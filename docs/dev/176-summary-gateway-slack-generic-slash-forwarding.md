# 176 总结：Gateway Slack 通用 slash 命令转发与即时确认

## 1. 背景

上一轮已经补了 Slack slash 命令入口，但还存在两个明显问题：

1. 若 Slack app 配置的是通用入口（例如 `/agent`），用户输入 `status` 时会被错误拼成 `/agent status`，无法命中现有 Gateway 命令内核。
2. slash command 只做了底层 `Ack`，用户界面没有即时反馈。

因此本轮继续把 Slack slash 入口从“能接住事件”提升到“能稳定转发通用命令”。

## 2. 本轮实现

### 2.1 通用 slash 入口转发

在 `internal/gateway/platforms/slack.go`：

- `renderSlackSlashCommand()` 现在会区分两类入口：
  - **内置命令名直接配置**：如 `/status`、`/pending`
  - **通用代理入口**：如 `/agent`

规则如下：

- 如果 `Text` 本身已经是 slash 命令，直接透传
- 如果 command 名属于 Gateway 内置命令，则保留 `/<cmd> <text>` 形式
- 否则把 text 规范化为 `/<text>`，例如：
  - `/agent status` -> `/status`
  - `/agent grant 300` -> `/grant 300`

这样 Slack app 既可以逐个配置命令，也可以只配置一个通用入口。

### 2.2 即时确认

收到 Slack slash command 后，Socket Mode `Ack` 现在会回一个轻量 ephemeral payload：

- `Accepted. Check the next bot reply.`

这样用户在 Slack 客户端里会立刻看到命令已被接收，而不是只有静默 ACK。

## 3. 结果

Slack 侧现在的原生命令入口更接近真实可用：

- 支持直接 `/status`
- 也支持 `/agent status` 这类通用代理式入口
- 同时具备即时确认反馈

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

- Slack slash command 的自动注册 / app manifest 管理
- 更完整原生 modal / form 流
- Yuanbao 原生命令菜单
- 更完整 token lock / 分布式协调
