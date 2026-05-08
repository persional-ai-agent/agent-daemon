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
