# 0005 codex research merged

## 模块

- `codex`

## 类型

- `research`

## 合并来源

- `0006-codex-research-merged.md`

## 合并内容

### 来源：`0006-codex-research-merged.md`

# 0006 codex research merged

## 模块

- `codex`

## 类型

- `research`

## 合并来源

- `0005-codex-responses-mode.md`

## 合并内容

### 来源：`0005-codex-responses-mode.md`

# 006 调研：Codex Responses 模式补齐

## 背景

在 005 阶段已补齐 OpenAI + Anthropic 双模式，但 Hermes 侧仍有 Responses/Codex 兼容能力。

为继续缩小核心差距，本阶段补齐 `provider=codex` 的最小可用模式。

## 差异点

- 当前无 `/responses` 接口 client
- 无 `function_call` 与 `function_call_output` 映射
- 无 codex provider 配置项

## 方案

保持 `model.Client` 不变，新增 `CodexClient`：

- 请求路径：`/responses`
- 输入映射：
  - user/assistant/system 消息映射为 `type=message`
  - assistant 工具调用映射为 `type=function_call`
  - tool 消息映射为 `type=function_call_output`
- 输出映射：
  - `message` 内容反解到 `assistant.content`
  - `function_call` 反解到 `assistant.tool_calls`

## 结论

该实现可在不改 Agent Loop 的前提下，把 provider 侧能力扩展到 OpenAI/Anthropic/Codex 三模式，为后续继续补齐 provider 级高级特性打下基础。
