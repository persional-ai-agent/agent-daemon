# 051 总结：技能条件过滤 + 预加载

## 变更摘要

1. **条件过滤**：SKILL.md 支持 YAML 前导元数据（`requires_tools`/`fallback_for_tools`），按可用工具集自动过滤技能索引
2. **预加载**：CLI `--skills s1,s2` 参数，启动时将指定技能完整内容注入 system prompt

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/agent/system_prompt.go` | `buildRuntimeSystemPrompt` 接受 `*tools.Registry`；`buildSkillsIndexBlock` 接受 registry 并解析 YAML frontmatter；新增 `parseSkillFrontmatter`/`skillShouldShow`/`buildToolNameSet` |
| `internal/agent/loop.go` | `Run()` 传递 `e.Registry` 给 system prompt builder |
| `internal/cli/chat.go` | `RunChat` 新增 `preloadSkills` 参数；新增 `buildPreloadedSkillsBlock` |
| `cmd/agentd/main.go` | `chat` 子命令新增 `--skills` flag |
| `internal/agent/system_prompt_test.go` | 新增 2 个过滤测试 |
| `go.mod` | 新增 `gopkg.in/yaml.v3` |

## 新增能力

### YAML 前导元数据

```markdown
---
requires_tools: [terminal, write_file]
fallback_for_tools: [web_search]
---
# Skill Title
...
```

- `requires_tools`：仅当列出的所有工具都可用时显示此技能
- `fallback_for_tools`：当任一工具可用时隐藏此技能（作为备选方案）

### CLI 预加载

```bash
agentd chat --message "review PR" --skills github-code-review,kanban
```

预加载的技能内容以 `[IMPORTANT: ...]` 块注入 system prompt，包含完整 SKILL.md 内容。

## 测试结果

`go build ./...` ✅ | `go test ./...` 全部通过 ✅ | `go vet ./...` 无警告 ✅
