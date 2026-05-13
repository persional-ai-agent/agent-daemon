# Breaking Change Acknowledgement

当 `docs/api/ui-chat-contract.openapi.yaml` 出现 breaking 变更时，PR 必须增加：

- `.contract/breaking-change-ack.md`

该文件最少应包含：

1. 变更点（删除/改名/类型变更）
2. 受影响客户端
3. 迁移方案与窗口
4. 回滚策略

CI 会在检测到 breaking 变更时检查此文件是否存在。
