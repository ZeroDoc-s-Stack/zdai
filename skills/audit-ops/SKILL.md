---
name: audit-ops
description: Audit operations skill for agent-kind:audit tickets. Use for drift reviews (spec vs. actual behavior), Lessons Learned reconciliation, coding-standard compliance spot-checks, technical debt classification, and stale-ticket sweeps. Read-only — proposes changes as a handoff artifact, never applies them directly. Invoke when running a quarterly audit pass or when any of the five audit scopes is explicitly requested.
---

# Audit Ops

Step-by-step procedures for the five audit scopes. A ticket's `## Request` should scope itself to one or more of (a)–(e) — don't combine all five into one pass unless explicitly asked for a "full quarterly audit."

Ticket state flow for all scopes: `stage: queued` → [audit work] → write ADR to `artifact_path` → `stage: needs-handoff`.

---

## (a) Drift review

Find cases where a persona's spec says "do X" but recent tickets show it consistently doing Y.

1. Read the `## Request` to know which persona(s) to audit.
2. For each target persona, read its `*-persona.md` and all  the skills it uses.
3. Query TaskNotes for the persona's 10–20 most recent tickets (filter: `agent-queue` tag, matching persona context, `status` in `done`/`needs-handoff`). Read each ticket's `## Agent Log` in full.
4. Note specific divergences — quote both the spec text and the log text side by side.
5. Classify each divergence: **accidental** (one-off slip, flag on that ticket) or **systematic** (pattern across multiple tickets, needs a spec edit).
6. For each systematic divergence, draft a specific, minimal edit to the relevant `*-persona.md` or skill. Don't rewrite sections the evidence doesn't support.
7. Write findings to the ADR: one section per persona or skill, each divergence as a bullet with evidence and proposed fix.

---

## (b) Lessons Learned reconciliation

Fold recurring patterns into durable rule changes — from both persona logs and skill execution traces.

### Persona Lessons Learned

1. Read each persona's `## Lessons Learned` section.
2. Group entries by theme (estimation misses, recurring access blockers, project-specific quirks).
3. For each recurring pattern (≥2 occurrences, not noise), draft the concrete rule change it implies and the specific text to add to the relevant persona file. Propose in the ADR — don't apply directly.
4. Leave one-off entries in place. Note in the ADR which entries were folded vs. left.
5. Actual clearing of reconciled entries happens only after CTO approves the proposed rule changes (handled by a follow-on coding ticket).

### Skill execution trace mining

Skills don't carry `## Lessons Learned` sections — their execution signal lives in ticket `## Agent Log` entries. Mine it here.

1. For each skill in `+/Things/AI/Skills/`, query the 10–20 most recent tickets where that skill was invoked (look for the skill name in `## Agent Log`).
2. In each log, look for:
   - Explicit deviations: "had to skip step N because…", "step N was unclear…", "step N didn't apply here because…"
   - Repeated retries on the same step (≥2 attempts before moving on)
   - Agent-added notes flagging a gap in the procedure
3. Classify each signal: **procedural gap** (a step is missing or ambiguous), **scope gap** (skill doesn't cover a case it should), or **false positive** (one-off, not a pattern).
4. For each procedural or scope gap found in ≥2 tickets, draft the minimal edit to the relevant `SKILL.md`. Propose in the ADR — don't apply directly.
5. Record in the ADR: which skills were clean, which had signals, and which proposed edits were generated.

---

## (c) Coding-standard compliance

Spot-check the Senior Developer's recent commits/PRs against standing discipline.

1. List 3–5 merged PRs from the most recent quarter (`gh pr list --state merged` across `~/Documents/lang/*` repos).
2. For each PR diff, check:
   - **Abstractions:** new interface with one implementation, factory for one product, config for a value that never changes.
   - **Comments:** any comment restating what the code already says (non-obvious WHY comments are fine).
   - **Go-only backend:** new non-Go server code added to a backend that didn't already have it.
3. Classify each flag: one-off (note on the relevant ticket) or spec gap (draft a new `agent-kind:general` ticket in the ADR targeting the Senior Developer's spec).
4. Write findings grouped by PR, with specific diff lines as evidence.

---

## (e) Technical debt classification

Classify, quantify, and surface debt found during any audit scope — not as a separate pass, but as a running finding appended as each scope runs.

### Classification model

| Type | Definition | Priority signal |
|------|-----------|----------------|
| **Intentional** | Conscious trade-off taken at a known point in time, documented | Low — check if the payback date has passed; escalate if so |
| **Accidental** | Crept in without awareness; no deliberate decision | Medium — surfaces during drift review or postmortems |
| **Bit rot** | Was correct when written; now outdated due to dependency/platform drift | High — actively increases incident risk |

### SQALE-style quantification

For each debt item found, estimate:
- **Remediation time** (hours or days) — concrete, not vague
- **Delivery impact** — how many features or tickets is it currently blocking or slowing?
- **Incident risk** — does this increase the probability of a production failure? (yes/no + why)

A debt item with a 2-hour fix that blocks 3 features is more actionable than "needs refactoring." Quantify; don't just label.

### Severity scale for debt

| Severity | Criteria |
|---------|---------|
| S1 | Active incident risk; bit rot on a security-relevant path |
| S2 | Blocks delivery of planned tickets; accidental debt in a hot path |
| S3 | Slows development but doesn't block; intentional debt past payback date |
| S4 | Low impact; minor accidental debt; cosmetic |

### Debt visibility rule

Every debt item classified S3 or above must become a TaskNotes `agent-kind:general` or `agent-kind:coding` ticket before this audit ticket closes. Priority maps to severity: S1 → urgent, S2 → high, S3 → normal. Hidden debt is not actionable debt.

### Debt ADR entry format

```
**Debt: <short slug>**
Type: Intentional / Accidental / Bit rot
Location: <file or component>
Severity: S1–S4
Remediation: ~<hours/days>
Delivery impact: <description>
Incident risk: yes/no — <reason>
Ticket: [[link]] (created if S3+)
```

---

## (d) Stale-ticket sweep

Surface tickets stuck in limbo with no CTO action.

1. Query TaskNotes for `agent-queue` tickets with `status` in `needs-handoff`, `needs-approval`, or `blocked`. Sort by `dateModified` ascending.
2. For each ticket older than 30 days with no CTO input, read `## Agent Log` to understand why it's stuck. Categorize: **awaiting CTO decision**, **awaiting external input** (credentials, third-party access), or **iteration cap hit**.
3. Check `+/Things/AI/Skills/planner/assets/` — compare template field names against what recent tickets actually use. Note fields that disappeared from usage or fields tickets use but templates don't have.
4. Write a summary table to the ADR:

   | Ticket | Age (days) | Stage | Stuck reason | Suggested action |
   |--------|-----------|-------|-------------|-----------------|

   Suggested action is advisory — don't auto-resolve.

---

## Closing: Self-assessment

Runs at the end of every full audit pass, regardless of which scopes were active. This is how audit-ops improves its own procedures.

1. Review this audit run: which scope procedures worked cleanly, which required improvisation, which produced findings that the ADR format couldn't express well?
2. For each gap found in the audit-ops procedure itself, draft a minimal, specific edit to this `SKILL.md`. Treat this skill the same as any other — apply the same evidence standard (one-off vs. pattern).
3. Record proposed self-edits in a dedicated **Audit Process Improvements** section of the ADR:

   | Scope | Gap observed | Proposed edit to audit-ops SKILL.md |
   |-------|-------------|--------------------------------------|

4. The CTO approves or rejects proposed self-edits the same way as all other proposed changes — via a follow-on coding ticket. No self-edits are applied during the audit pass itself.

This step ensures that every audit pass leaves the system slightly better calibrated than it started — not just findings about other components, but about the audit process itself.
