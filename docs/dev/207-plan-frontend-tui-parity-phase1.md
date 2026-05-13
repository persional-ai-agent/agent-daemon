# Frontend 与 TUI 对齐计划（Phase 1）

## 任务拆解

1. 增强 CLI 交互命令面  
验证：`internal/cli` 单测通过，`/help` 等命令具备稳定输出。

2. 新建 `web/` Vite + React 工程骨架  
验证：目录结构完整，`npm run dev` 可启动，Chat 页可请求 `/v1/chat`。

3. 同步产品/开发文档  
验证：`docs/overview-product.md`、`docs/overview-product-dev.md` 有前端/TUI 章节并与实现一致。

4. 更新 `docs/dev/README.md` 索引并形成阶段总结  
验证：索引可检索到 207 系列文档。

## 风险与应对

1. 仓库当前无 Node 构建流水线  
应对：本阶段只落地工程与运行说明，不强行将 Node 构建纳入 Go 测试流水线。

2. “一次性彻底对齐”范围过大  
应对：采用多批次迭代，每批可运行、可验证、可提交。
