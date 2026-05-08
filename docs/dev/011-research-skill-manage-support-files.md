# 011 调研：`skill_manage` 支撑文件能力补齐

## 背景

010 已补齐 `skill_manage` 的基础生命周期（`create/edit/patch/delete`），但仍有明显缺口：

- 无法为技能写入 `references/templates/scripts/assets` 等支撑文件
- 无法删除技能内无用支撑文件

这会限制技能从“单文件说明”演进为“可复用资产包”。

## 与 Hermes 差异

Hermes 的 `skill_manage` 已支持：

- `write_file`
- `remove_file`

并且对路径有子目录约束，防止越权写入。

## 本轮最小补齐目标

- 在 Go 版 `skill_manage` 增加：
  - `write_file`
  - `remove_file`
- 路径约束：
  - 必须为相对路径
  - 禁止路径穿越
  - 必须落在允许子目录：`references/`、`templates/`、`scripts/`、`assets/`
- 保持所有写删操作仍受工作区边界约束。
