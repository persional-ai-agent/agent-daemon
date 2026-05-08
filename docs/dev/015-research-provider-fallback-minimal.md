# 015 调研：Provider 故障切换最小补齐

## 背景

当前模型调用已支持 OpenAI / Anthropic / Codex 三种 provider，但一次运行仅绑定单 provider。  
当主 provider 出现限流或暂时故障时，会直接失败。

## 缺口

- 无主备 provider 自动切换
- 无可配置的故障降级路径

## 本轮目标

补齐最小故障切换能力：

- 新增 `FallbackClient` 包装器
- 主 provider 失败且命中可重试错误（如 429/5xx/timeout）时，自动回退备用 provider
- 通过配置启用：`AGENT_MODEL_FALLBACK_PROVIDER`

## 本轮边界

- 不做并行竞速请求
- 不做多级级联 fallback
- 不做 provider 健康探针与熔断器
