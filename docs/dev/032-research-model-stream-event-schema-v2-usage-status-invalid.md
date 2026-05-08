# 032 调研：`model_stream_event` v2 用量异常状态补齐

## 背景

031 已新增 `usage_consistency_status`，但对于上游返回的非数值 token（如字符串脏值）仍缺明确状态表达。

## 缺口

- `usage` 出现 token 字段但无法解析时，客户端无法区分“无数据”与“脏数据”
- 现有状态集合缺少异常输入标识

## 本轮目标

新增并补齐异常状态：

- `usage_consistency_status=invalid`

触发条件：

- 存在 token 相关字段，但无法形成可用一致性路径（无法推导/校正/判定 source_only）

## 本轮边界

- 仅新增最小异常状态，不扩展错误码体系
