# 275 总结：Memory / 学习闭环补齐

## 背景

用户要求一次性完成 Hermes 差异清单中的第 6 项：Memory / 学习闭环。目标不是接入外部记忆插件生态，而是在现有 Go 版内完成可验证的会话召回、摘要沉淀、记忆抽取与提示词主动使用闭环。

## 实现结果

- `internal/store/session_store.go` 新增 `session_summaries` 表，`session_search` 会按会话聚合返回摘要、关键词、事实、匹配数、最近时间和高亮信息。
- 搜索支持 FTS5 查询并在不可用或查询不适配时回退 LIKE；空 query 会返回最近会话摘要，用于主动浏览历史上下文。
- `internal/memory/store.go` 新增 `extract` 动作，可从对话文本抽取稳定偏好/项目事实，跳过重复项并过滤明显敏感信息。
- `memory` 工具 schema 增加 `extract`，`session_search` schema 允许空 query；工具返回 `mode`、`count` 等结构化字段。
- 默认系统提示词新增主动召回和主动记忆规则，要求先使用 `session_search` 召回，再把稳定偏好/事实沉淀到 memory，避免保存密钥或一次性信息。

## 边界

- 摘要与事实抽取采用本地确定性规则，避免引入模型调用依赖；相比 Hermes 的 LLM 摘要，质量仍偏保守。
- 当前记忆仍落在 `MEMORY.md` / `USER.md`，尚未接入 Hermes 的外部 memory provider 插件生态。
- `memory.extract` 只沉淀明显稳定的偏好/事实，避免把临时任务、凭证、密钥写入长期记忆。

## 验证

```bash
GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./internal/store ./internal/memory ./internal/tools ./internal/agent
GOCACHE=/tmp/agent-daemon-gocache GOMODCACHE=/tmp/agent-daemon-gomodcache go test ./...
```

结果：通过。
