---
name: postmortem
description: Blameless postmortem process for incidents and significant failures. Use after any incident that consumed meaningful error budget, caused user-visible downtime, or required a rollback. Invoke within 48 hours of incident resolution. Produces a structured postmortem document and action-item tickets. Use whenever asked to conduct a postmortem, review an incident, or create an incident report.
---

# Postmortem

Blameless post-incident review. The goal is to understand the system, not assign fault. Humans make mistakes in complex systems; the right response is to make the system harder to fail, not to find someone to blame.

## When to write one

- Any incident where error budget burn rate exceeded 2× for > 1 hour
- Any rollback was performed
- Any CTO was paged or manually intervened
- Any data was lost or corrupted (even partially)

For small self-resolved blips (< 5 min, no user impact, no rollback): a log entry on the ticket is sufficient — no postmortem needed.

## Document location

Write to: `+/Things/AI/Agents/SRE/postmortems/<YYYY-MM-DD>-<slug>.md`

## Postmortem format

```markdown
# Postmortem: <short title>

**Date:** <YYYY-MM-DD>
**Duration:** <start time> → <end time> (<total minutes>)
**Severity:** S<1–4> (see defect-triage for definitions)
**Services affected:** <list>
**Error budget impact:** <% of monthly budget consumed>
**Ticket:** [[TaskNotes/Tasks/<ticket>]]

## Impact

<One paragraph: what users/systems experienced and for how long. Use concrete numbers — "API returned 503 for 12 minutes, affecting all authenticated requests.">

## Timeline

| Time | Event |
|------|-------|
| HH:MM | <what happened> |
| HH:MM | <detection / first alert> |
| HH:MM | <response action> |
| HH:MM | <resolution> |

## Root cause

<One paragraph: the specific technical condition that caused the failure. Not "human error" — what was the system property that allowed the error to propagate?>

## Contributing factors

<Bullet list: conditions that made the root cause possible or worse. These are the levers — fixing contributing factors is usually more durable than fixing the root cause alone.>

## What went well

<What detection/response mechanisms worked? What limited the blast radius?>

## What went poorly

<Where did the response slow down? What was unclear or missing?>

## Action items

| Item | Owner | Ticket | Due |
|------|-------|--------|-----|
| <specific fix or improvement> | Developer / SRE / QA | [[link]] | <date> |

Action items must be specific and measurable. "Improve monitoring" is not an action item. "Add alert for Nomad allocation failure rate > 5% sustained for 2 min" is.
```

## Creating action-item tickets

For each action item in the table:
1. Create a TaskNotes ticket with the appropriate `agent-kind`
2. Set `priority` based on severity: S1 → `urgent`, S2 → `high`, S3/S4 → `normal`
3. Link back to the postmortem in `## Request`
4. Set `artifact_path` on the postmortem ticket to the document path

## Filing

1. Set `stage: needs-handoff` on the incident ticket once postmortem is written
2. CTO reviews and approves action items (flips status to `approved`)
3. Action-item tickets enter the queue normally
