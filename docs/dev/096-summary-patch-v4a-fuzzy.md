# 096 Summary - patch V4A UPDATE hunk best-effort 容错

## 背景

Hermes 的 V4A patch 应用在 UPDATE 场景具备较强容错（fuzzy match）。Go 版初始实现偏严格，可能因轻微空白差异或上下文位置变化导致 hunk 失败。

## 变更

对 UPDATE hunk 的定位增加 best-effort 兜底策略：

- 精确序列匹配失败后：
  - 若存在 `@@ ... @@` 的 `context_hint`：在 hint 附近窗口内尝试匹配
  - 进行“空白归一化”（`strings.Fields`）后的序列匹配

实现位置：

- `internal/tools/v4a_patch.go`

