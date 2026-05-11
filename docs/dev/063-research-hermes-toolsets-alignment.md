# 063 调研：Hermes Toolsets 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 用 `toolsets.py` 定义 toolset（工具集合）：

- toolset 可以直接列工具，也可以通过 `includes` 组合其它 toolset。
- 主要用途是缩减可用工具与 schema 面，减少 token 开销，并按场景启用能力（CLI、API server、cron、平台 bot 等）。

## 当前项目差异

本项目此前只有：

- 固定内置工具列表（register builtins + MCP discovery）。
- `tools.disabled` 禁用名单（从 registry 删除）。

缺少 Hermes 风格的 toolsets 组合与 “enabled_toolsets” 限制机制。

## 最小对齐目标

- 提供内置最小 toolsets（覆盖现有 builtins + cronjob）。
- 支持组合（includes）。
- 支持通过配置将 registry 收缩到指定 toolset 解析结果，从而缩减 schema 面。
- 提供 CLI 入口用于查看与解析 toolsets（便于调试与文档化）。

## 不在本次范围

- Hermes 的 toolset “可用性检查”（check_fn / 环境变量 gating）。
- 动态 schema patch、toolset 分发到不同平台配置、插件发现等。

