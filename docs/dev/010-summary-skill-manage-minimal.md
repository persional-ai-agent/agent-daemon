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
