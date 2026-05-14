# 0035 skills summary merged

## 模块

- `skills`

## 类型

- `summary`

## 合并来源

- `0044-skills-summary-merged.md`

## 合并内容

### 来源：`0044-skills-summary-merged.md`

# 0044 skills summary merged

## 模块

- `skills`

## 类型

- `summary`

## 合并来源

- `0007-skills-minimal-skeleton.md`
- `0044-skills-adv-trigger-sync.md`
- `0050-skills-filter-preload.md`

## 合并内容

### 来源：`0007-skills-minimal-skeleton.md`

# 008 总结：Skills 最小骨架补齐结果

## 已完成

- 在内置工具中新增：
  - `skill_list`
  - `skill_view`
- 默认技能目录：`<workdir>/skills`
- 支持按工具参数覆盖目录（仍受工作区路径约束）
- 新增路径安全校验：技能名禁止路径穿越与目录分隔符
- 新增测试覆盖技能列出、技能查看、非法名称校验

## 验证

- `go test ./...` 通过

## 当前边界

本次为技能最小骨架，仍未覆盖：

- `skill_manage`（创建/编辑/删除）
- skills hub 同步
- 自动技能触发与策略评分

### 来源：`0044-skills-adv-trigger-sync.md`

# 045 总结：技能高级能力（自动触发 + 同步）

## 变更摘要

1. **技能索引自动注入**：每次运行时扫描 `<workdir>/skills/`，在 system prompt 中注入精简技能目录，LLM 可自动发现并调用 `skill_view()` 加载相关技能
2. **`skill_manage sync` 动作**：支持从 URL 或 GitHub 仓库同步技能到本地

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/agent/system_prompt.go` | 新增 `buildSkillsIndexBlock()` + `readSkillDescription()`；`buildRuntimeSystemPrompt` 中追加调用 |
| `internal/tools/builtin.go` | `skill_manage` 新增 `sync` 动作（`source=url`/`source=github`）；新增 `fetchHTTPBytes`/`syncGitHubSkill` 辅助函数；schema 新增 `source`/`url`/`repo` 参数 |
| `internal/agent/system_prompt_test.go` | 新建：4 个技能索引注入测试 |
| `internal/tools/builtin_test.go` | 新增 3 个 sync 测试（URL sync、缺少 source、错误 source） |

## 新增能力

### 技能索引自动注入

system prompt 中追加的格式：

```
## Available Skills
Before each task, scan the skills below. If a skill is relevant, load it with skill_view(name) and follow its instructions.
- test-skill: A test skill for unit testing
- another-skill: Another useful skill
```

- 自动扫描 `<workdir>/skills/*/SKILL.md`
- 限制最大 50 个条目
- 描述截取首行并限制 120 字符
- 空目录/无 SKILL.md 时不注入

### `skill_manage sync`

| 参数 | 说明 |
|------|------|
| `action=sync` | 固定值 |
| `source=url` | URL 源：直接 HTTP GET 下载 SKILL.md |
| `source=github` | GitHub 源：通过 Contents API 遍历仓库子目录 |
| `url` | 用于 `source=url` 的原始文件 URL |
| `repo` | 用于 `source=github` 的仓库标识（`owner/name`） |
| `path` | 用于 `source=github` 的仓库子路径 |
| `name` | 本地技能名称 |

## 测试结果

```
go test ./... -count=1 全部通过
go vet ./... 无警告
```

## 后续扩展建议

- 条件过滤（`required_tools`/`fallback_for_tools`）：按工具可用性过滤技能索引
- 多源适配器：skills.sh、ClawHub 等
- 技能安全扫描
- CLI `--skills` 预加载参数

### 来源：`0050-skills-filter-preload.md`

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
