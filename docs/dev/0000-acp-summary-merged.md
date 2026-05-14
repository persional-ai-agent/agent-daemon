# 0000 acp summary merged

## 模块

- `acp`

## 类型

- `summary`

## 合并来源

- `0000-acp-summary-merged.md`

## 合并内容

### 来源：`0000-acp-summary-merged.md`

# 0000 acp summary merged

## 模块

- `acp`

## 类型

- `summary`

## 合并来源

- `0258-acp-api-adapter-minimal.md`

## 合并内容

### 来源：`0258-acp-api-adapter-minimal.md`

# 258 总结：ACP/IDE 最小 API 适配闭环

本次补齐 ACP/IDE 差异的最小可用闭环，复用现有 chat engine 与会话机制，新增 ACP 兼容接口层。

## 新增接口

- `POST /v1/acp/sessions`
  - 创建会话（可传 `session_id`，不传则自动生成）
- `POST /v1/acp/message`
  - 字段：`session_id`、`input`、`turn_id`、`resume`
  - 语义映射到现有 `/v1/chat`
- `POST /v1/acp/message/stream`
  - 语义映射到现有 `/v1/chat/stream`
- `POST /v1/acp/cancel`
  - 语义映射到现有 `/v1/chat/cancel`

## 实现方式

- 文件：`internal/api/server.go`
  - 增加 ACP request 结构
  - 增加 ACP handler
  - 通过请求体字段映射到 `chatRequest/cancelRequest`，直接复用既有处理逻辑

## 测试

- 文件：`internal/api/server_test.go`
  - 新增 `acp_sessions` / `acp_message` / `acp_stream` 回归用例

验证：

- `go test ./internal/api -count=1`
- `go test ./...`
