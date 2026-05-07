# 001 计划：Hermes Agent Go 版实施计划

## 目标

在 Go 中实现 Hermes 风格 Agent 的完整核心闭环，并同时提供 CLI 与 HTTP API。

## 实施步骤

1. 建立核心共享类型与模型客户端
验证：可向 OpenAI 兼容接口发送消息并解析响应

2. 建立工具注册中心与内置工具
验证：可输出 tool schema，并能按工具名 dispatch

3. 实现 Agent Loop
验证：模型返回 `tool_calls` 时，工具结果可回灌并继续多轮执行

4. 实现会话与记忆持久化
验证：可加载 session 历史，可执行 session_search，可写入 `MEMORY.md` / `USER.md`

5. 实现 CLI 与 HTTP API
验证：CLI 可交互调用；HTTP `/v1/chat` 可返回完整结果

6. 添加关键测试并跑通
验证：`go test ./...` 通过

7. 沉淀调研、设计、总结文档
验证：`docs/` 与 `docs/dev/README.md` 索引完整
