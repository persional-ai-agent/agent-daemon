# 075 总结：terminal/approval/skills 工具返回补齐 success

为对齐 Hermes 常见返回格式并便于上层统一处理，本次为以下工具返回增加 `success` 字段：

- `terminal`（前台：`success = (error==nil && exit_code==0)`；后台：`success=true`；pending approval：`success=false`）
- `process_status` / `stop_process`（兼容工具）
- `approval`
- `skill_list`/`skills_list`、`skill_view`、`skill_search`

不移除原字段，保持兼容。

