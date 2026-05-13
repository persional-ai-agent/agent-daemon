# ui-tui 增加后端能力预检命令（Phase 16）

新增 `ui-tui` 命令 `/doctor`，用于在交互期快速识别“后端接口版本不匹配/连通性异常”问题。

## 覆盖检查项

- `health`：`GET /health`
- `sessions_detail`：`GET /v1/ui/sessions/{session_id}`
- `approval_confirm`：`POST /v1/ui/approval/confirm`
  - `404` 明确提示后端版本过旧
  - `200/400` 视为接口已存在
- `config_effective`：展示当前生效配置（ws/http/重连/超时/上限）
- `ws_reachable`：WebSocket 握手检查

## 回归

- 新增 `TestRunDoctor` 覆盖 `/doctor` 主链路。
- `ui-tui/e2e_smoke.sh` 已纳入 `/doctor` 执行。

## 验证

- `go test ./...`：通过
- `./ui-tui/e2e_smoke.sh`：通过
