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
