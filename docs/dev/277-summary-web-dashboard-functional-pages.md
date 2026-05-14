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
