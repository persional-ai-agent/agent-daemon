# 066 计划：Hermes web tools 最小对齐

## 目标（可验证）

- 新增内置工具：`web_search`、`web_extract`。
- `toolsets.web` 默认包含 `web_search/web_extract`（保留 `web_fetch` 兼容）。
- 单测覆盖：DDG 结果解析与 HTML->text 抽取基础行为。

## 实施步骤

1. 在 builtin tools 中注册并实现 `web_search/web_extract`。
2. 新增最小解析与清洗 helper。
3. 更新 toolsets/web。
4. 更新 docs 与 `docs/dev/README.md` 索引。

