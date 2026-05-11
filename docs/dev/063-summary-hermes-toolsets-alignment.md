# 063 总结：Hermes Toolsets 最小对齐

## 完成情况

- 新增最小 toolsets：`internal/tools/toolsets.go`，支持 `includes` 组合。
- 新增配置：`tools.enabled_toolsets`（env：`AGENT_ENABLED_TOOLSETS`）用于收缩 registry/tool schema 面。
- 新增 CLI：`agentd toolsets list`、`agentd toolsets resolve ...`。
- 文档对齐矩阵更新：toolsets 标记为“部分对齐”。

## 使用方式

- 推荐默认：不设置 `AGENT_ENABLED_TOOLSETS`，保持现有行为（完整内置工具 + MCP 工具）。
- 如需缩减：`AGENT_ENABLED_TOOLSETS=core`（或多个 toolset：`core,web`）。

## 边界

- 不做 Hermes 的 check_fn 可用性 gating；也不对 MCP tools 做独立分组过滤（仍会被 registry 过滤逻辑影响，取决于工具名是否在解析集合内）。

