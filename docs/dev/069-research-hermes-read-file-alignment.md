# 069 调研：Hermes read_file 输出格式与 Go 版对齐

## Hermes 现状（参考）

Hermes 的 `read_file` 返回纯文本内容（并用 offset/limit 做分页），便于模型直接粘贴/分析，不需要再剥离行号前缀。

## 当前项目差异

Go 版此前默认在每行前加 `N→` 行号，这会：

- 增加 token 开销
- 影响模型进行精确字符串匹配/patch

## 对齐目标（本次）

- `read_file` 默认返回纯文本（不带行号）。
- 提供可选 `with_line_numbers=true` 保留旧行为（调试/定位场景）。

