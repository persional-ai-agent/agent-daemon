# 066 调研：Hermes web_search/web_extract 与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 的核心 web 工具是：

- `web_search`：搜索网页结果
- `web_extract`：抓取并提取网页正文（可读文本）

用于 research 场景，通常比单纯 `web_fetch` 更省上下文。

## 当前项目差异

Go 版此前只有 `web_fetch`（返回原始内容），缺少搜索与正文抽取。

## 最小对齐目标（本次）

- 新增 `web_search`：使用 DuckDuckGo HTML 页面抓取并解析结果链接（可通过 `base_url` 覆盖，便于测试/自托管）。
- 新增 `web_extract`：抓取并做最小 HTML->text 清洗，返回可读文本并支持 `max_chars` 截断。
- toolsets/web 由 `web_fetch` 调整为 `web_search+web_extract`（保留 `web_fetch` 兼容）。

## 边界

- 解析策略是最小实现（regex/清洗），不保证对所有站点完美。
- 没有高级抓取（JS 渲染、反爬、阅读模式、结构化提取）。

