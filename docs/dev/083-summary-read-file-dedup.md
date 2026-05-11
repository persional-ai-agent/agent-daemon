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

