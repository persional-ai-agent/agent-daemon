# 066 总结：Hermes web tools 最小对齐

## 完成情况

- 新增 `web_search`（DDG HTML 抓取 + 最小解析）与 `web_extract`（最小 HTML->text 清洗 + 截断）。
- toolsets `web` 默认调整为 `web_search/web_extract`。
- 保留 `web_fetch` 作为兼容与调试用途。

## 边界

- `web_search` 依赖第三方 HTML 页面结构，可能随时间变化；必要时可通过 `base_url` 指向自托管/替代实现。

