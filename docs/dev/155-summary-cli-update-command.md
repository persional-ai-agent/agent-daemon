# 155 - Summary: `update` minimal git-based flow

## Goal

Close the remaining CLI update gap for developer/git-checkout installs by adding a minimal command that can inspect and apply fast-forward updates.

## What changed

- `cmd/agentd/main.go`:
  - Adds `agentd update` with `check` and `apply` modes.
  - `check` can optionally run `git fetch` and then reports branch, commit, upstream, ahead/behind counts, dirty state, and fast-forward eligibility.
  - `apply` runs `git fetch` + `git pull --ff-only` and reports the before/after commit IDs.
  - The implementation explicitly scopes itself to git checkouts in this minimal phase.
- Documentation:
  - Refreshes CLI parity docs to mark `update` as minimally available while keeping installer-level update flow as a remaining gap.
