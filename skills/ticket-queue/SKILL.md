---
name: ticket-queue
description: Universal ticket pickup, state management, and logging skill for the agent-queue system. Use at the start and end of every agent-queue ticket — picks up eligible tickets, parses Agent State, enforces the iteration cap, manages append-only logging, resolves code workspace targets, and handles stage flips. Invoke as the first action of any ticket execution and as the closing action when setting the final stage.
---

# Ticket Queue

Stateless procedural skill for interacting with the TaskNotes `agent-queue` system. Identical across all executing personas (Developer, Researcher, Auditor).

## 1. Pick up an eligible ticket

Query TaskNotes for tickets matching:
- tag: `agent-queue`
- `status` in `open`, `approved`, `ready`, or `needs-rework`

Parse each result's `## Agent State`. Only act when:
- `stage: queued` (fresh ticket), OR
- `status` just flipped to `approved`/`ready` (resuming a `needs-approval` gate), OR
- `status` just flipped to `needs-rework` (redoing a rejected attempt)

Never infer a go-ahead from body text alone — only the `status` field is authoritative.

## 2. Increment iterations

Before doing any work, increment `iterations` in `## Agent State`. Cap is **5**.

On the 5th failed attempt: set `stage: blocked`, tag `agent-issue:needs-work`, log the reason, stop. Never retry past the cap.

## 3. Append-only log discipline

Every action gets one bullet appended to `## Agent Log`:
- Never edit or delete a prior entry
- Never compress the log mid-execution (that's `context-purge`)
- Format: `- [YYYY-MM-DD] <action taken and outcome>`

## 4. Resolve the target workspace

For Developer tickets targeting code outside the vault, resolve in order:

1. `~/Documents/lang/<stack>/<project>` — any subfolder is fair game
2. `~/scripts`

For host short-names (`zp0dune`, `vp0dune`, `mdune`, etc.):
1. `~/Documents/lang/ansible/zdansible/.inventory.toml`
2. Fallback: `~/scripts/lua/scripts/dune/servers.lua`

Check both before declaring a host unreachable — the inventory address may refuse SSH while the fallback answers.

## 5. Close the ticket

Set `stage` to the appropriate value and stop:

| Outcome | Stage | Additional |
|---------|-------|------------|
| Goal fully met, verification passed | `done` | |
| Finding delivered, CTO must act | `needs-handoff` | set `artifact_path` |
| CTO decision required to proceed | `needs-approval` | log the specific question |
| Needs rework by another agent | `needs-rework` | log what must change |
| Iteration cap hit without resolution | `blocked` | tag `agent-issue:needs-work` |
| Change caused a regression | `failed` | tag `agent-issue:bug`; invoke `incident-response` |

Never set `stage: done` with a known open issue.

## Script convention

When a ticket requires a deterministic, reusable procedure (sending an email, calling an API,
running a build step), prefer a script over inline shell commands.

- **Check first:** `~/scripts/SCRIPTS.md` — index of all existing scripts.
- **Create only in:** `~/scripts/` — never in `/tmp`, the vault, or skill directories.
- **Languages:** bash, python, or lua. Prefer lua when the user will self-run via `zdcli`.
- **After creating/updating:** add or update the entry in `~/scripts/SCRIPTS.md`.
