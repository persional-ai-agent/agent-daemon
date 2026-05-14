# 0035 skills plan merged

## 模块

- `skills`

## 类型

- `plan`

## 合并来源

- `0044-skills-plan-merged.md`

## 合并内容

### 来源：`0044-skills-plan-merged.md`

# 0044 skills plan merged

## 模块

- `skills`

## 类型

- `plan`

## 合并来源

- `0007-skills-minimal-skeleton.md`
- `0044-skills-adv-trigger-sync.md`

## 合并内容

### 来源：`0007-skills-minimal-skeleton.md`

# 008 计划：Skills 最小骨架补齐

## 目标

补齐本地技能最小入口能力，让 Agent 可发现并读取技能说明。

## 实施步骤

1. 新增技能工具注册
验证：工具列表包含 `skill_list` 与 `skill_view`

2. 实现技能目录扫描
验证：可列出 `skills/<name>/SKILL.md`，并返回名称与简介

3. 实现技能查看
验证：按技能名读取技能全文；非法名称会被拒绝

4. 增加测试并回归
验证：技能工具测试通过，`go test ./...` 通过

### 来源：`0044-skills-adv-trigger-sync.md`

# 045 实施计划：技能高级能力（自动触发 + 同步）

## 实现步骤

### 1. 技能索引注入 system prompt → 验证：编译 + 测试
- 新增 `buildSkillsIndexBlock(workdir string) string` 函数
- 扫描 `<workdir>/skills/*/SKILL.md`，提取 name + description
- 限制最大 50 个条目
- 在 `buildRuntimeSystemPrompt()` 中追加注入
- 新增 `TestBuildSkillsIndexBlock` 测试（空目录、有技能、超限截断）

### 2. `skill_manage` 新增 `sync` 动作 → 验证：编译 + 测试
- 支持 `source=github`：`repo` + `path` 参数
  - 调用 `https://api.github.com/repos/{repo}/contents/{path}`
  - 遍历含 SKILL.md 的子目录，下载 SKILL.md
  - 写入本地 `skills/<name>/SKILL.md`，同时下载 support files
- 支持 `source=url`：`url` + `name` 参数
  - HTTP GET 获取原始内容
  - 写入 `skills/<name>/SKILL.md`
- 新增 `TestSkillManageSyncURL`、`TestSkillManageSyncGitHub` Mock 测试

### 3. 完整验证
- `go build ./...`
- `go test ./...` 全部通过
- `go vet ./...` 无警告

## 文件变更清单

| 文件 | 动作 | 内容 |
|------|------|------|
| `internal/agent/system_prompt.go` | 修改 | 新增 `buildSkillsIndexBlock()` |
| `internal/tools/builtin.go` | 修改 | `skill_manage` 新增 `sync` 动作 |
| `internal/tools/builtin_test.go` | 修改 | 新增 sync 测试 |
| `internal/agent/loop_test.go` | 修改 | 新增 system prompt 技能注入测试 |

## 关键设计决策

1. **技能索引注入位置**：在 `buildRuntimeSystemPrompt` 中追加，与 memory 和 workspace rules 同级
2. **索引格式**：`## Available Skills` + 强指令 + `name: description` 列表
3. **GitHub sync**：使用 GitHub Contents API（无需认证，限流 60req/h），30s 超时
4. **URL sync**：直接 HTTP GET，支持任何可公开访问的原始 SKILL.md URL
