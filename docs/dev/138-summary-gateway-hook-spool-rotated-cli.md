# 138 - Summary: CLI for rotated spool files (`list` + `replay -all`)

## Goal

After enabling spool rotation, operators need to enumerate and replay rotated spool segments. Add CLI helpers to manage rotated spool files without manual file globs.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd gateway hooks spool list` to list rotated spool files plus the base spool file.
  - Extends `agentd gateway hooks spool replay` with `-all` to replay all spool segments (rotated files oldest-first, then base).

## Usage

- `agentd gateway hooks spool list`
- `agentd gateway hooks spool replay -all`

