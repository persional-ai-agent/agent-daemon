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
