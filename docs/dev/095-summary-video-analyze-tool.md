# 095 Summary - video_analyze 工具最小实现（ffprobe）

## 背景

Hermes 提供 `video` toolset（`video_analyze` 非 core，默认可选开启）。Go 版此前缺失该工具。

## 变更

- 新增 `video_analyze`：
  - 输入：`path`（workdir 内视频文件）、可选 `timeout`
  - 实现：调用 `ffprobe -show_format -show_streams -print_format json`
  - 若系统无 `ffprobe`，返回 `available=false`

实现位置：

- `internal/tools/video_analyze.go`
- `internal/tools/builtin.go`：注册 tool
- `internal/tools/toolsets.go`：新增 `video` toolset

