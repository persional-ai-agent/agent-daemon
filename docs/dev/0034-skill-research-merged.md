# 0034 skill research merged

## 模块

- `skill`

## 类型

- `research`

## 合并来源

- `0043-skill-research-merged.md`

## 合并内容

### 来源：`0043-skill-research-merged.md`

# 0043 skill research merged

## 模块

- `skill`

## 类型

- `research`

## 合并来源

- `0009-skill-manage-minimal.md`
- `0010-skill-manage-support-files.md`

## 合并内容

### 来源：`0009-skill-manage-minimal.md`

# 010 调研：`skill_manage` 最小补齐

## 背景

当前项目已具备：

- `skill_list`：列出本地技能
- `skill_view`：读取单个技能

但仍缺少 Hermes 中由 Agent 直接维护技能内容的关键入口，即 `skill_manage`。这会导致技能体系只能“读”，不能“写”。

## 与 Hermes 的差异

Hermes 的 `skill_manage` 支持较完整生命周期（create/edit/patch/delete/write_file/remove_file）以及更复杂的安全、审计与策略。

当前 Go 版差异集中在：

- 无技能创建能力
- 无技能修改能力
- 无技能删除能力
- 无“定点 patch”能力

## 本轮补齐目标（最小可用）

在保持现有工作区安全边界与简洁实现的前提下，先补齐最小管理闭环：

- 新增 `skill_manage` 工具
- 支持动作：`create` / `edit` / `patch` / `delete`
- 操作范围限制在 `<workdir>/skills`（或工具参数指定的受限子路径）
- 技能名校验，拒绝路径穿越与目录分隔符
- `patch` 默认要求唯一匹配，`replace_all=true` 才允许批量替换

## 明确暂不覆盖

为了避免一次性引入过高复杂度，本轮不做：

- `write_file` / `remove_file` 支撑文件管理
- Skills hub 同步与来源追踪
- 技能使用统计、pin/curator 联动
- 自动触发与策略评分

### 来源：`0010-skill-manage-support-files.md`

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
