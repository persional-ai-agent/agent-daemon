# 0015 hermes plan merged

## 模块

- `hermes`

## 类型

- `plan`

## 合并来源

- `0022-hermes-plan-merged.md`

## 合并内容

### 来源：`0022-hermes-plan-merged.md`

# 0022 hermes plan merged

## 模块

- `hermes`

## 类型

- `plan`

## 合并来源

- `0000-hermes-agent-go-port.md`
- `0001-hermes-gap-closure.md`
- `0052-hermes-feature-alignment.md`
- `0061-hermes-cron-alignment.md`
- `0062-hermes-toolsets-alignment.md`
- `0063-hermes-send-message-alignment.md`
- `0064-hermes-patch-tool-alignment.md`
- `0065-hermes-web-tools-alignment.md`
- `0066-hermes-clarify-tool-alignment.md`
- `0067-hermes-execute-code-alignment.md`

## 合并内容

### 来源：`2026-05-14-hermes-functional-gap-todo`

# Hermes 功能差异补齐 TODO 总计划

## 目标

对齐 `/data/source/hermes-agent/` 的完整产品功能面。忽略技术栈差异，仅补齐用户可感知能力与系统行为差异。

当前项目已经覆盖 Hermes 的核心 Agent daemon 主链路：Agent Loop、工具调用、会话存储、记忆、MCP、Skills、Provider 韧性、HTTP/SSE/WebSocket、基础 Gateway、插件基础闭环、Cron 最小闭环、Research trajectory 最小闭环。

后续补齐重点不是重写核心 Loop，而是按入口体验、网关生态、工具能力、学习闭环、自动化和研究链路逐步扩展。

## 执行原则

- 一次实现一个大功能，完成后单独提交。
- 先做用户可见主链路，再补外围生态。
- 每个 TODO 都必须包含：代码实现、必要测试、文档更新。
- CLI、TUI、Gateway、Web 中同名动作必须保持语义一致。
- 不追求逐文件复刻 Hermes，只补功能行为。

## 进度看板（2026-05-14）

状态定义：
- `done`：已满足原验收标准，且入口可稳定使用。
- `partial`：已落地核心子功能，但未满足完整验收标准。
- `todo`：尚未进入可用实现阶段。

| TODO | 状态 | 当前结论 | 证据 |
|---|---|---|---|
| `TODO-001` | `partial` | TUI/CLI 已完成多轮重构与流式链路收口，但仍有交互与渲染边角待稳定。 | `docs/dev/0036-summary-summary-merged.md`（`# 274`） |
| `TODO-002` | `partial` | 网关命令一致性与部分统一已做，未完成统一 dispatcher 全闭环。 | `docs/dev/0036-summary-summary-merged.md`（`# 257`） |
| `TODO-003` | `partial` | `send_message` home target 与目标模型已补一部分，跨平台 continuity 未完全闭环。 | `docs/dev/0036-summary-summary-merged.md`（`# 111`、`# 281`） |
| `TODO-004` | `done` | `signal`、`email`、`webhook`、`homeassistant` 已具备最小 inbound/outbound 闭环，并接入 gateway setup/status/platforms 与运行时适配器装配。 | `internal/gateway/platforms/signal.go`、`internal/gateway/platforms/email.go`、`internal/gateway/platforms/webhook.go`、`internal/gateway/platforms/homeassistant_adapter.go`、`cmd/agentd/main.go` |
| `TODO-005` | `done` | 已补 `matrix` + `feishu` + `dingtalk` + `wecom` + `mattermost` + `sms` + `bluebubbles` 网关适配器（inbound/outbound 最小闭环），并接入 gateway setup/status/platforms、运行时装配与 webhook API 路由。 | `internal/gateway/platforms/matrix.go`、`internal/gateway/platforms/feishu.go`、`internal/gateway/platforms/dingtalk.go`、`internal/gateway/platforms/wecom.go`、`internal/gateway/platforms/mattermost.go`、`internal/gateway/platforms/sms.go`、`internal/gateway/platforms/bluebubbles.go`、`internal/api/server.go`、`cmd/agentd/main.go` |
| `TODO-006` | `partial` | 审批命令链路已推进，平台原生交互深度仍未齐。 | `docs/dev/0036-summary-summary-merged.md`（`# 257`） |
| `TODO-007` | `partial` | 多个工具从 stub 升级到最小可用，但与能力级实现仍有差距。 | `docs/dev/0036-summary-summary-merged.md`（`# 090`~`# 110`） |
| `TODO-008` | `partial` | toolsets 最小对齐已完成，动态可用性与 UI 管理仍未完全到位。 | `docs/dev/0036-summary-summary-merged.md`（`# 063`、`# 107`） |
| `TODO-009` | `partial` | provider 运行时与插件闭环有进展，profile/能力矩阵未完整。 | `docs/dev/0036-summary-summary-merged.md`（`# 255`、`# 279`） |
| `TODO-010` | `partial` | skills 多源与管理能力已补，自动学习与审计回滚未完整。 | `docs/dev/0036-summary-summary-merged.md`（`# 051`、`# 052`、`# 275`） |
| `TODO-011` | `partial` | memory 学习闭环有阶段成果，insights/外部 provider 仍待补。 | `docs/dev/0036-summary-summary-merged.md`（`# 275`） |
| `TODO-012` | `partial` | cron 表达式、投递、链式、脚本已补，重试/并发/审计还需补完。 | `docs/dev/0036-summary-summary-merged.md`（`# 280`~`# 283`） |
| `TODO-013` | `partial` | ACP/IDE 最小适配已完成，完整协议层能力未齐。 | `docs/dev/0036-summary-summary-merged.md`（`# 258`） |
| `TODO-014` | `partial` | research/trajectory 最小运行闭环已完成，策略评估与导出体系需增强。 | `docs/dev/0036-summary-summary-merged.md`（`# 259`） |
| `TODO-015` | `partial` | setup/install/update 等有进展，迁移与可回滚备份恢复未闭环。 | `docs/dev/0036-summary-summary-merged.md`（`# 054`~`# 058` 及后续安装相关总结） |
| `TODO-016` | `partial` | Web 管理面已补 dashboard/cron/model-provider，距“日常可完全替代 CLI”仍有缺口。 | `docs/dev/0036-summary-summary-merged.md`（`# 277`~`# 279`） |

## 下一迭代优先级（先做功能）

1. `TODO-001`：收口 TUI/CLI 流式渲染与输入稳定性残留问题。
2. `TODO-002`：统一 command dispatcher，彻底消除 CLI/TUI/Gateway 语义漂移。
3. `TODO-006`：补平台原生命令与审批交互深度。
4. `TODO-003`：补齐 `send_message` continuity 与跨平台会话映射闭环。
5. `TODO-008`：补 toolsets 动态可用性与 UI 管理闭环。

## P0：先稳定用户入口与会话体验

### TODO-001：完整 TUI/CLI 交互体验

目标：

- 对齐 Hermes 的完整终端体验：多行编辑、slash 自动补全、历史浏览、中断后重定向、流式工具输出。
- 让 `ui-tui` 成为主 CLI 入口，而不是只作为事件显示面。

范围：

- `ui-tui/`
- `internal/cli/`
- `cmd/agentd`

功能项：

- 多行输入与编辑。
- slash 命令补全。
- 命令历史与会话历史浏览。
- `Ctrl+C` 中断当前 turn，并允许输入新消息重定向。
- 流式 assistant / tool / thinking 分块稳定展示。
- Markdown 流式稳定渲染，最终态不重复输出。
- 多轮 session 内 user/assistant/tool 顺序稳定。

验收：

- 一个 session 内连续发送 5 轮消息，输出顺序保持 `user -> assistant/tool -> user -> assistant/tool`。
- `/new`、`/reset`、`/retry`、`/undo`、`/compress`、`/usage`、`/skills`、`/personality`、`/model` 可在 TUI 中执行。
- 工具执行过程中能实时显示开始、参数摘要、完成/失败状态。
- 运行 `go test ./ui-tui` 与相关 CLI 测试通过。

### TODO-002：CLI 与消息平台命令语义统一

目标：

- 让同一 slash 命令在 CLI、TUI、Gateway 中行为一致。

最新进展（2026-05-15）：

- 已将 gateway identity/session resolve 连同 CLI/API/Gateway 三端统一到 `internal/tools/gateway_identity.go`，消除三套 continuity/global-id 解析漂移。
- Gateway `identity_store` 已改为复用 `internal/tools` 的持久化读写能力；`runner` 的 `/resolve` 已复用同一 resolve 结果结构。
- 已将 `/sethome` 参数解析统一到 `internal/tools.ParseSetHomeArgs`，CLI/TUI/Gateway 全部复用同一解析路径，消除 `<platform:chat_id>` 与 `<platform> <chat_id>` 两种写法的行为漂移。
- 已将 `/resolve` 参数解析统一到 `internal/tools.ParseGatewayResolveArgs`（Gateway 支持默认当前会话上下文走 `ParseGatewayResolveArgsWithDefaults`），CLI/TUI/Gateway 复用同一校验规则。
- 已将 `/whoami`、`/setid`、`/unsetid`（含 gateway 内 `/setid`）参数解析统一到 `internal/tools`，三端共享同一 identity 参数校验规则。
- 已将 `/continuity` 更新参数归一化到 `internal/tools.ParseGatewayContinuityModeArg`，CLI/TUI/Gateway 三端共享同一 mode 校验与别名归一（`id/name` -> `user_id/user_name`）。
- 已统一 continuity 读取语义到 `internal/tools.ResolveGatewayContinuityMode`（环境变量 `AGENT_GATEWAY_CONTINUITY` 优先，配置文件兜底），并接入 CLI/API/Gateway/session-resolve，消除展示与路由分歧。
- 已将 `/model` 参数解析统一到 `internal/tools.ParseGatewayModelSpecArgs`，CLI/TUI/Gateway 共用同一 `provider:model` / `provider model` 解析与校验路径。
- 已统一 model 偏好读取/更新到 `internal/tools.ResolveGatewayModelPreference` 与 `UpdateGatewayModelPreference`，CLI/Gateway 共用同一持久化与环境变量同步逻辑。
- 已统一 model 偏好默认展示策略到 `internal/tools.DisplayGatewayModelPreference`，并让 `/v1/ui/model` 返回 `model + preference`（保留 ModelInfo 输出并追加统一偏好视图）。
- 已将 `/whoami` 与 `/resolve` 的响应结构与文本拼装统一到 `internal/tools`（`Build*Payload`/`Render*Text`），CLI/API/Gateway 复用同一结果字段集合。
- 已将 UI API 层 gateway 参数校验入口改为复用 `internal/tools` 解析函数（identity/ref/setid/resolve），消除 API 与 CLI/TUI/Gateway 的参数语义漂移。
- 已新增 gateway 状态/诊断共享渲染辅助（`internal/tools/gateway_status.go`），并接入 Gateway `/status` 元数据与 API diagnostics fallback，统一关键字段映射。
- 已将 Gateway `/status` 文本与元数据统一复用单一 `gatewayStatusSnapshot()` 计算路径，消除同命令内部双路径字段差异风险。
- 已新增 `ExtractGatewayStatusSnapshot` 标准化提取函数并补测试；当前保持 API 对外契约不变，为后续状态字段进一步对齐提供统一转换入口。
- 已新增 `NormalizeGatewayStatusMap` 并接入 API gateway status 读取路径；在保持现有契约不变前提下，统一内部状态标准化入口。
- 已新增 `NormalizeGatewayDiagnosticsMap` 并接入 API gateway diagnostics 两条分支（自定义/fallback），统一内部诊断结构标准化入口且不改变外部契约。
- 已将 gateway status/diagnostics 标准化逻辑沉淀到 `internal/tools/gateway_status.go`（snapshot 提取 + status/diagnostics 归一），后续多入口字段对齐可直接复用。
- 已新增 `UpdateGatewayContinuityMode` 并接入 CLI/API/Gateway continuity 更新路径，统一“参数归一化 + env 同步 + 持久化”写入行为。
- 已将 API `/v1/ui/model/set` 切到共享校验与写入路径（`ParseGatewayModelSpecArgs` + `UpdateGatewayModelPreference`），并补 `model_base_url` 统一写入函数。
- 已统一 `/model` 用法提示文案来源到 `internal/tools`（`GatewayModelUsageEN/ZH`），CLI/Gateway/UI-TUI/API 共用同一 usage 表达，减少提示漂移。
- 已统一 gateway 常用命令 usage 文案来源（`sethome/continuity/whoami/resolve/setid/unsetid`），CLI/Gateway/UI-TUI 复用 shared helper，降低多入口文案偏差。
- 已将 API gateway `invalid_argument`（continuity/identity/resolve）错误文案切到 `internal/tools` 统一 helper，减少 API 与其他入口提示语义偏差。
- 已新增 `BuildGatewayIdentityPayload` 并接入 API `/v1/ui/gateway/identity`（GET/POST/DELETE）与 CLI `setid/unsetid` 输出，统一 identity 成功响应字段风格。
- 已新增 `BuildGatewayContinuityPayload` 并接入 API/CLI/Gateway `/continuity` 成功输出（含 Gateway 命令元数据），统一 continuity 字段结构为 `continuity_mode`。
- 已新增 `/model` 成功响应共享 helper（`BuildGatewayModelPayload` / `BuildGatewayModelUpdatePayload`）并接入 CLI/Gateway/API，统一查询与更新字段风格（`provider/model/base_url/updated/note`）。
- 已新增 `/sethome` 成功响应共享 helper（`BuildSetHomePayload`）并接入 CLI/Gateway/API，统一 `platform/chat_id/target/home_target/env` 字段语义。
- 已新增 `/targets` 成功响应共享 helper（`BuildTargetsPayload`）并接入 CLI/Gateway/API，统一 `platform/count/platforms/targets` 字段集合与语义。
- 已新增 `session usage/stats` 共享响应 helper（`BuildSessionUsagePayload` / `BuildSessionStatsPayload`），并接入 CLI `/usage`、CLI `/stats`、Gateway `/usage` 元数据，统一 `session_id + usage/stats` 结构。
- 已新增 `session compress` 共享响应 helper（`BuildSessionCompressPayload`），并接入 CLI `/compress` 与 Gateway `/compress` 元数据，统一 `session_id/compacted/before/after/dropped/tail_messages` 字段语义。
- 已新增 `session save` 共享响应 helper（`BuildSessionSavePayload`），并接入 CLI `/save` 与 Gateway `/save` 元数据，统一 `session_id/path/messages` 字段语义。
- 已新增 `session reload` 共享响应 helper（`BuildSessionReloadPayload`），并接入 CLI `/reload` 与 Gateway `/reload` 元数据，统一 `session_id/count/messages` 字段语义。
- 已新增 `session undo` 共享响应 helper（`BuildSessionUndoPayload`），并接入 CLI `/undo` 与 Gateway `/undo` 元数据，统一 `session_id/removed_messages/messages_in_context` 字段语义。
- 已新增 `session clear` 共享响应 helper（`BuildSessionClearPayload`），并接入 CLI `/clear` 与 Gateway `/clear` 元数据，统一 `previous_session_id/session_id/cleared` 字段语义（行为保持各入口原语义）。
- 已新增 `session recover` 共享响应 helper（`BuildSessionRecoverPayload`），并接入 CLI `/recover context` 与 Gateway `/recover context` 元数据，统一 `recovered/previous_session_id/session_id` 字段语义（Gateway 额外标记 `replay=true`）。
- 已新增 `session switch` 共享响应 helper（`BuildSessionSwitchPayload`），并接入 CLI `/new` `/reset` `/resume` 与 Gateway 对应命令元数据，统一 `previous_session_id/session_id/reset/loaded_messages` 字段语义。
- 已新增 `session overview` 共享响应 helper（`BuildSessionOverviewPayload`），并接入 CLI `/session` `/status` 与 Gateway `/session` 查询元数据，统一 `session_id/route_session/messages_in_context/tools` 字段语义。
- 已新增 `session show` 共享响应 helper（`BuildSessionShowPayload`），并接入 CLI `/show` 与 Gateway `/show` `/next` `/prev` 元数据，统一 `session_id/offset/limit/count/messages` 字段语义。
- 已新增 `session list` 共享响应 helper（`BuildSessionListPayload`），并接入 CLI `/sessions` 与 Gateway `/sessions` 元数据，统一 `count/limit/sessions` 字段语义。
- 已将 Gateway `/stats` 元数据切换为复用 `BuildSessionStatsPayload`，与 CLI `/stats` 统一为 `session_id + stats` 结构。
- 已新增 `session history/pick` 共享响应 helper（`BuildSessionHistoryPayload` / `BuildSessionPickPayload`），并接入 Gateway `/history` 与 `/pick` 成功元数据，统一会话导航字段语义。
- 已新增 `BuildGatewayIdentityBindPayload` 并接入 Gateway `/setid`，同时 Gateway `/unsetid` 改为复用 `BuildGatewayIdentityPayload`，统一身份绑定命令成功元数据字段语义。
- 已新增 `BuildApprovalCommandPayload` 并批量接入 Gateway 审批命令元数据（`/approve` `/deny` `/approvals` `/pending` `/grant` `/revoke`），统一审批命令返回字段语义。
- 已新增通用命令元数据 helper（`BuildSlashPayload` / `BuildSlashModePayload`），并批量接入 Gateway 管理命令成功路径（`/skills` `/tools` `/personality` `/queue` `/help`），统一 `slash/mode/subcommand` 字段构造方式。
- 已将 Gateway 命令分发中大量仅含 `slash` 的手写元数据 map（覆盖会话、投递、模型、审批、导航等 usage/error 分支）批量替换为 `BuildSlashPayload(...)`，进一步减少命令元数据构造漂移面。
- 已新增 CLI 管理命令共享 payload helper（`BuildCollectionPayload` / `BuildMemoryContentPayload` / `BuildMemorySnapshotPayload` / `BuildPersonalityPayload`），并批量接入 CLI `/todo` `/memory` `/personality` `/tools` `/toolsets` 等成功路径，减少 CLI 侧手写响应 map 漂移。
- 已在 Gateway command 分发中进一步批量替换 `slash` 元数据构造（覆盖大量 usage/error 分支）为 `BuildSlashPayload(...)`，将命令元数据入口进一步收敛到 shared helper。
- 已补充 CLI 单对象成功响应共享 helper（`BuildObjectPayload`），并接入 `/tools show` 与 `/toolsets show`，进一步减少 CLI 手写 payload 分支。
- 已新增 UI API 会话操作共享 payload helper（`BuildUISession*Payload`），并批量接入 `/v1/ui/sessions/branch|resume|compress|undo|replay` 成功返回，统一 API 会话操作响应字段语义。
- 已新增 UI API 成功封装 helper（`BuildUIResultEnvelope`），并批量替换多处 `ok+result` 直写返回（如 model set、gateway identity、approval confirm、cron action 等路径），进一步收敛 API 响应封装入口。
- 已进一步批量替换 UI API 侧剩余纯 `ok+result` 成功响应（覆盖 config set、targets home、session branch/resume/compress/undo/replay、gateway continuity/identity/session-resolve、gateway action、skills reload/search/sync 等），统一复用 `BuildUIResultEnvelope`，并保留含兼容字段的接口返回不变。
- 已将 Gateway 命令分发中最后两处手写 `slash` 元数据 map（`/skills show` 与 `/tools show` 的 not-found 分支）改为复用 `BuildSlashModePayload`，进一步消除命令元数据构造漂移点。
- 已新增 `AttachSlashPayload` 并批量接入 Gateway `runner` 成功元数据构造路径（session/whoami/resolve/continuity/setid/unsetid/history/show/sessions/pick/stats/new/reset/resume/recover/undo/clear/reload/save/sethome/targets/compress/usage/model），清理全部 `meta["slash"]=...` 手写赋值分支，进一步收敛 command dispatcher 元数据语义。
- 已新增 `BuildSlashSubcommandPayload` 并批量接入 Gateway `/skills`、`/tools` 成功元数据路径，清理 `meta["subcommand"]=...` 手写赋值分支，统一 `slash+subcommand` 构造入口。
- 已在 Gateway `runner` 内新增通用 `mergePayloadMeta`，并批量替换 delivery hook 与 `/status` 命令中的手写 map 合并循环，统一元数据合并路径、减少分发逻辑重复分支。
- 已在 Gateway `runner` 新增 `sendSlashText`，并批量替换 slash 命令 usage/error/help 等 70+ 处分支的 `sendText + BuildSlashPayload` 重复调用，统一命令回包发送样板。
- 已在 Gateway `runner` 新增 `sendSlashModeText` 与 `sendSlashSubcommandText`，并批量接入 `/personality`、`/skills`、`/tools` 的通用回包分支，继续收敛 `sendText + BuildSlashMode/SubcommandPayload` 重复调用。
- 已新增 `BuildSlashModePayloadWithExtra` 与 `BuildSlashSubcommandPayloadWithExtra`，并批量接入 `/skills show` 与 `/tools show`（成功/未找到）分支，清理 `mode/subcommand payload` 后续手工 `name/tool` 字段拼装。
- 已在 Gateway `runner` 新增 `sendApprovalText`，并批量接入审批命令回包分支（`/approve` `/deny` `/approvals` `/pending` `/grant` `/revoke`），统一 `sendText + BuildApprovalCommandPayload` 发送样板。
- 已在 Gateway `runner` 新增 `sendMetaText`，并批量替换 success 回包分支中 `sendText(..., meta)` 的重复调用，进一步压缩 command dispatcher 重复样板代码。
- 已新增 `BuildAuthPayload`（tools）与 `sendAuthText`（gateway runner），并替换未授权回包的手写 `auth` 元数据 map，继续统一命令回包元数据构造入口。
- 已新增共享命令用法常量与 `UsageEN/UsageZH`（`internal/tools/command_usage.go`），并批量替换 CLI/Gateway 中 session/tools/toolsets/personality/targets/compress/usage 等命令的硬编码 usage 文案来源，进一步降低多入口命令提示漂移。
- 已补齐共享提示文案 helper（`CLIWelcomeHintZH`/`UnknownCommandMessageZH`）并接入 CLI，同时将 TUI 中 `/new`、`/resume`、`/personality` 的用法错误提示切到共享 usage 常量，继续收敛 CLI/TUI/Gateway 命令提示来源。
- 已补齐 TUI 侧 session/approval/show 命令的共享 usage 常量接入（`/recover` `/reset` `/save` `/pick` `/usage` `/compress` `/targets` `/pending` `/sessions` `/show` `/stats` `/open`），并同步替换 Gateway `/compress` 的残留 usage 硬编码，进一步减少多入口命令提示漂移。
- 已进一步批量替换 TUI 命令分发中的固定 `用法:` 提示（覆盖 panel/view/fullscreen/diag/reconnect/rerun/bookmark/workbench/workflow/gateway/config 及审批提示），统一接入 `internal/tools/command_usage.go` 常量与 `UsageZH`；保留仅少量依赖动态参数占位的提示为运行时格式化。
- 已新增动态 usage helper（`UsageZHOptionalN` / `UsageZHOptionalNPositive` / `UsageZHRequiredIndex` / `UsageZHRequiredIndexPositive` / `UsageZHActionIndexRange`）并接入 TUI 剩余动态提示分支（`/actions` 与通用数字参数解析函数），进一步收敛命令提示来源到 `internal/tools/command_usage.go`。
- 已新增英文侧共享提示 helper（`UsageENEither` / `NotSupportedBySessionStoreEN`），并批量接入 Gateway `runner` 的会话能力不支持提示与 `approve|deny` usage 拼装分支，减少命令错误提示重复拼接。
- 已新增 CLI 错误提示共享 helper（`SessionStoreUnavailableEN` / `SessionStoreNotSupportedZH` / `CLICancelNotSupportedZH`），并批量替换 CLI slash 命令中会话存储不可用/能力不支持/`/cancel` 不支持等硬编码文案，继续降低 CLI/TUI/Gateway 提示语义漂移。
- 已新增“未找到”共享提示 helper（`NotFoundEN` / `PendingApprovalNotFoundZH`），并批量接入 CLI `tools/toolset`、Gateway `skill/tool`、TUI 审批提示分支，收敛跨入口 not-found 文案来源。
- 已补充 `SkillsDirectoryNotFoundEN` 并接入 Gateway skills 列表 fallback，同时将审批 pattern 分支与 grant pattern usage 残留改为复用 `NotFoundEN`/`UsageEN`，进一步清理 Gateway 内零散提示拼装。
- 已新增 `InvalidActionIndexZH` 与 `WorkflowCommandsEmptyEN`，并接入 TUI `/actions` 与 `/workflow save` 分支，继续收敛分散错误文案到 `internal/tools/command_usage.go`。

范围：

- `internal/cli/`
- `ui-tui/`
- `internal/gateway/`
- `internal/gateway/platforms/`

功能项：

- 建立统一 command dispatcher。
- 将 `/new`、`/reset`、`/model`、`/retry`、`/undo`、`/compress`、`/usage`、`/skills`、`/stop`、`/status`、`/sethome` 归一。
- Gateway 平台命令只做适配，不复制业务逻辑。
- 命令返回统一结构，便于 CLI/TUI/Gateway/Web 渲染。

验收：

- 同一命令在 CLI 与 Telegram/Discord/Slack/WhatsApp/Yuanbao 上结果一致。
- `/stop` 可取消当前平台对应 session 的活动 turn。
- `/model provider:model` 能切换当前会话模型并持久化。

### TODO-003：Gateway 会话连续性与投递目标模型

目标：

- 对齐 Hermes 的 home channel、显式 target、跨平台 session continuity 和 channel directory 能力。

范围：

- `internal/gateway`
- `internal/gateway/platforms`
- `internal/tools/send_message.go`
- `internal/store`

功能项：

- 统一 target 语法：`telegram`、`telegram:<id>`、`discord:<id>`、`slack:<id>`、`whatsapp:<id>`、`yuanbao:<id>`。
- 建立 channel directory，记录平台、channel、用户、home target。
- `send_message` 支持平台默认 home target 与显式 target。
- 跨平台用户身份映射保留可扩展字段。
- Gateway session 与 HTTP/CLI session 互通查询。

验收：

- `send_message(action=list)` 能列出可投递平台和 home target。
- Cron / agent 工具可以投递到 bare platform name 或显式 target。
- 同一用户从不同平台进入时可按配置恢复同一 session。

## P1：补齐 Hermes 主要功能面

### TODO-004：Gateway 平台第一批扩展

目标：

- 补齐 Hermes 文档明确高价值入口：Signal、Email、Webhook、Home Assistant。

范围：

- `internal/gateway/platforms`
- `internal/gateway`
- `internal/config`
- `internal/tools/send_message.go`

功能项：

- Signal：文本 inbound/outbound、附件最小投递、rate-limit 基础处理。
- Email：IMAP/SMTP 或 webhook 风格最小 inbound/outbound。
- Webhook：通用 HTTP inbound，支持 deliver 转发到其他平台。
- Home Assistant：事件 inbound 与 service/action outbound。
- 每个平台接入状态诊断、配置检查、home target、基础权限。

验收：

- `agentd gateway platforms` 包含新增平台。
- 每个平台支持最小文本收发和 `/status`。
- `send_message` 可向新增平台投递。
- 新增平台有最小单元测试或集成假实现测试。

### TODO-005：Gateway 平台第二批扩展

目标：

- 补齐 Hermes 更完整平台矩阵：Matrix、Feishu、DingTalk、WeCom/Weixin、Mattermost、SMS、BlueBubbles。

范围：

- `internal/gateway/platforms`
- `internal/config`
- `internal/tools/send_message.go`

功能项：

- 先实现文本收发、session 映射、home target。
- 第二阶段补媒体、按钮、线程、mention 策略。
- 平台配置统一暴露到 `gateway setup/status/doctor`。

验收：

- 新平台均可独立 enable/disable。
- inbound 消息能触发 agent turn。
- outbound 能通过 `send_message` 和 Cron 投递。

### TODO-006：Gateway 原生交互深度

目标：

- 补更多平台的原生 slash command、按钮审批、mention/free-response 策略、线程/群组策略。

范围：

- `internal/gateway`
- `internal/gateway/platforms`
- `internal/tools/approval_store.go`

功能项：

- 平台原生 `/approve`、`/deny`、`/grant`、`/revoke`、`/pending`。
- 审批按钮或快捷回复。
- mention required / free response channel / ignored channel。
- thread/reply-to 策略。
- group/dm policy。

验收：

- Telegram、Discord、Slack、WhatsApp、Yuanbao 与新增平台的审批命令行为一致。
- 群组中可配置必须 mention 才响应。
- 平台 manifest 或 setup 输出包含原生命令安装信息。

### TODO-007：工具能力级补齐

目标：

- 将已有“工具名对齐”的轻量实现升级成能力级实现。

范围：

- `internal/tools`
- `internal/model`
- `internal/config`

优先级：

1. `browser`：真实页面状态、JS/DOM、截图、表单操作、导航历史。
2. `vision`：图片理解走模型推理，支持本地文件和 URL。
3. `tts`：真实语音合成，支持 provider 配置、音频输出、Gateway 投递。
4. `image_generate`：接真实图像生成后端，支持 prompt、尺寸、输出路径。
5. `transcription`：音频转写，用于语音备忘录入口。

验收：

- 每个工具有 schema、配置项、错误分类、测试。
- Web/TUI/Gateway 能显示工具进度与结果文件。
- 工具失败不会破坏 Agent Loop。

### TODO-008：Toolsets 动态行为

目标：

- 对齐 Hermes toolsets 的 availability check、平台/环境动态过滤、UI 管理和 schema patch。

范围：

- `internal/tools/toolsets.go`
- `internal/tools/registry.go`
- `cmd/agentd`
- `ui-tui`
- `web`

功能项：

- toolset availability check。
- 根据平台凭证、运行环境、enabled/disabled 配置动态过滤工具。
- toolset includes / excludes / conflicts。
- CLI/TUI/Web 管理 toolsets。
- schema patch：按 toolset 或环境裁剪参数。

验收：

- 未配置凭证的平台工具不会暴露给模型。
- `agentd toolsets list/show/resolve` 能解释工具来源和不可用原因。
- TUI/Web 可启停 toolset 并立即影响新 turn。

## P2：补齐系统级完整度

### TODO-009：Provider 生态与 Profile

目标：

- 补 Hermes 支持的主流 provider 与 profile：Nous Portal、OpenRouter、NVIDIA NIM、MiMo、GLM、Kimi、MiniMax、HuggingFace、自定义端点。

范围：

- `internal/model`
- `internal/config`
- `cmd/agentd`
- `web`
- `ui-tui`

功能项：

- provider profile 管理。
- provider capability 描述：tool calling、streaming、vision、image、tts、max context。
- 密钥池与 credential profile。
- provider 流式事件归一。
- provider 失败隔离、熔断、fallback 复用现有能力。

验收：

- `agentd model providers` 可列出 provider、能力和配置状态。
- `agentd model set provider:model` 支持新增 provider。
- TUI/Web 可查看并切换 profile。

### TODO-010：Skills 闭环学习

目标：

- 补复杂任务后自动创建 skill、使用中自我改进、来源/版本/冲突策略、多源 Skills Hub。

范围：

- `internal/tools` skills 相关工具。
- `internal/agent`
- `internal/store`
- `cmd/agentd`
- `web`

功能项：

- 任务完成后根据轨迹建议创建 skill。
- skill provenance：来源、创建时间、触发任务、版本。
- skill 使用统计和效果反馈。
- skill update / audit / snapshot / import / export。
- 多源 tap：GitHub repo、agentskills.io、local marketplace。
- 冲突策略：同名、版本、文件覆盖。

验收：

- 长任务结束后可生成 skill 草稿。
- `/skills` 能显示来源、版本、使用次数。
- skill 修改可审计、可回滚。

### TODO-011：Memory 与用户模型增强

目标：

- 补外部 memory provider、LLM 级摘要质量、用户画像/insights、周期性记忆提醒。

范围：

- `internal/memory`
- `internal/store`
- `internal/agent`
- `cmd/agentd`
- `web`

功能项：

- memory provider 接口。
- `memory status/off/reset`。
- `/insights`：按天数或 session 生成用户偏好、事实、近期主题。
- 周期性 memory nudge：提醒 agent 保存稳定事实。
- 记忆撤销、来源追踪、可信度。

验收：

- 新记忆有来源 session/turn。
- 用户可查看、撤销、禁用外部 memory provider。
- session search 摘要质量可通过 LLM 提升。

### TODO-012：Cron 高级动作与无人值守自动化

目标：

- 补脚本动作、自然语言任务创建、平台投递策略、失败重试、运行审计。

范围：

- `internal/cron`
- `internal/tools/cronjob.go`
- `cmd/agentd`
- `web`
- `internal/gateway`

功能项：

- `no_agent` 脚本动作。
- 自然语言创建 cron job。
- deliver_on、retry、timeout、max_concurrency。
- chained context 增强。
- 运行审计与 replay。
- CLI/Web 管理：create/edit/pause/resume/run/remove/status/tick。

验收：

- 能创建日报、备份、周审计任务并投递到 Gateway。
- 失败任务可重试和查看日志。
- `cronjob` 工具与 CLI 管理状态一致。

### TODO-013：ACP/IDE 完整协议

目标：

- 从最小 API 适配升级到完整能力声明、细粒度事件、鉴权、取消、会话同步。

范围：

- `internal/api`
- `internal/agent`
- `internal/store`

功能项：

- ACP capability declaration。
- session create/list/get/delete。
- message send/stream/cancel/resume。
- tool event / model event / approval event 细分。
- 鉴权与权限边界。

验收：

- IDE 客户端可稳定创建 session、发送消息、流式接收、取消、恢复。
- ACP 事件与内部 `AgentEvent` 可追踪映射。

### TODO-014：Research/RL/trajectory 链路

目标：

- 补 batch trajectory、环境基准、策略评估、轨迹压缩和训练数据导出。

范围：

- `internal/research`
- `cmd/agentd research`
- `scripts`
- `internal/tools`

功能项：

- batch runner：任务集、并发度、失败策略。
- environment benchmark：可插拔任务环境。
- trajectory schema：消息、工具、事件、奖励/结果。
- trajectory compressor 增强。
- stats、export、sample、filter。
- RL/Atropos 兼容导出。

验收：

- `agentd research run/compress/stats/export` 可跑完整闭环。
- 生成 JSONL 轨迹可复放。
- 支持按成功/失败/工具/模型过滤。

## P3：补运维、安装、Web 管理面完整度

### TODO-015：安装、迁移与备份恢复

目标：

- 对齐 Hermes 的 install/setup/update/uninstall/migration 体验。

功能项：

- 完整 setup wizard。
- OpenClaw/Hermes 数据迁移预览与导入。
- backup/export/import/checkpoints。
- update release/channel 管理。
- shell completion 安装。

验收：

- 新用户可通过一个 setup wizard 完成模型、网关、工作区配置。
- 迁移支持 dry-run、preset、overwrite。
- update/backup 操作可回滚。

### TODO-016：Web Dashboard 完整化

目标：

- 将当前 Web Phase 1 从数据页升级为可日常使用的管理后台。

功能项：

- Chat/TUI 同等流式体验。
- Sessions 浏览、重命名、删除、导出。
- Gateway 平台配置、状态、pairing、home channel。
- Cron 可视化管理。
- Skills/Plugins/Tools/Models 完整管理。
- Logs/Diagnostics/Usage/Insights。

验收：

- Web 中可完成主要日常操作，不依赖 CLI。
- Dashboard slot 插件可真实挂载。

## 推荐执行顺序

1. `TODO-001` 完整 TUI/CLI 交互体验。
2. `TODO-002` CLI 与消息平台命令语义统一。
3. `TODO-003` Gateway 会话连续性与投递目标模型。
4. `TODO-004` Gateway 平台第一批扩展。
5. `TODO-005` Gateway 平台第二批扩展。
6. `TODO-006` Gateway 原生交互深度。
7. `TODO-007` 工具能力级补齐。
8. `TODO-008` Toolsets 动态行为。
9. `TODO-009` Provider 生态与 Profile。
10. `TODO-010` Skills 闭环学习。
11. `TODO-011` Memory 与用户模型增强。
12. `TODO-012` Cron 高级动作。
13. `TODO-013` ACP/IDE 完整协议。
14. `TODO-014` Research/RL/trajectory 链路。
15. `TODO-015` 安装、迁移与备份恢复。
16. `TODO-016` Web Dashboard 完整化。

## 每个 TODO 的完成定义

- 功能可从至少一个用户入口实际使用。
- CLI/TUI/Gateway/Web 的交叉影响已检查。
- 有针对性自动化测试，或有明确无法自动化的手工验证步骤。
- `README.md`、`docs/overview-product.md`、`docs/overview-product-dev.md` 根据状态更新。
- `docs/dev/0015-hermes-summary-merged.md` 增加对应总结。
- 单独提交，提交信息标明 TODO 编号。

### 来源：`0000-hermes-agent-go-port.md`

# 001 计划：Hermes Agent Go 版实施计划

## 目标

在 Go 中实现 Hermes 风格 Agent 的完整核心闭环，并同时提供 CLI 与 HTTP API。

## 实施步骤

1. 建立核心共享类型与模型客户端
验证：可向 OpenAI 兼容接口发送消息并解析响应

2. 建立工具注册中心与内置工具
验证：可输出 tool schema，并能按工具名 dispatch

3. 实现 Agent Loop
验证：模型返回 `tool_calls` 时，工具结果可回灌并继续多轮执行

4. 实现会话与记忆持久化
验证：可加载 session 历史，可执行 session_search，可写入 `MEMORY.md` / `USER.md`

5. 实现 CLI 与 HTTP API
验证：CLI 可交互调用；HTTP `/v1/chat` 可返回完整结果

6. 添加关键测试并跑通
验证：`go test ./...` 通过

7. 沉淀调研、设计、总结文档
验证：`docs/` 与 `docs/dev/README.md` 索引完整

### 来源：`0001-hermes-gap-closure.md`

# 002 计划：Hermes 核心闭环差异补齐

## 目标

补齐当前 Go 版与 Hermes 核心闭环之间的剩余关键差异，使 Agent 在跨请求、多轮运行和工具执行安全边界上达到“完整核心功能”状态。

## 实施步骤

1. 重构系统提示词装配
验证：无论是否存在历史消息，每次 `Run()` 都会携带 system message，且不会重复叠加多份 system message

2. 补齐持久记忆回灌
验证：`MEMORY.md` / `USER.md` 的内容会进入系统提示词，后续 session 可直接复用

3. 注入工作区规则
验证：工作目录存在 `AGENTS.md` 时，其内容会以受控方式进入系统提示词

4. 增加文件工具工作区路径约束
验证：`read_file` / `write_file` / `search_files` 只能访问 `Workdir` 内路径，越界访问返回明确错误

5. 增加 terminal 硬阻断护栏
验证：明显灾难性命令会被拒绝执行，正常命令保持兼容

6. 增加针对性测试并回归验证
验证：新增单元测试覆盖提示词装配、记忆回灌、路径约束、危险命令阻断，`go test ./...` 通过

## 模块影响

- `internal/agent`
- `internal/memory`
- `internal/tools`
- `cmd/agentd`
- `docs/`

## 取舍

- 先补“闭环缺口”，不在本次引入完整审批系统与上下文压缩，避免为了追求 1:1 复刻而显著扩大范围
- 安全侧优先实现硬阻断与工作区边界，后续再扩展到审批、URL 安全和更细粒度权限

### 来源：`0052-hermes-feature-alignment.md`

# 053 Plan：Hermes 功能对齐文档完善

## 目标

明确当前 Go 项目与 `/data/source/hermes-agent` 的功能对齐范围，并补齐总览文档中的差异说明。

## 变更范围

- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `README.md`
- `docs/dev/053-research-hermes-feature-alignment.md`
- `docs/dev/053-plan-hermes-feature-alignment.md`
- `docs/dev/053-summary-hermes-feature-alignment.md`
- `docs/dev/README.md`

不修改 Go 源码、不新增依赖、不改变运行行为。

## 执行步骤

1. 梳理 Hermes 和当前项目功能面。
   - 验证：Research 文档列出已对齐、最小覆盖、未覆盖能力。
2. 更新产品总览。
   - 验证：总览明确当前项目是 Hermes 核心 Agent daemon 子集，而非完整复刻。
3. 更新 README 入口说明。
   - 验证：仓库首页能直接看到对齐边界并链接到详细矩阵。
4. 更新开发总览。
   - 验证：开发文档包含模块级功能矩阵和后续补齐建议。
5. 更新需求索引和 Summary。
   - 验证：`docs/dev/README.md` 能追溯到 053 三份文档。

## 不做事项

- 不实现 Hermes 缺失功能。
- 不调整已有配置或源码。
- 不修改当前工作区中已有的非文档变更。

## 验证方式

- 查看 `git diff -- docs README.md`，确认只包含文档补齐。
- 人工复核对齐矩阵与本地源码、Hermes 文档一致。

## 三角色审视

- 高级产品：任务聚焦“分析与文档完善”，没有扩展到功能开发。
- 高级架构师：文档按产品/开发/需求沉淀分层，便于后续需求引用。
- 高级工程师：变更可回滚、无运行时风险，验证成本低。

### 来源：`0061-hermes-cron-alignment.md`

# 062 计划：Hermes Cron 最小对齐（interval/one-shot）

## 目标（可验证）

- `cronjob` 工具可用：`create/list/get/pause/resume/remove/trigger`。
- 开启 `AGENT_CRON_ENABLED=true` 后，调度器会周期性扫描 due job 并触发独立 session 的 agent run。
- cron job 与 run 结果可持久化在 SQLite（与 `sessions.db` 同库）。
- 文档对齐矩阵更新：Cron 从“未覆盖”更新为“部分对齐”，并写明边界。

## 实施步骤

1. **存储**
   - 新增 `cron_jobs`、`cron_runs` 表，复用现有 SQLite 连接。
2. **调度器**
   - ticker 扫描 due jobs；按并发度执行；对 interval/once 计算 next_run_at。
3. **工具**
   - 新增内置工具 `cronjob`，action 压缩 schema。
4. **集成**
   - `serve` 与 `chat` 模式按配置启动 scheduler。
5. **文档**
   - 更新 `README.md`、`docs/overview-product*.md` 与 `docs/dev/README.md` 索引。
6. **测试**
   - schedule 解析与 cron store CRUD 单测（无网络/无端口依赖）。

## 不在本次范围

- cron 表达式执行
- 平台投递与 origin 捕获
- prompt threat scanning
- `no_agent` 脚本作业、context_from 链式作业

### 来源：`0062-hermes-toolsets-alignment.md`

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

### 来源：`0063-hermes-send-message-alignment.md`

# 064 计划：Hermes send_message 最小对齐

## 目标（可验证）

- `send_message(action='list')` 返回当前进程已连接的 gateway adapters。
- `send_message(action='send', platform, chat_id, message)` 可投递文本消息。
- Gateway runner 会在 adapter connect/disconnect 时注册/注销 adapter。
- 文档对齐矩阵更新：Gateway/toolsets 标记调整，补 docs/dev 索引。

## 实施步骤

1. 解耦 adapter 接口到 `internal/platform`。
2. 新增运行时 adapter registry。
3. Gateway runner hook：connect 后 register，退出前 unregister。
4. 新增工具 `send_message` 并注册到 engine。
5. 更新 toolsets `messaging` + `core` includes。
6. 补单测与文档。

### 来源：`0064-hermes-patch-tool-alignment.md`

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

### 来源：`0065-hermes-web-tools-alignment.md`

# 066 计划：Hermes web tools 最小对齐

## 目标（可验证）

- 新增内置工具：`web_search`、`web_extract`。
- `toolsets.web` 默认包含 `web_search/web_extract`（保留 `web_fetch` 兼容）。
- 单测覆盖：DDG 结果解析与 HTML->text 抽取基础行为。

## 实施步骤

1. 在 builtin tools 中注册并实现 `web_search/web_extract`。
2. 新增最小解析与清洗 helper。
3. 更新 toolsets/web。
4. 更新 docs 与 `docs/dev/README.md` 索引。

### 来源：`0066-hermes-clarify-tool-alignment.md`

# 067 计划：Hermes clarify 工具最小对齐

## 目标（可验证）

- 新增内置工具 `clarify`，并纳入 `toolsets.core`。
- `clarify` 对空 question 报错；对 options 做最小校验与清洗。
- 更新 docs/dev 索引与工具清单。

## 实施步骤

1. 在 builtin tools 中注册并实现 `clarify`。
2. 在 toolsets 中新增 `clarify` toolset，并让 core includes 它。
3. 更新文档与索引。

### 来源：`0067-hermes-execute-code-alignment.md`

# 068 计划：Hermes execute_code 最小对齐

## 目标（可验证）

- 新增工具 `execute_code`，可运行 python 片段并返回 stdout/stderr/exit code。
- 限制在 workdir 内执行，支持超时。
- 单测覆盖基础执行成功路径。

## 实施步骤

1. 新增 `internal/tools/execute_code.go` 并注册到 engine。
2. toolsets 增加 `code_execution`（默认不纳入 core）。
3. 更新 docs/dev 索引与工具清单。
