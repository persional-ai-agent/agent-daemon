# 014 计划：MCP `/call` 流式响应兼容

## 目标

让 MCP `/call` 在 `text/event-stream` 响应下也能返回稳定结果，并与现有非流式调用保持兼容。

## 实施步骤

1. 在 `mcpToolProxy.Call` 中识别 `Content-Type: text/event-stream`。
2. 新增 SSE 解析函数：
   - 按事件空行分隔
   - 聚合 `data:` 多行
   - 解析 JSON 事件并提取 `result/structuredContent/error`
3. 保留原非流式 JSON 处理逻辑，避免回归。
4. 新增测试覆盖 SSE `/call` 成功链路。
5. 运行 `go test ./...` 全量回归。

## 验证标准

- SSE MCP `/call` 能返回正确结果
- 原 HTTP JSON 模式无回归
- 全量测试通过
