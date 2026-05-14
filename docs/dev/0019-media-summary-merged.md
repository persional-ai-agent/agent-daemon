# 0019 media summary merged

## 模块

- `media`

## 类型

- `summary`

## 合并来源

- `0027-media-summary-merged.md`

## 合并内容

### 来源：`0027-media-summary-merged.md`

# 0027 media summary merged

## 模块

- `media`

## 类型

- `summary`

## 合并来源

- `0092-media-real-artifacts-and-browser-form.md`

## 合并内容

### 来源：`0092-media-real-artifacts-and-browser-form.md`

# 093 Summary - image_generate 生成真实 PNG、text_to_speech 占位音频增强、browser 表单输入态

## 变更

- `image_generate`：不再写占位文本文件，改为生成“确定性纯色”真实 PNG（512x512），便于下游工具/用户直接查看与处理。
- `text_to_speech`：占位输出从“静音”改为“简单正弦波 beep”，并按文本长度估算时长（0.5s~10s），便于快速验证音频管线。
- `browser_type`：在轻量 browser session 中缓存输入（`field -> text`）。
- `browser_press key=Enter`：尽力提交页面中第一个 `<form>`（仅支持 GET），把缓存输入作为 query 参数后 `browser_navigate` 跳转。

实现位置：

- `internal/tools/media_tools.go`
- `internal/tools/browser_light.go`
