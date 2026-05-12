# 162-summary-gateway-token-lock

## 背景

上一轮已经补了同 `workdir` 单实例锁，但这还不能阻止“两个不同工作区使用同一组平台凭证同时消费消息”。Hermes 的 `token lock` 本质是避免同一 bot/token 被多个进程并发消费，因此本轮继续补一个最小跨工作区凭证锁。

## 本次实现

- 新增基于平台凭证指纹的全局 `token lock`
- 锁文件路径：`$TMPDIR/agent-daemon-gateway-locks/<fingerprint>.lock`
- 指纹来源：
  - `telegram bot token`
  - `discord bot token`
  - `slack bot/app token`
  - `yuanbao token/app_id`
- `agentd gateway run` 启动时会同时获取：
  - `<workdir>/.agent-daemon/gateway.lock`
  - 凭证指纹 `token lock`
- `agentd serve` 内嵌 gateway 启动时也会走同样的双锁逻辑
- `agentd gateway start` 在 fork 之前会预检查 `token lock`
- `agentd gateway status` 新增：
  - `token_locked`
  - `token_lock_pid`
  - `token_lock_path`

## 设计取舍

### 1. 先做本机级跨工作区协调

本轮仍然不做分布式锁，也不接第三方存储。范围限定为“同一台机器上，不同工作区不能同时用同一套平台凭证消费”。

### 2. 用凭证指纹而不是 workdir

单实例锁解决的是“同一工作区重复启动”；`token lock` 解决的是“不同工作区复用同一平台身份”。两者语义不同，因此保留两把锁并行存在。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway status -json`
- 验证输出新增 `token_locked/token_lock_pid/token_lock_path`

说明：当前环境没有真实在线平台凭证，本轮以锁路径计算、状态暴露、启动前预检查与编译验证为主，没有做真实双工作区竞争烟测。

## 文档更新

- README 说明从“同 workdir 单实例锁”更新为“同 workdir 锁 + 跨工作区 token lock”
- 产品/开发总览从“仍缺 token lock”收口为“已具备最小 token lock，仍缺更完整策略”

## 剩余差距

Gateway 主线剩余高价值缺口继续集中在：

- 原生平台 slash UI
- 审批按钮流
- 更完整 token lock 策略 / 分布式协调
- 更多平台适配器
