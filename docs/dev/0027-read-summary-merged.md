# 0027 read summary merged

## 模块

- `read`

## 类型

- `summary`

## 合并来源

- `0036-read-summary-merged.md`

## 合并内容

### 来源：`0036-read-summary-merged.md`

# 0036 read summary merged

## 模块

- `read`

## 类型

- `summary`

## 合并来源

- `0071-read-file-max-chars.md`
- `0082-read-file-dedup.md`
- `0085-read-file-max-chars-reject.md`
- `0086-read-file-loop-guard.md`

## 合并内容

### 来源：`0071-read-file-max-chars.md`

# 072 总结：read_file 增加 max_chars 防护（Hermes 风格）

## 变更

`read_file` 新增：

- `max_chars`（默认 100000，上限 200000）
- `reject_on_truncate`（默认 true；超限时返回 error 而非截断内容）
- `truncated` 标记（仅在 `reject_on_truncate=false` 的兼容模式下返回截断内容时使用）

用于防止一次性读取超大文件导致上下文爆炸，并与 Hermes 的 read-size guard 思路对齐。

### 来源：`0082-read-file-dedup.md`

# 083 Summary - read_file 增加去重返回（未变更文件返回 stub）

## 背景

Hermes 的 `read_file` 为避免模型陷入“重复读取同一文件”循环，会对同一会话内重复读取且文件未变化的情况返回轻量 stub（不再重复发送文件内容），提示模型直接参考之前的读结果。

Go 版 `agent-daemon` 之前每次都会返回完整内容，容易造成：

- token/上下文浪费；
- 读同一文件的死循环；
- 大文件读取时加剧上下文压力。

## 变更

- `read_file` 增加参数 `dedup`（默认 `true`）：
  - 在同一 `session_id` 内，如果同一路径/范围参数的文件 `size+mtime` 未变化，则返回 stub：
    - `dedup=true`
    - `status=unchanged`
    - `message` 提示参考之前结果
    - `content_returned=false`
  - 文件发生变化后自动恢复返回完整内容（`dedup=false`）。

实现位置：

- `internal/tools/builtin.go`：新增读去重 tracker（上限 1000，超限清空）

## 验证

- `internal/tools/read_file_guardrails_test.go`：新增用例覆盖首次读返回内容、二次读返回 dedup stub、文件修改后再次返回内容。

### 来源：`0085-read-file-max-chars-reject.md`

# 086 Summary - read_file 超过 max_chars 默认拒绝（Hermes 行为）

## 背景

Hermes 的 `read_file` 对单次读取返回的字符数有安全上限；当返回内容超过 `max_chars` 时，会直接返回错误并提示使用 `offset/limit` 进行更精确的分页读取，以避免上下文被大文件占满。

Go 版 `agent-daemon` 之前在超过 `max_chars` 时会返回截断内容（`truncated=true`），与 Hermes 默认行为存在差异。

## 变更

- `read_file` 新增参数 `reject_on_truncate`（默认 `true`）：
  - 当读取将超过 `max_chars` 时：
    - `reject_on_truncate=true`：返回 `success=false` + `error`，并附带 `file_size/total_lines` 等元信息（Hermes 风格）。
    - `reject_on_truncate=false`：保持旧行为，返回截断内容 + `truncated=true`（兼容模式）。

实现位置：

- `internal/tools/builtin.go`：`read_file` 增加 oversize 拒绝逻辑与参数 schema。

## 验证

- `internal/tools/read_file_guardrails_test.go`：
  - 覆盖默认拒绝
  - 覆盖兼容模式下允许截断

### 来源：`0086-read-file-loop-guard.md`

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
