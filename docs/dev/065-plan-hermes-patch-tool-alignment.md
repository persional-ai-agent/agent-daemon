# 065 计划：Hermes patch 工具最小对齐

## 目标（可验证）

- 新增内置工具 `patch`，并纳入 `file` toolset。
- `patch` 受 `AGENT_WORKDIR` 限制，避免越权写文件。
- 单测覆盖单次替换与多匹配保护策略。

## 实施步骤

1. 内置工具注册 `patch`。
2. 实现替换逻辑（与 `skill_manage patch` 一致的最小语义）。
3. toolsets `file` 增加 `patch`。
4. 更新文档与索引。

