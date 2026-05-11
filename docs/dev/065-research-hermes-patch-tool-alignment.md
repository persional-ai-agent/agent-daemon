# 065 调研：Hermes patch 工具与 Go 版最小对齐

## Hermes 现状（参考）

Hermes 内置 `patch` 工具用于对文件做局部修改，避免整文件重写带来的 token 与冲突风险。

## 当前项目差异

Go 版此前只有 `write_file`（整文件写入）与 `skill_manage patch`（仅技能文件），缺少通用 `patch`。

## 最小对齐目标（本次）

- 新增通用 `patch` 工具：
  - `path`：文件路径（限制在 `AGENT_WORKDIR` 内）
  - `old_string/new_string`：字符串替换
  - `replace_all`：控制是否允许多处匹配

## 不在本次范围

- unified diff / fuzzy patch / 多文件 patch。

