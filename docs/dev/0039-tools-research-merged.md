# 0039 tools research merged

## 模块

- `tools`

## 类型

- `research`

## 合并来源

- `0018-file-research-merged.md`
- `0023-homeassistant-research-merged.md`
- `0034-process-research-merged.md`
- `0049-tool-research-merged.md`

## 合并内容

### 来源：`0049-tool-research-merged.md`

# 0049 tool research merged

## 模块

- `tool`

## 类型

- `research`

## 合并来源

- `0058-tool-disable-config.md`

## 合并内容

### 来源：`0058-tool-disable-config.md`

# 059 Research：工具禁用配置最小对齐

## 背景

Hermes 支持通过工具配置控制可用工具。当前 Go 项目已经能查看工具 schema，但所有注册工具都会暴露给模型，缺少最小工具启停能力。

## 目标

实现最小工具禁用配置：

- 支持环境变量 `AGENT_DISABLED_TOOLS`。
- 支持 INI `[tools] disabled = terminal,web_fetch`。
- 支持 CLI `agentd tools disable|enable|disabled`。
- 被禁用工具不出现在 registry names/schemas 中，也无法 dispatch。

## 范围

- 只实现禁用列表，不实现 toolset 分组。
- 不实现按平台/会话的工具集。
- 不实现 allowlist 模式。

## 推荐方案

- 在 `config.Config` 增加 `DisabledTools`。
- 在 `tools.Registry` 增加 `Disable`。
- `mustBuildEngine` 完成 builtins/MCP 注册后，统一删除禁用工具。
- CLI 通过 `[tools] disabled` 写入逗号分隔列表。

## 三角色审视

- 高级产品：用户可以关闭高风险或不需要的工具，直接提升可控性。
- 高级架构师：采用简单 denylist，不引入 Hermes 完整 toolset 架构。
- 高级工程师：测试覆盖列表解析、registry 过滤、配置加载。
