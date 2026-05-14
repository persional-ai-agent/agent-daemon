# 0039 tools plan merged

## 模块

- `tools`

## 类型

- `plan`

## 合并来源

- `0018-file-plan-merged.md`
- `0023-homeassistant-plan-merged.md`
- `0034-process-plan-merged.md`
- `0049-tool-plan-merged.md`

## 合并内容

### 来源：`0049-tool-plan-merged.md`

# 0049 tool plan merged

## 模块

- `tool`

## 类型

- `plan`

## 合并来源

- `0058-tool-disable-config.md`

## 合并内容

### 来源：`0058-tool-disable-config.md`

# 059 Plan：工具禁用配置最小实现

## 任务

1. 增加配置项。
   - 验证：`LoadFile` 能读取 `[tools] disabled`。
2. 增加 registry 禁用能力。
   - 验证：禁用后 schema 中不再出现对应工具。
3. 增加 CLI。
   - 验证：`tools disable` 写入列表，`tools enable` 移除列表，`tools disabled` 可查看。
4. 更新文档。
   - 验证：README、overview、dev index 均说明新能力。

## 边界

- 不做 toolset。
- 不做平台级工具配置。
- 不改变已有工具实现。

## 验证

- `GOCACHE=/tmp/agent-daemon-gocache go test ./internal/config ./cmd/agentd`
- `GOCACHE=/tmp/agent-daemon-gocache go test ./...`
- 手动验证 `tools disable/list/enable`
