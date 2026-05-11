# 101 Summary - 飞书（Feishu/Lark）doc/drive 工具实现（OpenAPI）

## 变更

将以下工具从占位升级为可用实现（需要 `FEISHU_APP_ID` + `FEISHU_APP_SECRET`；可选 `FEISHU_BASE_URL`，默认 `https://open.feishu.cn`）：

- `feishu_doc_read doc_token=...`：GET `/open-apis/docx/v1/documents/{doc_token}/raw_content`
- `feishu_drive_list_comments`：GET `/open-apis/drive/v1/files/{file_token}/comments`
- `feishu_drive_list_comment_replies`：GET `/open-apis/drive/v1/files/{file_token}/comments/{comment_id}/replies`
- `feishu_drive_reply_comment`：POST `/open-apis/drive/v1/files/{file_token}/comments/{comment_id}/replies`
- `feishu_drive_add_comment`：POST `/open-apis/drive/v1/files/{file_token}/new_comments`

鉴权：

- 使用 tenant access token：POST `/open-apis/auth/v3/tenant_access_token/internal/`（带缓存）

实现位置：

- `internal/tools/feishu.go`
- `internal/tools/builtin.go`：注册与 schema（沿用同名 params 函数）
- `internal/tools/toolsets.go`：toolset `feishu`

