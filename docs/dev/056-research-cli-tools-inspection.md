# 056 Research：CLI 工具查看最小对齐

## 背景

Hermes 提供 `hermes tools` 管理入口。当前 Go 项目已有 `agentd tools`，但只能列出工具名，无法查看模型实际接收的工具 schema。

## 目标

补齐最小工具查看能力：

- 保持 `agentd tools` 继续列出工具名。
- 新增 `agentd tools list` 显式列出工具名。
- 新增 `agentd tools show tool_name` 查看单个工具 schema。
- 新增 `agentd tools schemas` 输出完整工具 schema 列表。

## 范围

- 不实现工具启停配置。
- 不实现 toolset 解析。
- 不改变工具注册或 dispatch 行为。

## 推荐方案

在 `cmd/agentd` 中复用现有 `mustBuildEngine()` 与 `Registry.Schemas()`，只增加 CLI 输出层。`show/schemas` 使用 JSON pretty print，便于前端、SDK 或人工排查。

## 三角色审视

- 高级产品：解决工具可发现性不足，保留向后兼容。
- 高级架构师：不引入 toolset 或插件架构，避免超出当前阶段。
- 高级工程师：新增 helper 测试覆盖 schema 查找，运行时通过手动命令验证输出。
