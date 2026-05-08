# 038 调研：`model_stream_event` v2 用量状态表驱动测试补齐

## 背景

当前 `usage_consistency_status` 的断言已较多，分散在多个独立测试中，新增状态或规则时维护成本上升。

## 缺口

- 缺少统一表驱动用例，难以一眼覆盖核心状态矩阵
- 新增状态时容易漏改多个测试函数

## 本轮目标

新增一个表驱动测试，统一覆盖核心状态：

- `ok`
- `derived`
- `source_only`
- `adjusted`

并同时校验关键副字段（如 `total_tokens_adjusted`、`total_tokens`）。

## 本轮边界

- 仅补测试组织方式，不改业务规则
