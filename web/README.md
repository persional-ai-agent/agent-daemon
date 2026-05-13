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
- Sessions 页面：调用 `/v1/ui/sessions`
- Tools 页面：调用 `/v1/ui/tools`
- Gateway 页面：调用 `/v1/ui/gateway/status`
- Config 页面：调用 `/v1/ui/config`

## 烟测与诊断样本

```bash
ARTIFACTS_DIR=./artifacts ./web/e2e_smoke.sh
```

- 执行 web test/build
- 生成 `diag.v1` 样本：`artifacts/diag-web.sample.json`
