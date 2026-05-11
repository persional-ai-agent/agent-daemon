# 067 计划：Hermes clarify 工具最小对齐

## 目标（可验证）

- 新增内置工具 `clarify`，并纳入 `toolsets.core`。
- `clarify` 对空 question 报错；对 options 做最小校验与清洗。
- 更新 docs/dev 索引与工具清单。

## 实施步骤

1. 在 builtin tools 中注册并实现 `clarify`。
2. 在 toolsets 中新增 `clarify` toolset，并让 core includes 它。
3. 更新文档与索引。

