# 257 总结：Gateway 命令矩阵与僵尸锁收口

本次围绕 Gateway “平台命令一致性 + 锁可靠性”做了一次性收口。

## 改动

- `cmd/agentd/main.go`
  - `gateway status` 增加锁状态字段：
    - `stale_lock`
    - `stale_token_lock`
  - `gateway start` 启动前自动清理僵尸锁（PID 不存活的 runtime/token lock）。
  - Slack manifest 命令补齐：
    - `/pair` `/unpair` `/cancel` `/queue` `/help`
    - 形成与其它平台一致的核心命令集合。

- 新增锁状态辅助逻辑：
  - `readGatewayLockState`
  - `cleanupStaleGatewayLock`

## 测试

- `cmd/agentd/main_test.go`
  - 核心命令跨平台一致性测试（Telegram/Discord/Slack/Yuanbao）。
  - 僵尸锁识别与清理测试。

验证：

- `go test ./cmd/agentd -count=1`
- `go test ./...`
