# 177 总结：CLI 新增 Slack gateway manifest 导出命令

## 1. 背景

Slack 侧此前已经具备：

- 原生审批按钮
- slash 命令入口
- 通用 `/agent <cmd>` 转发

但仍有一个落地缺口：Slack slash command 与 app scopes 需要在 Slack app 后台手工配置，项目本身没有给出结构化导出结果，部署时容易漏项。

## 2. 本轮实现

### 2.1 新增 `agentd gateway manifest`

在 `cmd/agentd/main.go` 新增：

- `agentd gateway manifest -platform slack`
- `agentd gateway manifest -platform slack -command /agent`
- `agentd gateway manifest -platform slack -json`

当前仅支持 `slack` 平台。

### 2.2 导出内容

输出包含：

- `commands`：推荐 slash command 列表
- `app_manifest`：Slack app manifest 片段
- `next_actions`：后续配置提示
- `command_routes`：通用入口示例映射

manifest 中包含最小所需：

- `slash_commands`
- `commands` scope
- `chat:write`
- 历史消息读取 scope
- `socket_mode_enabled`
- `interactivity.is_enabled`

### 2.3 命令前缀可配置

通过 `-command /agent` 可指定通用 slash 入口前缀，导出的命令示例和路由映射会随之变化。

## 3. 结果

Slack Gateway 现在不只是“支持 slash command”，还具备最小安装配置导出能力，便于把代码能力落到真实 Slack app 配置中。

## 4. 验证

- `go test ./...`
- `go run ./cmd/agentd gateway manifest -platform slack -command /agent -json`

## 5. 文档同步

已更新：

- `README.md`
- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`

## 6. 仍未覆盖

本轮没有补：

- Slack app 的自动注册 / 自动应用 manifest
- 更完整原生 modal / form 流
- Yuanbao 原生命令菜单
- 更完整 token lock / 分布式协调
