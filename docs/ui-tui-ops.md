# ui-tui 运维手册

## 1. 常见故障

- 网络错误（`code=network`）：
  - 检查 `ws_base` / `http_base` 是否可达。
  - 检查代理、防火墙、SSH 隧道或端口映射。
- 超时（`code=timeout`）：
  - 调大 `[ui-tui] ws_read_timeout_seconds` 与 `ws_turn_timeout_seconds`。
  - 检查后端模型响应速度与工具执行时长。
- 鉴权失败（`code=auth`）：
  - 检查后端 provider token 与网关 token 配置。
  - 重新执行 `agentd doctor` / `agentd config get` 核验密钥来源。
- 请求错误（`code=request`）：
  - 检查命令参数格式，如 `/events save` 的时间格式必须是 RFC3339。

## 2. 审批相关

- 查看待审批：
  - `/pending`：最近一条
  - `/pending 5`：最近 5 条
- 审批操作：
  - `/approve` / `/deny`：默认处理最近待审批
  - `/approve <approval_id>` / `/deny <approval_id>`：处理指定项
- 若提示找不到待审批：
  - 用 `/show <session_id> 0 200` 检查当前会话内是否存在 `pending_approval` 记录。
- 若 `/approve` 或 `/deny` 返回 `404 page not found`：
  - 说明当前运行的后端版本未启用 `POST /v1/ui/approval/confirm`。
  - 先升级后端到最新版本后重试。
- 若 `/pending` 返回 `http 500` 且包含 `converting NULL to int64`：
  - 属于历史数据兼容问题（会话统计字段存在旧数据空值）。
  - 建议先升级后端并执行一次会话存储修复/迁移，再重试该命令。

## 3. 配置重载

- 配置文件：`config/config.ini` 的 `[ui-tui]` 段。
- 运行时重载：`/reload-config`。
- 重载后可用 `/status` 与提示符确认当前状态。

## 4. 状态文件修复

- 状态文件位置：`~/.agent-daemon/ui-tui-state.json`。
- 若文件损坏，ui-tui 启动时会自动：
  - 备份原文件到 `ui-tui-state.json.corrupt.<timestamp>`
  - 重建默认状态文件

## 5. 自检命令

- 全量测试：`go test ./...`
- ui-tui 烟测：`./ui-tui/e2e_smoke.sh`
