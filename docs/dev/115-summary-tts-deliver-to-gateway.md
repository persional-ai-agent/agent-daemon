# 115 - Summary: `text_to_speech` optional gateway delivery (`deliver=true`)

## Goal

Reduce friction for Hermes-style “generate media then deliver to current chat” by letting `text_to_speech` optionally push the generated audio file through the active gateway adapter when running inside a gateway-triggered tool context.

## What changed

- `text_to_speech` schema adds:
  - `format` (mp3/wav/opus/aac; used by real backends)
  - `deliver` (boolean)
- When `deliver=true` and a gateway context is present (`ToolContext.GatewayPlatform` + `GatewayChatID`):
  - If the connected adapter implements `platform.MediaSender`, the tool sends the generated file immediately.
  - Tool output includes `delivered`, `delivery_platform`, `delivery_chat_id`, `delivery_message_id` (best-effort).
  - If delivery is not possible, tool still succeeds (audio generated) but sets `delivered=false` and `delivery_error`.

## Usage

- In a gateway-triggered conversation:
  - `text_to_speech(text="...", format="mp3", deliver=true)`

## Notes

- Delivery currently works on adapters that support `platform.MediaSender` (Telegram/Discord/Slack). Yuanbao delivery remains pending.
- Reply behavior uses the triggering message id as `reply_to` when available.

