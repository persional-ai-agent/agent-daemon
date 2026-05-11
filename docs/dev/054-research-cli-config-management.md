# 054 Research：CLI 配置管理最小对齐

## 背景

Hermes 提供 `hermes config set` 等配置管理入口，使用户可以不直接编辑配置文件完成模型、工具、网关等运行参数调整。当前 Go 项目已经支持从 `config/config.ini` 与环境变量加载配置，但缺少 CLI 读写入口。

## 目标

补齐最小 CLI 配置管理面：

- 查看当前配置文件中的键值。
- 读取单个 `section.key`。
- 写入单个 `section.key`。
- 保持环境变量优先级不变。

## 范围

本次只实现 INI 文件管理，不做完整 Hermes 配置系统：

- 不引入 YAML 配置。
- 不实现交互式 setup。
- 不做 provider/model 在线发现。
- 不实现工具启停配置。

## 方案

- 在 `internal/config` 增加小型 INI 管理函数：`ListConfigValues`、`ReadConfigValue`、`SaveConfigValue`。
- 增加 `AGENT_CONFIG_FILE` 作为配置文件路径覆盖入口；未设置时沿用 `config/config.ini` / `config.ini` 查找。
- 在 `cmd/agentd` 增加 `config list|get|set` 子命令。
- `config list` 默认隐藏包含 `api_key/token/secret/password` 的值，避免误打印凭据。

## 三角色审视

- 高级产品：解决用户配置入口缺失，不扩展到 setup wizard。
- 高级架构师：复用现有 INI 配置和 `ini.v1` 依赖，不改变运行时配置结构。
- 高级工程师：新增单元测试覆盖读写、列表排序、密钥脱敏与 `AGENT_CONFIG_FILE`。
