# 140 - Summary: Filtered webhook spool replay (`-type` / `-id`)

## Goal

Improve webhook spool replay operability by allowing targeted replay for specific event classes or a single event.

## What changed

- `cmd/agentd/main.go`:
  - Extends `agentd gateway hooks spool replay` with:
    - `-type <event_type>`: replay only matching event type
    - `-id <event_id>`: replay only matching event id
  - Non-matching entries are preserved in spool; only matching entries are attempted.

## Usage

- `agentd gateway hooks spool replay -type gateway.delivery.media`
- `agentd gateway hooks spool replay -id <event_id>`

