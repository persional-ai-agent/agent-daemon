# agent-daemon

Go 版 Agent 守护进程与交互式 CLI，参考 `/data/source/hermes-agent` 的 Agent 架构实现。

## 当前能力

- OpenAI 兼容 `chat/completions` 模型调用
- 多轮 Agent Loop，支持 `tool_calls`
- 内置工具注册与分发
- 内置工具：`terminal`、`process_status`、`stop_process`、`read_file`、`write_file`、`search_files`、`todo`、`memory`、`session_search`、`web_fetch`
- SQLite 会话持久化
- `MEMORY.md` / `USER.md` 持久记忆
- CLI 交互模式
- HTTP API 服务模式

## 快速开始

```bash
go run ./cmd/agentd chat
```

设置模型：

```bash
export OPENAI_API_KEY=your_key
export OPENAI_MODEL=gpt-4o-mini
export OPENAI_BASE_URL=https://api.openai.com/v1
```

启动 HTTP 服务：

```bash
go run ./cmd/agentd serve
```

请求示例：

```bash
curl -s http://127.0.0.1:8080/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"请读取 README.md 并总结"}'
```

## 文档

- `docs/overview-product.md`
- `docs/overview-product-dev.md`
- `docs/dev/README.md`
