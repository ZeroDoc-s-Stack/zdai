---
name: context-purge
description: Quarterly context and log compression skill. Use to compress closed ticket Agent Logs, trim oversized handoff artifacts, and reconcile Lessons Learned sections into durable rule changes. Run by the Auditor during its quarterly pass — not per-cycle, and never on an open ticket. Invoke when asked to clean up ticket context, compress agent logs, or trim stale artifacts.
---

# Context Purge

Quarterly housekeeping for ticket logs and handoff artifacts. Runs as part of the Auditor's quarterly pass — not automated per-cycle.

## Why not per-cycle

Per-cycle purging risks trimming an `## Agent Log` entry still load-bearing for a ticket in flight (e.g. the reasoning behind a `needs-approval` flip the CTO hasn't answered yet). Quarterly, human-paced review avoids that — by then, a ticket's history is either closed and safe to compress, or still open and worth reading in full.

## Steps

1. **Scope.** Pull tickets at terminal stages (`done`, `failed`) with `dateModified` older than ~3 months, plus any `## Lessons Learned` section across the four persona specs that's grown long enough to be skimmed rather than read.

2. **Compress, don't delete, `## Agent Log`.** For a closed ticket with a long log, collapse the play-by-play into one summary bullet (what was attempted, what worked, final stage). Keep the original entries in Obsidian's edit history so nothing is unrecoverable — just no longer taking up live context on every future read.

3. **Trim oversized handoff artifacts.** If a `## Handoff Artifact` section or its linked `artifact_path` note has grown large (e.g. a full fetched dataset already delivered and acted on), replace the inline content with a short pointer to where the full data lives, once confirmed the CTO no longer needs it inline.

4. **Reconcile `## Lessons Learned`.** Fold recurring actual-vs-estimated drift into durable rule changes in the Senior Planner's estimation guidance (proposed via handoff, not applied unilaterally — same CTO-nod discipline as everything else). Trim the section back to unreconciled entries only after the CTO approves the proposed changes.

5. **Never purge an open ticket.** Anything not at a terminal stage is out of scope regardless of age — an old `needs-handoff`/`blocked` ticket is waiting on the CTO, not stale.
