# 087 Summary - read_file 连续重复读取告警与阻断（防循环）

## 背景

Hermes 的 `read_file` 除了 dedup stub 外，还会对“连续重复读取同一文件区域”进行告警与阻断，避免模型陷入无效循环（反复读取同一段内容、浪费上下文与时间）。

## 变更

- 当同一 `session_id` 下重复读取同一路径 + 相同 `offset/limit`，且文件内容未变化（dedup 命中）：
  - 第 3 次：在 dedup stub 中增加 `_warning`
  - 第 4 次及以上：返回 `success=false` + `error`（包含 `BLOCKED`），强制打断循环
- 当 `read_file` 返回真实内容（文件变化或未走 dedup）后，循环计数对该 key 重置。

实现位置：

- `internal/tools/builtin.go`：新增 per-session 的连续读取 tracker（`readLoop`）。

## 验证

- `internal/tools/read_file_guardrails_test.go`：新增用例覆盖第 3 次告警与第 4 次阻断。

