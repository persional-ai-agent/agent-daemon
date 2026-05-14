# 0003 browser summary merged

## 模块

- `browser`

## 类型

- `summary`

## 合并来源

- `0003-browser-summary-merged.md`

## 合并内容

### 来源：`0003-browser-summary-merged.md`

# 0003 browser summary merged

## 模块

- `browser`

## 类型

- `summary`

## 合并来源

- `0089-browser-vision-image-tts-minimal.md`
- `0091-browser-light-actions.md`
- `0093-browser-vision-cdp-console-dialog.md`
- `0109-browser-snapshot-ref-ids.md`
- `0118-browser-cdp-backend.md`
- `0119-browser-cdp-console-dialog.md`

## 合并内容

### 来源：`0089-browser-vision-image-tts-minimal.md`

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

### 来源：`0091-browser-light-actions.md`

# 092 Summary - browser_click/get_images/type/scroll/press 轻量实现

## 背景

Hermes 的 browser 工具集包含 click/type/scroll/press/get_images 等动作。Go 版 `agent-daemon` 在无真实浏览器引擎的前提下，之前仅实现 `browser_navigate/snapshot/back`，其余动作返回 not supported。

## 变更

在“轻量 HTTP 抓取 + HTML 文本解析”的实现基础上补齐部分动作：

- `browser_click`：在当前 HTML 中解析 `<a href=...>...</a>`，按 `text`（link 文本包含）或 `href_contains` 匹配，找到后执行一次 `browser_navigate` 跳转。
- `browser_get_images`：解析 `<img src=...>`，返回解析到的图片 URL（相对 URL 会按当前页面 URL resolve）。
- `browser_type` / `browser_scroll` / `browser_press`：作为轻量 no-op（无 DOM/JS），返回 `success=true` 并提示该限制，用于与 Hermes 工具名对齐且避免直接失败。

实现位置：

- `internal/tools/browser_light.go`
- `internal/tools/builtin.go`：更新 browser_* 注册与 schema

## 边界

不执行 JS，不维护 DOM 状态，适用于静态页面与简单超链接跳转；复杂交互仍需要真实浏览器引擎后端。

### 来源：`0093-browser-vision-cdp-console-dialog.md`

# 094 Summary - browser_vision/browser_cdp/browser_console/browser_dialog 轻量实现

## 背景

Hermes core tools 中包含 `browser_vision/browser_console/browser_cdp/browser_dialog`。Go 版此前保留工具名但返回 not supported。

## 变更

在轻量 browser（HTTP 抓取 HTML）基础上补齐最小实现：

- `browser_vision`：解析 `<img src>`，按 `limit` 下载图片并返回 `format/width/height/bytes` 元信息（不做语义识别）
- `browser_cdp`：返回当前页面的 `status/headers/html_len`，可选 `include_html=true` 返回截断 HTML
- `browser_console`：返回空 logs（轻量实现无 JS，不产生 console）
- `browser_dialog`：返回 `dialog=nil`（轻量实现无 JS，不产生 dialog）

实现位置：

- `internal/tools/browser_light.go`
- `internal/tools/builtin.go`：更新注册与 schema

### 来源：`0109-browser-snapshot-ref-ids.md`

# 110 Summary - browser_snapshot/ref 对齐增强（轻量实现）

## 变更

- `browser_snapshot`：输出增加 best-effort ref IDs（`@e1/@e2/...`），并返回 `pending_dialogs: []` 字段（Hermes 风格）。
- `browser_click`：新增 `ref` 参数，可直接点击 snapshot 中的 ref（链接）。
- `browser_type`：新增 `ref` 参数，可对 snapshot 中的 input ref 写入（用于 GET 表单提交的 best-effort）。
- `browser_snapshot` schema 增加 `full` 参数（兼容 Hermes；轻量实现中忽略）。

## 修改文件

- `internal/tools/browser_light.go`
- `internal/tools/builtin.go`

### 来源：`0118-browser-cdp-backend.md`

# 119 - Summary: Optional real CDP backend for `browser_*` tools

## Goal

Upgrade the existing lightweight HTTP-only browser implementation to an optional “real browser” mode that can execute JS and interact with the live DOM, closer to Hermes `browser-cdp` toolset expectations.

## What changed

- Added a minimal Chrome DevTools Protocol client:
  - `internal/tools/browser_cdp_client.go`
  - Supports connecting via:
    - direct `ws://...` target URL
    - HTTP base like `http://127.0.0.1:9222` (resolves `/json/version` or `/json/new`)
- Added CDP-backed implementations for key browser tools:
  - `internal/tools/browser_cdp_impl.go`
  - `browser_navigate` / `browser_snapshot` / `browser_click` / `browser_type` / `browser_press`
  - `browser_cdp` returns live DOM metadata and optional HTML (`include_html=true`)
- Wired tool selection:
  - If `BROWSER_CDP_URL` is set (and `BROWSER_CDP_ENABLED` is not `false`), the standard `browser_*` tools use CDP.
  - Otherwise they continue to use the lightweight HTTP mode.

## Configuration

- Enable CDP mode:
  - `BROWSER_CDP_URL=ws://...` (recommended) or `BROWSER_CDP_URL=http://127.0.0.1:9222`
- Disable explicitly:
  - `BROWSER_CDP_ENABLED=false`

## Notes / limitations

- Console log collection and JS dialogs are still not implemented in CDP mode (tools return empty logs / nil dialog with a note).
- Ref IDs (`@e1`...) in CDP mode are derived from a JS snapshot list; they can become stale if the page changes.

### 来源：`0119-browser-cdp-console-dialog.md`

# 120 - Summary: CDP browser console + dialog support

## Goal

Bring CDP mode closer to Hermes expectations for:

- `browser_console`: return console output / JS errors
- `browser_dialog`: accept/dismiss native JS dialogs and surface `pending_dialogs` via `browser_snapshot`

## What changed

- `internal/tools/browser_cdp_client.go`:
  - Parses CDP event envelopes (`method`/`params`) and forwards to an event callback.
- `internal/tools/browser_cdp_impl.go`:
  - Buffers logs from:
    - `Runtime.consoleAPICalled`
    - `Runtime.exceptionThrown`
    - `Log.entryAdded`
  - Tracks dialogs from:
    - `Page.javascriptDialogOpening` / `Page.javascriptDialogClosed`
  - `browser_snapshot` now populates `pending_dialogs` when a dialog is open.
  - `browser_console` returns and clears buffered logs.
  - `browser_dialog(action=accept|dismiss, prompt_text=...)` calls `Page.handleJavaScriptDialog`.
- `internal/tools/builtin.go`:
  - Updates `browser_dialog` schema to accept `action` and `prompt_text` (Hermes-compatible).

## Notes

- CDP mode must be enabled (`BROWSER_CDP_URL` set) for these behaviors; otherwise tools remain lightweight.
- Logs are buffered per session and capped (200 entries).
