# 055 Research：CLI 模型管理最小对齐

## 背景

Hermes 提供 `hermes model` 入口用于查看和切换模型。当前 Go 项目已经具备 OpenAI、Anthropic、Codex 三类 provider 和 `agentd config set`，但缺少面向用户的模型切换命令。

## 目标

补齐最小模型管理面：

- 查看当前运行时 provider、model、base URL。
- 列出当前内置 provider。
- 写入 provider 与 provider 对应的 model 配置。

## 范围

- 只支持当前内置 provider：`openai`、`anthropic`、`codex`。
- 不做在线模型发现。
- 不处理 OAuth、凭据登录或 provider 插件。
- 不改变运行时模型调用逻辑。

## 推荐方案

- 在 `cmd/agentd` 增加 `model show|providers|set`。
- `model set openai gpt-4o-mini` 写入 `api.type=openai` 与 `api.model`。
- `model set anthropic claude-...` 写入 `api.type=anthropic` 与 `api.anthropic.model`。
- `model set codex gpt-5-codex` 写入 `api.type=codex` 与 `api.codex.model`。
- 可选 `-base-url` 写入对应 provider 的 `base_url`。

## 三角色审视

- 高级产品：模型切换是 Hermes CLI 体验中的核心高频操作，最小实现直接提升可用性。
- 高级架构师：复用已有 INI 管理能力，不扩展 provider 架构。
- 高级工程师：通过纯函数测试覆盖解析与配置键位，避免 CLI `os.Exit` 路径导致测试脆弱。
