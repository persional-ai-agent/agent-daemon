# Agent Daemon

> 一个基于 本地 LLM Agent

## 主要两个进程

- agent-daemon: 后台进程
- agent-cli: 与 agent-daemon 通信交互的终端进程

## 编译

在源码根目录中执行

```shell
make
```

即可看到两个进程

## 当前状态（2026-05-16）

- `docs/dev/0015-hermes-plan-merged.md` 中定义的 `TODO-001` ~ `TODO-016` 已全部完成并标记为 `done`。
- 已补齐的关键收口项包含：
  - ACP 完整协议层最小闭环（capabilities、session 管理、鉴权、事件映射）
  - Research/RL trajectory 闭环（run/compress/stats/export + 过滤/导出）
  - Setup/迁移/回滚与 shell completion 运维闭环
  - Web Dashboard 日常管理闭环（session rename/delete/export）
  - Toolsets Web 管理闭环（list + set/enable/disable/clear）
  - 工具能力级补齐收口（含 `transcription`）

详情见：
- `docs/dev/0015-hermes-plan-merged.md`
- `docs/dev/0015-hermes-summary-merged.md`
