# 0034 skill plan merged

## 模块

- `skill`

## 类型

- `plan`

## 合并来源

- `0043-skill-plan-merged.md`

## 合并内容

### 来源：`0043-skill-plan-merged.md`

# 0043 skill plan merged

## 模块

- `skill`

## 类型

- `plan`

## 合并来源

- `0009-skill-manage-minimal.md`
- `0010-skill-manage-support-files.md`

## 合并内容

### 来源：`0009-skill-manage-minimal.md`

# 010 计划：`skill_manage` 最小补齐

## 目标

补齐 Go 版技能管理最小闭环，使 Agent 在工作区内可安全地创建、编辑、定点修改和删除技能。

## 实施步骤

1. 在 `internal/tools/builtin.go` 注册 `skill_manage` 工具及参数 schema。
2. 实现 `skill_manage` handler，支持：
   - `create`：创建 `<skills>/<name>/SKILL.md`
   - `edit`：覆盖 `SKILL.md`
   - `patch`：按 `old_string/new_string` 做唯一或全量替换
   - `delete`：删除技能目录
3. 复用工作区路径约束，确保技能操作不越界。
4. 增加技能名校验规则（仅允许安全字符集）。
5. 在 `internal/tools/builtin_test.go` 补充功能测试与拒绝路径测试。
6. 执行 `go test ./...` 完整回归。
7. 更新文档索引与总览文档。

## 验证标准

- `skill_manage` 四类动作可用且返回结构化结果。
- 非法技能名会被拒绝。
- `patch` 在多重匹配且未开启 `replace_all` 时拒绝执行。
- 全量测试 `go test ./...` 通过。

### 来源：`0010-skill-manage-support-files.md`

# 011 计划：`skill_manage` 支撑文件能力补齐

## 目标

让 `skill_manage` 支持技能目录内支撑文件的写入与删除，保持最小可用和安全约束。

## 实施步骤

1. 在 `skill_manage` 中新增动作分支：
   - `write_file`
   - `remove_file`
2. 扩展 `skill_manage` 参数 schema：
   - `file_path`
   - `file_content`
3. 新增路径校验逻辑：
   - 仅允许相对路径
   - 拒绝 `..` 穿越
   - 强制首段在 `references/templates/scripts/assets`
4. 新增测试：
   - 写入并删除支撑文件成功
   - 非法路径被拒绝（穿越、非法子目录）
5. 执行 `go test ./...` 回归。

## 验证标准

- `write_file/remove_file` 可用且只在允许子目录生效
- 非法路径有明确拒绝错误
- 全量测试通过
