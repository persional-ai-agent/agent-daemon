# 0045 web summary merged

## 模块

- `web`

## 类型

- `summary`

## 合并来源

- `0056-web-summary-merged.md`

## 合并内容

### 来源：`0056-web-summary-merged.md`

# 0056 web summary merged

## 模块

- `web`

## 类型

- `summary`

## 合并来源

- `0246-web-reconnect-controls.md`
- `0247-web-ws-primary-fallback-sse.md`
- `0248-web-realtime-diagnostics-panel.md`
- `0277-web-dashboard-functional-pages.md`
- `0278-web-cron-management.md`
- `0279-web-model-provider-management.md`

## 合并内容

### 来源：`0246-web-reconnect-controls.md`

# Web Chat 重连状态可视化与控制对齐

本轮将 TUI 侧的重连语义同步到 Web Chat，提升前端实时会话可用性。

## 主要改动

- `web/src/lib/api.ts`
  - 新增统一错误归一化：`normalizeAPIError`
  - 新增流式重连控制参数：
    - `reconnectEnabled`
    - `maxReconnect`
    - `readTimeoutMs`
    - `turnTimeoutMs`
    - `timeoutAction`（`wait/reconnect/cancel`）
  - `streamChat` 支持 `resume/turn_id`、重连状态回调与超时动作
  - 新增事件去重键：`streamEventDedupeKey`
- `web/src/App.tsx`
  - Chat 页新增连接状态条：`connecting/resumed/degraded/failed`
  - 新增重连控制面板：
    - 自动重连开关
    - 最大重连次数
    - 读超时/轮次超时
    - 超时策略选择
    - 手动重连按钮
  - 流式事件渲染去重，避免重连后重复事件渲染
  - 错误展示统一读取 `error_code/error/error_detail/reason`
- `web/src/styles.css`
  - 增加连接状态样式与控制区样式
- `web/src/lib/api.test.ts`
  - 增加错误归一化与事件去重键回归测试
- `web/package.json`
  - 新增 `test` 脚本（vitest）
  - 增加 `vitest` dev dependency

## 验证

- `npm --prefix web run test`
- `npm --prefix web run build`
- `make contract-check`
- `go test ./...`

### 来源：`0247-web-ws-primary-fallback-sse.md`

# Web Chat：WS 主通道 + SSE 降级

本轮将 Web Chat 流式通道升级为 WS 主通道，并保留 SSE 降级，统一重连状态与错误语义。

## 主要改动

- `web/src/lib/api.ts`
  - `streamChat` 支持 `transport=ws|sse` 与 `fallbackToSSE`
  - 新增 WS 实现：
    - 发送 `session_id/message/turn_id/resume`
    - 状态回调：`connecting/resumed/degraded/failed`
    - 超时策略：`wait/reconnect/cancel`
  - 保留并复用 SSE 流式逻辑（降级）
  - 新增错误归一化 `normalizeAPIError`
  - 新增 `streamEventDedupeKey`
- `web/src/App.tsx`
  - Chat 页新增传输模式切换（WS/SSE）
  - 连接状态条显示 `streamStatus + transport`
  - 事件去重渲染，避免重连重复展示
  - 错误展示统一读取归一化结果
- `web/src/styles.css`
  - 优化控制区与状态条样式，适配多控件布局
- `web/src/lib/api.test.ts`
  - 新增/更新回归测试（错误归一化、去重键稳定性）
- `web/package.json`
  - 保持 `vitest` 测试入口可用

## 验证

- `npm --prefix web run test`
- `npm --prefix web run build`
- `make contract-check`
- `go test ./...`

### 来源：`0248-web-realtime-diagnostics-panel.md`

# Web Chat：实时诊断面板与诊断包导出

本轮在 Web Chat 页补齐实时诊断可观测能力，覆盖传输状态、降级提示、重连统计与诊断包导出。

## 主要改动

- `web/src/lib/api.ts`
  - `StreamChatOptions` 新增 `onTransport` 回调，暴露当前实际传输通道（`ws|sse`）。
  - 新增 `createTransportFallbackEvent`，统一构造 `transport_fallback` 事件（`from/to/reason/at`）。
  - `streamChat` 在 WS 失败并降级 SSE 时：
    - 回调 `onTransport("sse")`
    - 发出 `transport_fallback` 事件，供上层 UI 诊断展示。
- `web/src/App.tsx`
  - 新增实时诊断状态：`activeTransport`、`lastTurnID`、`reconnectCount`、`lastErrorCode`、`fallbackHint`。
  - Chat 流式发送中接入 `onTransport` 与 `transport_fallback` 事件，展示降级原因与时间。
  - 新增“实时诊断”面板，实时展示核心运行字段。
  - 新增“导出诊断包”按钮，导出当前诊断上下文 JSON。
- `web/src/styles.css`
  - 新增降级提示条 `fallback-note` 样式。
  - 优化诊断面板 `pre` 容器可读性样式。
- `web/src/lib/api.test.ts`
  - 新增 `createTransportFallbackEvent` 契约测试，固定字段形状与枚举语义。

## 验证

- `npm --prefix web run test`
- `npm --prefix web run build`
- `make contract-check`
- `go test ./...`

### 来源：`0277-web-dashboard-functional-pages.md`

# 277 总结：Web Dashboard 功能页补齐

## 背景

用户要求按差异清单逐步实现，并强调“先做功能”。本次选择 Web Dashboard 作为第一批功能补齐对象，优先复用现有 `/v1/ui/*` 后端接口，不新增后端协议。

## 完成内容

- `web/src/lib/api.ts`
  - 新增 Skills、Agents、Plugins、Gateway diagnostics、Voice 相关 UI API 封装。
- `web/src/App.tsx`
  - 新增 `skills` 页面：列表、详情、创建、编辑、删除、搜索、同步、reload。
  - 新增 `agents` 页面：delegate 列表、active、history、详情查询、中断。
  - 新增 `plugins` 页面：展示 plugin dashboard slot。
  - 新增 `voice` 页面：voice 开关、TTS 开关、录音状态、TTS 请求。
  - Gateway 页面补充 diagnostics 展示。
- `web/src/styles.css`
  - 补齐新增页面所需的响应式布局和编辑器样式。
- 文档同步
  - 更新 Web README、Frontend/TUI 用户与开发文档、产品/开发总览。

## 验证

- `npm run test`
- `npm run build`

## 边界

本次只把现有后端能力接到 Web Dashboard。尚未补齐 Hermes dashboard 的 provider/auth、profiles、cron、logs、analytics、PTY、主题与远程插件管理等能力。

### 来源：`0278-web-cron-management.md`

# 278 总结：Web Cron 管理面补齐

## 背景

延续“先做功能”的原则，在 Web Dashboard 已接入基础管理页后，继续补齐 Cron 任务管理能力。后端已有 `cronjob` 工具与 `CronStore`，本次不复制调度逻辑，只为 UI 增加薄 API 封装。

## 完成内容

- `internal/api/server.go`
  - 新增 `GET /v1/ui/cron/jobs`：列出 Cron jobs。
  - 新增 `POST /v1/ui/cron/jobs`：创建 Cron job。
  - 新增 `GET /v1/ui/cron/jobs/{job_id}`：查看单个 job。
  - 新增 `POST /v1/ui/cron/jobs/action`：执行 `pause/resume/trigger/remove/runs/run_get/update` 等操作。
  - 所有操作底层复用 `cronjob` 工具分发。
- `web/src/lib/api.ts`
  - 新增 Cron UI API client。
- `web/src/App.tsx`
  - 新增 `cron` 页面，支持创建、列表、详情、暂停、恢复、触发、删除与运行记录查看。
- 文档同步
  - 更新 Web README、Frontend/TUI 用户与开发文档、产品/开发总览。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/api`
- `npm run test`
- `npm run build`

## 边界

本次补齐的是管理面和 API 封装。后续 `280` 已补齐 cron expression 调度执行，`281` 已补齐运行结果投递；链式上下文和更完整运行详情仍是后续功能。

### 来源：`0279-web-model-provider-management.md`

# 279 总结：Web Model / Provider 管理面补齐

## 背景

延续“先做功能”的原则，在 Web Dashboard 已具备基础管理、Cron 管理后，补齐模型/Provider 管理入口，让用户可以不通过 CLI 直接查看与切换模型配置。

## 完成内容

- `internal/api/server.go`
  - 新增 `GET /v1/ui/model`：当前 provider/model/base_url 与模型运行配置。
  - 新增 `GET /v1/ui/model/providers`：可用 provider 列表，包含内置 provider 与 provider 插件。
  - 新增 `POST /v1/ui/model/set`：写入 provider/model/base_url。
- `cmd/agentd/main.go`
  - 在 `runServe` 注入模型管理回调，复用 CLI 已有的 provider 发现、校验与保存逻辑。
- `web/src/lib/api.ts`
  - 新增模型管理 API client。
- `web/src/App.tsx`
  - 新增 `models` 页面，支持当前模型查看、provider 列表展示与模型切换。
- 文档同步
  - 更新 Web README、Frontend/TUI 用户与开发文档、产品/开发总览。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/api ./cmd/agentd`
- `npm run test`
- `npm run build`

## 边界

本次补的是模型选择与 provider 列表管理面；OAuth 登录、API key 安全录入、provider profile、账号用量与 provider routing 仍是后续功能。
