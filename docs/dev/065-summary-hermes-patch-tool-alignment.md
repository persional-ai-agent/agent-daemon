# 065 总结：Hermes patch 工具最小对齐

## 完成情况

- 新增内置工具 `patch`（字符串替换语义）。
- `file` toolset 纳入 `patch`。
- 补单测与文档索引。

## 边界

- 目前是最小 patch（old/new string），不支持 unified diff 与 fuzzy patch。

