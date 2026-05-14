# 0035 skills research merged

## 模块

- `skills`

## 类型

- `research`

## 合并来源

- `0044-skills-research-merged.md`

## 合并内容

### 来源：`0044-skills-research-merged.md`

# 0044 skills research merged

## 模块

- `skills`

## 类型

- `research`

## 合并来源

- `0007-skills-minimal-skeleton.md`
- `0044-skills-adv-trigger-sync.md`

## 合并内容

### 来源：`0007-skills-minimal-skeleton.md`

# 008 调研：Skills 最小骨架补齐

## 背景

当前项目已经具备工具注册中心、MCP 代理和多 provider，但仍缺少本地技能系统入口。

Hermes 的 Skills 体系较完整（发现、调用、管理、更新），本阶段先补齐最小可用能力：本地技能发现与查看。

## 差异点

- 当前无技能目录发现能力
- 当前无技能查看工具
- 当前无技能内容注入入口

## 本次范围

- 新增 `skill_list`：列出本地 `skills/*/SKILL.md`
- 新增 `skill_view`：按技能名读取 `SKILL.md`
- 做好路径安全限制，禁止路径穿越

不纳入：

- `skill_manage` 创建/编辑
- skill 执行脚本与依赖管理
- skills hub 同步

## 结论

该骨架能够打通“技能可见性”闭环，为后续补齐技能管理与自动技能调用提供基础。

### 来源：`0044-skills-adv-trigger-sync.md`

# 045 调研：技能高级能力（自动触发 + 同步）

## 任务背景

当前 agent-daemon 的 Skills 系统实现了基础骨架（`skill_list`/`skill_view`/`skill_manage`），但 LLM 无法自动发现技能目录中的技能，也无法从外部源同步技能。需要补齐两个核心缺口：

1. **自动触发**：运行时将技能目录索引注入 system prompt，使 LLM 自动发现并加载相关技能
2. **技能同步**：支持从外部源（如 GitHub 仓库）下载技能到本地目录

## Hermes 参考

### 自动触发机制

Hermes 使用三层渐进式揭露：

| 层 | 机制 | 触发条件 |
|----|------|---------|
| 1. Skill Index | `build_skills_system_prompt()` 每轮注入技能目录 | 自动，每次模型调用前 |
| 2. 条件过滤 | `required_tools`/`fallback_for_tools` 按工具可用性过滤 | 静态配置 |
| 3. 显式加载 | CLI `--skills`、Slash `/skill-name`、LLM 调用 `skill_view()` | 用户或 LLM 触发 |

第 1 层是核心：系统提示词中注入强指令 + 精简技能目录，LLM 自行判断何时调用 `skill_view()`。

### 技能同步机制

Hermes 的 Skills Hub 是联邦式多源适配器架构（~10 个 source adapter），对 agent-daemon 过于庞大。最小落地方案建议仅支持 **GitHub 仓库** 和 **直接 URL** 两种源。

## 推荐方案

### 范围（L3：跨模块，但不引入新基础设施）

**本期实现**：
1. **Skill Index 注入**：`buildRuntimeSystemPrompt` 中扫描 `<workdir>/skills/` 目录，生成精简技能目录注入 system prompt
2. **`skill_manage sync` 动作**：支持从 GitHub 仓库下载技能到本地

**不纳入**：
- 条件过滤（`required_tools`/`fallback_for_tools`）
- 多源适配器（HermesIndex、skills.sh、ClawHub 等）
- 技能安全扫描（skills_guard）
- 技能缓存清单（bundled_manifest）
- 技能预加载 CLI 参数

### 技能索引格式

```
## Available Skills
Before each task, check if any skill below matches. If relevant, load it with skill_view(name).
- sample-skill: A brief description of what this skill does
- git-workflow: Git branch, commit, PR workflow helper
- api-testing: REST API testing with curl and jq
```

### 技能同步：GitHub 仓库

```
skill_manage action=sync source=github repo=<owner/name> path=<subdir>
```

流程：
1. 请求 `https://api.github.com/repos/{owner}/{name}/contents/{path}`
2. 遍历目录，对每个包含 `SKILL.md` 的子目录下载文件
3. 写入本地 `<workdir>/skills/<skill-name>/`

### 技能同步：直接 URL

```
skill_manage action=sync source=url url=<skill-url> name=<name>
```

流程：
1. HTTP GET 获取原始 SKILL.md 内容
2. 写入 `<workdir>/skills/<name>/SKILL.md`

## 修改文件清单

| 文件 | 变更 |
|------|------|
| `internal/agent/system_prompt.go` | 新增 `buildSkillsIndexBlock()`，`buildRuntimeSystemPrompt` 中调用 |
| `internal/tools/builtin.go` | `skill_manage` 新增 `sync` 动作 |
| `internal/tools/builtin_test.go` | 新增技能索引 + 同步测试 |
| `internal/agent/system_prompt_test.go` | 新增系统提示词技能注入测试 |

## 技术风险

| 风险 | 等级 | 缓解 |
|------|------|------|
| 技能目录过大导致 prompt 膨胀 | 低 | 限制索引条目数（默认 50），仅包含 name + description |
| GitHub API 限流 | 低 | 单次操作，不频繁触发 |
| 网络超时 | 低 | 30s timeout，`context.Context` 传播 |

## 结论

技能索引注入是 Skill 闭环的关键缺失环节，实现后 LLM 真正拥有自动发现并加载技能的能力。同步功能提供最小可用的技能获取渠道。
