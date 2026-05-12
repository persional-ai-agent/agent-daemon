# 117 - Summary: Yuanbao media delivery via COS upload (best-effort)

## Goal

Close the remaining “gateway media delivery” gap for Yuanbao by supporting `send_message(media_path=...)` / `MEDIA:` on the Yuanbao adapter.

Hermes Yuanbao 媒体投递依赖：调用 `genUploadInfo` 获取 COS 临时凭证 → PUT 上传 → 构造 `TIMImageElem` / `TIMFileElem` 发送。

## What changed

- 新增 Yuanbao COS 上传最小实现（Go 版 port）：
  - `internal/yuanbao/media.go`：`GetUploadInfo` + `UploadToCOS` + COS HMAC-SHA1 签名
  - `internal/yuanbao/proto.go`：补齐 `TIMImageElem` / `TIMFileElem` 所需的 MsgContent 编码
  - `internal/yuanbao/client.go`：新增 `SendC2CImage/SendGroupImage/SendC2CFile/SendGroupFile`
- `internal/gateway/platforms/yuanbao.go` 实现 `platform.MediaSender`：
  - 自动判断图片（`image/*`）→ `TIMImageElem`
  - 其他类型 → `TIMFileElem`
  - caption 作为单独的文本消息发送（Yuanbao 媒体 elem 不保证支持 caption 字段）

## Usage

- `send_message(action="send", platform="yuanbao", chat_id="direct:<uid>", media_path="/tmp/a.mp3", message="caption")`
- `send_message(action="send", platform="yuanbao", chat_id="group:<group_code>", message="MEDIA: /tmp/a.png")`

## Notes / limitations

- 依赖对外网络访问：需要能访问 `YUANBAO_API_DOMAIN` 以及 COS 上传域名；在受限网络环境下会失败并返回错误。
- 文件大小限制当前为 50MB（与 Hermes 默认一致）。
- `uuid` 使用 MD5 hex（Hermes 行为）；图片 `image_format` 做了最小映射（jpeg/gif/png/bmp，否则 255）。

