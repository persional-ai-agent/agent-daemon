# Web Dashboard (Phase 1)

## 运行

```bash
cd web
npm install
npm run dev
```

默认连接 `http://127.0.0.1:8080` 的 agent API。

## 当前范围

- Chat 页面：调用 `/v1/chat` 与 `/v1/chat/cancel`
- Chat 页面支持流式模式：调用 `/v1/chat/stream` 并展示事件时间线
- Sessions 页面：调用 `/v1/ui/sessions`，支持会话详情分页。
- Tools 页面：调用 `/v1/ui/tools`，支持工具筛选与 schema 查看。
- Skills 页面：调用 `/v1/ui/skills`，支持列表、查看、创建、编辑、删除、搜索、同步与 reload。
- Agents 页面：调用 `/v1/ui/agents*`，支持 delegate/active/history 查看、详情查询与中断。
- Cron 页面：调用 `/v1/ui/cron/jobs*`，支持任务创建、可选结果投递目标、链式上下文模式、列表、详情、暂停、恢复、触发、删除与运行记录查看。
- Models 页面：调用 `/v1/ui/model*`，支持当前模型查看、provider 列表与 provider/model/base_url 切换。
- Plugins 页面：调用 `/v1/ui/plugins/dashboards`，展示插件 dashboard slot。
- Gateway 页面：调用 `/v1/ui/gateway/status` 与 `/v1/ui/gateway/diagnostics`。
- Voice 页面：调用 `/v1/ui/voice/*`，支持开关、录音状态与 TTS 操作。
- Config 页面：调用 `/v1/ui/config`。

## 烟测与诊断样本

```bash
ARTIFACTS_DIR=./artifacts ./web/e2e_smoke.sh
```

- 执行 web test/build
- 生成 `diag.v1` 样本：`artifacts/diag-web.sample.json`
