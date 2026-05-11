# 063 计划：Hermes Toolsets 最小对齐

## 目标（可验证）

- `tools.enabled_toolsets` / `AGENT_ENABLED_TOOLSETS` 可限制 registry 仅保留解析后的工具集合。
- `agentd toolsets list` 输出内置 toolsets。
- `agentd toolsets resolve core` 输出 core toolset 展开后的工具名列表。
- 文档对齐矩阵更新：toolsets 从“未覆盖”调整为“部分对齐”。

## 实施步骤

1. 新增 `internal/tools/toolsets.go`：toolset 定义 + includes 解析。
2. 新增配置项：`tools.enabled_toolsets`（env：`AGENT_ENABLED_TOOLSETS`）。
3. Engine 构建时应用 toolset 过滤（先 enabled，再 disabled）。
4. 新增 CLI：`agentd toolsets list|resolve`。
5. 更新文档与索引，补单测。

