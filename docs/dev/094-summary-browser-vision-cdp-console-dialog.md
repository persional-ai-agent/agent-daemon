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

