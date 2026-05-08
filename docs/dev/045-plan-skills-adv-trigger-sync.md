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
