# 090 Summary - browser/vision/image/tts 从 stub 升级为最小可用实现

## 背景

Hermes core tools 中包含 browser/vision/image/tts 相关工具。Go 版 `agent-daemon` 之前仅提供 stub（存在工具名但返回 not implemented），影响“工具名对齐但不可用”的体验。

## 变更

- `vision_analyze`：最小实现，读取 workdir 内图片并返回 `format/width/height` 元信息（不做语义识别）。
- `image_generate`：最小实现，输出占位文件到 `output_path`（当前不生成真实图片内容）。
- `text_to_speech`：最小实现，输出 16kHz PCM silence 的占位 WAV 到 `output_path`。
- `browser_*`：
  - `browser_navigate`：轻量实现（HTTP GET 抓取 HTML，不执行 JS）
  - `browser_snapshot`：将 HTML 粗略转为纯文本快照
  - `browser_back`：支持轻量 history 回退
  - 其他 browser 动作保留但返回 `not supported`（缺少真实浏览器引擎）

实现位置：

- `internal/tools/browser_light.go`
- `internal/tools/media_tools.go`
- `internal/tools/builtin.go`：替换原 stub 注册为上述实现

## 边界与后续

上述实现仅用于“最小可用/非 stub”，与 Hermes 完整 browser/vision/image/tts 能力仍有差距；如需完全对齐需引入真实浏览器与多模态模型后端。

