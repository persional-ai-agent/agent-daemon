# 109 Summary - image_generate 增加 FAL_KEY gate（Hermes 可用性对齐）

## 变更

- `image_generate`：当缺少 `FAL_KEY` 时返回 `available=false`（避免“看似可用但实际不是远端生成”的误导）。
- 仍保留 placeholder 输出实现（在 key 存在时返回本地生成的确定性 PNG），后续可按需接入真实 FAL backend。

## 修改文件

- `internal/tools/media_tools.go`

