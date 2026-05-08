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
