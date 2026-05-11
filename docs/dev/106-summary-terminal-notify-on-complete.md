# 106 Summary - terminal(background) 补齐 notify_on_complete（Hermes 体验对齐）

## 变更

- `terminal` 工具新增参数 `notify_on_complete`：
  - 当 `background=true` 且 `notify_on_complete=true` 时，进程结束会 best-effort 发出一个工具事件 `process_complete`（通过现有 ToolEventSink 通道）。

## 修改文件

- `internal/tools/builtin.go`

