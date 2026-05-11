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
