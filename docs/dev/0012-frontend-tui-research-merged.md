# 0012 frontend-tui research merged

## 模块

- `frontend-tui`

## 类型

- `research`

## 合并来源

- `0019-frontend-tui-research-merged.md`

## 合并内容

### 来源：`0019-frontend-tui-research-merged.md`

# 0019 frontend-tui research merged

## 模块

- `frontend-tui`

## 类型

- `research`

## 合并来源

- `0206-frontend-tui-parity.md`

## 合并内容

### 来源：`0206-frontend-tui-parity.md`

# Frontend 与 TUI 对齐调研（Phase 1）

## 背景

当前 `agent-daemon` 已有核心 API、CLI、Gateway 与工具体系，但缺少独立 Web 前端工程与完整 TUI 产品面。  
对比 `/data/source/hermes-agent`，前端/TUI 是独立子系统，不是单文件补丁可对齐。

## 差距结论

1. 仓库内无 `web/` 与 `ui-tui/` 工程基座。
2. CLI 交互缺少统一 slash 命令面，无法承载 “类 TUI” 的日常操作流。
3. 使用文档与开发文档缺少前端/TUI 专章，无法指导后续迭代。

## Phase 1 目标

1. 新增 `web/` 最小可运行工程，先打通 chat/cancel 主链路。
2. 增强 CLI 交互，补齐核心 slash 命令（help/session/tools/history/reload/clear/tui）。
3. 补齐产品与开发文档，明确后续分批收敛路径。

## 边界

- 本阶段不引入完整 Hermes UI/TUI 功能全集（插件槽位、复杂主题系统、完整 slash 生态等）。
- 本阶段重点是“工程基座 + 可运行链路 + 文档对齐”，确保后续可持续推进。
