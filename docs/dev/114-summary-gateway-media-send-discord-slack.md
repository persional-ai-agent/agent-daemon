# 114 - Summary: Gateway media delivery for Discord/Slack

## Goal

Align Hermes “MEDIA:” / attachment delivery experience so that `text_to_speech` (and other tools producing local artifacts) can be delivered through gateway adapters, not only as local files.

## What changed

- Implemented optional `platform.MediaSender` on gateway adapters:
  - Discord: `internal/gateway/platforms/discord.go` now supports sending a local file as an attachment with optional caption and reply reference.
  - Slack: `internal/gateway/platforms/slack.go` now supports uploading a local file (files.upload.v2 flow in slack-go) with optional initial comment and thread reply.
- `send_message` already supports `media_path` and `MEDIA:` prefix; with these adapter changes, `send_message` can now deliver local files on Discord/Slack when the gateway is connected.

## Usage

- Direct attachment:
  - `send_message(action="send", platform="discord", chat_id="<channel_id>", media_path="/tmp/a.mp3", message="caption")`
  - `send_message(action="send", platform="slack", chat_id="<channel_id>", media_path="/tmp/a.mp3", message="caption")`
- Hermes-style:
  - `send_message(..., message="MEDIA: /tmp/a.mp3")`

## Notes / limitations

- Slack upload uses `slack.UploadFileParameters` which requires a non-zero `FileSize`; we `stat()` the local file before upload.
- Discord attachment delivery requires the bot to have permissions to post messages and attach files in the target channel.
- Yuanbao media delivery is still pending (typically requires an upload/hosting flow rather than raw local file paths).

