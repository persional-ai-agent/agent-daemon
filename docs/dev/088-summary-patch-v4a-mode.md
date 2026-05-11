# 088 Summary - patch 支持 V4A patch 格式（mode=patch）

## 背景

Hermes 的 `patch` 工具除字符串替换外，还支持 `mode="patch"`：使用 V4A patch 格式一次性对多个文件执行 Add/Update/Delete/Move 操作。

Go 版 `agent-daemon` 之前仅支持 `old_string/new_string` 的替换模式（单文件），功能缺口明显。

## 变更

- `patch` 增加参数：
  - `mode`：`replace`（默认）或 `patch`
  - `patch`：V4A patch 文本（当 `mode=patch` 必填）
- `mode=patch` 支持操作：
  - `*** Add File: ...`（创建文件）
  - `*** Update File: ...`（按 hunk 精确匹配并应用）
  - `*** Delete File: ...`（删除文件）
  - `*** Move File: src -> dst`（重命名/移动）
- 访问文件仍受 workdir 约束，并沿用现有安全护栏（拒绝 symlink 组件与非普通文件）。

实现位置：

- `internal/tools/v4a_patch.go`：V4A parser + apply
- `internal/tools/builtin.go`：`patch` 工具接入 `mode=patch`

## 边界与后续

- 当前 UPDATE 支持“精确匹配 + 基于 context_hint 的窗口匹配 + 空白归一化匹配”的 best-effort 容错；仍未达到 Hermes 的完整 fuzzy-match 能力。
