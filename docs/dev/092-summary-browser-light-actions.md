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

