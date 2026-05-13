# 后端兼容修复：SessionStats 空值与审批确认接口检查（Phase 15）

本轮修复了真实环境回归中暴露的后端兼容问题，并补充了升级检查说明。

## 修复内容

- `SessionStats` 空值兼容：
  - 对 `SUM/MIN/MAX` 聚合结果增加 `COALESCE`，避免旧数据场景下出现 `NULL -> int64/string` 扫描报错。
  - 直接修复了 `/v1/ui/sessions/{id}` 在特定历史数据上的 500 风险（`converting NULL to int64`）。
- 回归测试补齐：
  - 空会话统计返回 0/空字符串
  - 历史 `tool_calls_json=NULL` 行统计兼容
- 升级说明：
  - 在 `docs/ui-tui-ops.md` 增加 `POST /v1/ui/approval/confirm` 的接口探针检查，帮助识别旧后端导致的 `/approve` `/deny` 404。

## 验证

- `go test ./...`：通过
