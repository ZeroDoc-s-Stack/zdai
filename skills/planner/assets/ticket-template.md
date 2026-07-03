---
title: "{{title}}"
status: open
priority: normal
contexts: []
tags:
  - task
  - agent-queue
dateCreated: "{{date}}"
dateModified: "{{date}}"
---
Part of plan: {{plan link}}

(Add an `agent-kind:<kind>` tag above — coding / email-approval / ui-design /
hardware / api-consumer / general / research / audit. Also add `plan:<slug>` if
from a planning round.)

## Goal

(one line: the concrete, checkable success condition)

## User Input

(leave blank — CTO answers here, then flips `status` to
`approved`/`ready`/`needs-rework` to signal the executing agent)

## Agent State

- stage: queued
- iterations: 0
- artifact_path:
- agile-point: {{points}}

## Agent Log

---

## Credentials handling (include only when a ticket actually needs a secret)

```markdown
## Credentials Needed

(state exactly what's required and how to supply it — e.g. "read-only API
key for <service>, scoped to <endpoint>")

## Credentials

(CTO pastes the value here. The agent reads it once for this ticket's own
use and never echoes it back into Agent Log, a handoff artifact, or a
commit message.)
```
