# 0034 skill summary merged

## 模块

- `skill`

## 类型

- `summary`

## 合并来源

- `0043-skill-summary-merged.md`

## 合并内容

### 来源：`0043-skill-summary-merged.md`

# 0043 skill summary merged

## 模块

- `skill`

## 类型

- `summary`

## 合并来源

- `0009-skill-manage-minimal.md`
- `0010-skill-manage-support-files.md`
- `0051-skill-search-hub.md`

## 合并内容

### 来源：`0009-skill-manage-minimal.md`

# 010 总结：`skill_manage` 最小补齐结果

## 已完成

- 新增 `skill_manage` 内置工具
- 支持 `create` / `edit` / `patch` / `delete` 四类动作
- 技能操作默认定位到 `<workdir>/skills`
- 复用工作区路径约束，支持受限 `path` 覆盖
- 增加技能名校验（拒绝路径穿越和非法字符）
- `patch` 支持唯一替换与 `replace_all=true` 全量替换
- 新增测试覆盖：
  - 创建、编辑、定点替换、删除主路径
  - 非法技能名拒绝
  - 多重匹配未开启 `replace_all` 的拒绝行为

## 验证

- `go test ./...` 通过

## 当前边界

本次为最小技能管理补齐，仍未覆盖：

- `write_file` / `remove_file` 支撑文件操作
- skills hub 同步与来源治理
- 自动技能触发与策略评分

### 来源：`0010-skill-manage-support-files.md`

# 011 总结：`skill_manage` 支撑文件能力补齐结果

## 已完成

- `skill_manage` 新增动作：
  - `write_file`
  - `remove_file`
- 新增 `file_path` / `file_content` 参数
- 新增支撑文件路径校验：
  - 仅允许相对路径
  - 禁止路径穿越
  - 限定允许子目录：`references`、`templates`、`scripts`、`assets`
- 新增测试覆盖：
  - 支撑文件写入与删除主路径
  - 非法路径拒绝（穿越、非法子目录）

## 验证

- `go test ./...` 通过

## 当前边界

技能系统仍未覆盖：

- skills hub 同步
- 自动技能触发与策略评分

### 来源：`0051-skill-search-hub.md`

# 052 总结：技能多源搜索（skill_search）

## 变更摘要

新增 `skill_search` 内置工具，通过 GitHub Code Search API 搜索技能仓库中的 SKILL.md 文件。

## 修改文件

| 文件 | 变更 |
|------|------|
| `internal/tools/builtin.go` | 注册 `skill_search` 工具；新增 `skillSearch`/`fetchSkillDescription` 方法 + schema |

## 新增能力

### skill_search

```
skill_search(query="github PR", repo="anthropics/skills")
→ [
    {"name": "github-pr-workflow", "description": "...", "repo": "...", "path": "..."},
    ...
  ]
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `query` | 搜索关键词 | 必填 |
| `repo` | GitHub 仓库 owner/name | `anthropics/skills` |

使用 GitHub Search API（`/search/code`），配合 `skill_manage sync` 可完成搜索 → 下载 → 安装的全流程。

## 测试结果

`go build ./...` ✅ | `go test ./...` 全部通过 ✅ | `go vet ./...` 无警告 ✅
