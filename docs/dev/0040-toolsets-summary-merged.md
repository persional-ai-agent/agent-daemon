# 0040 toolsets summary merged

## 模块

- `toolsets`

## 类型

- `summary`

## 合并来源

- `0050-toolsets-summary-merged.md`

## 合并内容

### 来源：`0050-toolsets-summary-merged.md`

# 0050 toolsets summary merged

## 模块

- `toolsets`

## 类型

- `summary`

## 合并来源

- `0106-toolsets-parity-aliases.md`

## 合并内容

### 来源：`0106-toolsets-parity-aliases.md`

# 107 Summary - Toolsets 名称对齐补充（Hermes toolsets-reference 兼容）

## 变更

为对齐 Hermes 的 toolsets 名称与引用（尤其是平台 toolset 名称），新增/补充以下 toolsets：

- 核心/别名：`todo`（=原 planning/todo）、`search`、`browser-cdp`
- 组合：`debugging`、`safe`
- 平台：`hermes-cli`、`hermes-api-server`、`hermes-acp`、`hermes-cron`、`hermes-telegram`、`hermes-discord`、`hermes-slack`、`hermes-yuanbao`

说明：

- 这些 toolsets 主要用于 **配置与名称兼容**（`AGENT_ENABLED_TOOLSETS` / `toolsets resolve`），不等价于 Hermes 的完整 UI/TUI 与动态注册逻辑。

## 修改文件

- `internal/tools/toolsets.go`
