# 073 总结：web 与 session_search 工具返回补齐 success

为对齐 Hermes 常见返回格式并便于上层统一处理，本次为以下工具返回增加 `success=true`：

- `session_search`
- `web_fetch`
- `web_search`
- `web_extract`

不移除原字段，保持兼容。

