# 067 总结：Hermes clarify 工具最小对齐

## 完成情况

- 新增内置工具 `clarify`（返回结构化 question/options）。
- toolsets 增加 `clarify` 并纳入 `core`。
- 文档与索引更新。

## 边界

- `clarify` 不直接收集用户答案；需要用户下一条消息回复选项 label 或自由文本。

