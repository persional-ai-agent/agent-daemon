# 0030 rl summary merged

## 模块

- `rl`

## 类型

- `summary`

## 合并来源

- `0039-rl-summary-merged.md`

## 合并内容

### 来源：`0039-rl-summary-merged.md`

# 0039 rl summary merged

## 模块

- `rl`

## 类型

- `summary`

## 合并来源

- `0102-rl-tools-local-runner.md`

## 合并内容

### 来源：`0102-rl-tools-local-runner.md`

# 103 Summary - RL 工具从占位升级为本地 runner（可配置命令）

## 变更

将 `rl_*` 工具从占位升级为可用的本地 runner：

- 状态存储：`{workdir}/.agent-daemon/rl_state.json`
- 环境列表：读取 `RL_ENVIRONMENTS`（逗号分隔）
- 训练启动：读取 `RL_TRAIN_COMMAND`，支持 `{env}` 占位符，后台启动并记录 `session_id/output_file`
- 推理测试：读取 `RL_INFER_COMMAND`，支持 `{env}` 占位符，前台执行并返回 `output/exit_code`
- 支持：
  - `rl_list_environments`
  - `rl_select_environment`
  - `rl_get_current_config`
  - `rl_edit_config`
  - `rl_start_training` / `rl_stop_training` / `rl_check_status`
  - `rl_get_results`
  - `rl_list_runs`（最小：空列表）
  - `rl_test_inference`（基于 `RL_INFER_COMMAND`）

实现位置：

- `internal/tools/rl_tools.go`
- `internal/tools/builtin.go`：注册与 schema 更新
