# 057 Research：CLI 本地诊断最小对齐

## 背景

Hermes 提供 `hermes doctor` 用于诊断本地环境。当前 Go 项目已有配置、模型、工具查看命令，但缺少启动前诊断入口。

## 目标

实现最小 `agentd doctor`，检查本地可判定的问题：

- 配置文件路径与环境变量优先级提示。
- 工作目录存在性。
- 数据目录可创建且可写。
- 当前 provider/model 是否受支持。
- 当前 provider API key 是否为空。
- MCP transport 配置是否明显错误。
- Gateway 启用时是否至少配置一个平台 token。
- 内置工具是否成功注册。

## 范围

- 不发起网络请求。
- 不调用模型 API。
- 不启动 Gateway 或 MCP 进程。
- 不检查远端凭据是否有效。

## 推荐方案

在 `cmd/agentd` 中新增 `doctor` 子命令，输出 `ok/warn/error`。硬错误返回非零退出码；缺少 API key、Gateway 启用但没有 token 等情况作为 warning。

## 三角色审视

- 高级产品：诊断覆盖用户最常见的启动前问题，不把远端健康检查纳入本期。
- 高级架构师：只读检查，不改变配置或运行时状态；数据目录检查仅创建临时文件后删除。
- 高级工程师：通过 helper 测试覆盖缺 key、坏 workdir、Gateway token 缺失等分支。
