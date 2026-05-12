# 161-summary-gateway-single-instance-lock

## 背景

虽然前几轮已经补齐了 `gateway run/start/stop/restart/install/uninstall`，但 Gateway 仍缺少最小的多实例保护。同一个 `workdir` 下重复启动多个 gateway 进程，会导致同一平台消息被重复消费，因此本轮先补一个最小单实例锁，收口 token lock 的核心语义。

## 本次实现

- 新增 `gateway.lock` 文件，位置为 `<workdir>/.agent-daemon/gateway.lock`
- `agentd gateway run` 启动前会尝试获取非阻塞独占锁
- `agentd serve` 在启用 gateway 且存在可用 adapter 时，也会尝试获取同一把锁
- 如果锁已被其他进程持有：
  - `gateway run` 直接失败
  - `serve` 仅跳过 gateway 启动，不影响 HTTP 服务本身
- `agentd gateway status` 新增：
  - `locked`
  - `lock_pid`
  - `lock_path`
- `agentd gateway start` 在 fork 前会优先检查现有锁，避免无意义拉起子进程

## 设计取舍

### 1. 先补“同 workdir 单实例”

本轮不是完整分布式 token lock，只覆盖本机同一工作目录下的最小互斥。这样已经能消除最常见的重复消费问题，同时不需要引入外部存储或平台级协调。

### 2. `serve` 与 `gateway run` 共用一把锁

Hermes 语义上 Gateway 是同一类消费端，不应因为运行在 `serve` 内嵌模式还是独立 `gateway run` 模式而并发消费同一工作区消息。因此本轮将两条启动路径统一到同一锁文件。

## 验证

- `go test ./...`
- `go run ./cmd/agentd gateway status -json`
- 在当前环境中验证 `status` 输出新增 `locked/lock_pid/lock_path`

说明：由于烟测环境没有持续在线的真实平台凭证，本轮没有做双进程真实竞争测试；以锁路径写入、状态暴露、启动前检查和编译验证为主。

## 文档更新

- README 增加“Gateway 同一 workdir 单实例锁”说明
- 产品/开发总览从“完全缺 token lock”更新为“已具备最小单实例锁，仍缺更完整 token lock”

## 剩余差距

Gateway 主线剩余缺口继续收敛为：

- 原生平台 slash UI
- 审批按钮流
- 更完整 token lock / 跨实例协调
- 更多平台适配器
