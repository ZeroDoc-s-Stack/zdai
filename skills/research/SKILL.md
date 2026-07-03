---
name: research
description: Technical research and ADR-writing skill for agent-kind:research tickets. Use for investigating third-party API changes, dependency breaks, OSS repo analysis, system design surveys, or breaking-change checks. Delivers findings as a structured ADR — never applies changes directly. Invoke whenever a ticket asks for investigation, a "what changed", "should we use X", "is Y still maintained", or any question requiring research before a coding decision is made.
---

# Research Skill

Technical scout workflow for `agent-kind:research` tickets. Analytical, objective — reports what changed and what it means, not opinions.

## Steps

1. Read the ticket's `## Request` for the specific question. If it's actually a different agent-kind (`coding`, `email-approval`, `ui-design`, `hardware`, `api-consumer`, `audit`, `qa`, `sre`) described imprecisely, flag it in `## Agent Log`, set `stage: needs-approval` with a note to reclassify, and stop — don't silently recharacterize someone else's ticket.

2. Gather:
   - `WebFetch` / `WebSearch` for external docs, changelogs, API specs.
   - Local filesystem (`grep`, `find`, `go.mod`, `package.json`, etc.) across `~/Documents/lang/*` to ground findings in what's actually deployed, not just upstream docs.
   - Direct repo read (clone/read, never modify) for OSS analysis.

3. If the work touches a private/paid API needing a credential this agent doesn't have, use the credentials split (see template below): state exactly what's needed, set `stage: needs-approval`, never request broader scope than the question requires, never echo a supplied value back into the log or ADR.

4. Write the ADR (see format below), record its path in `artifact_path`, append a one-line summary to `## Agent Log`, set `stage: needs-handoff`.

5. If the question is too broad to answer confidently as posed (e.g. "look into GraphQL" with no concrete decision attached) — don't produce a padded ADR to look complete. Append a clarifying question to `## Agent Log`, set `stage: blocked`, and stop.

## ADR format

Write each ADR to `TaskNotes/AI/ADRs/<slug>.md`. After writing, check `TaskNotes/AI/Knowledge Index.md` and add or update the entry for this ADR.

```markdown
# ADR: <slug>

**Status:** proposed  
**Date:** <YYYY-MM-DD>  
**Ticket:** [[TaskNotes/Tasks/<ticket-title>]]

## Context

<What question was asked. What the current situation is. What's at stake.>

## Options considered

<Each option in one short paragraph — what it is, relevant tradeoffs.>

## Recommendation

<One clear recommendation with the key reason. Not a hedge.>

## Consequences

<What changes if this recommendation is followed. What gets harder, what gets easier.>
```

Keep ADRs short — context + options + recommendation + consequences, not a raw dump of search results.

Use Markdown footnotes to cite every source that informs a recommendation or decision. Internet sources: `[^1]: https://...`. Vault sources: `[^1]: [[Note Title]]`. Multiple sources get sequential numbers.

## Credentials template

```markdown
## Credentials Needed

Need a read-only API key for <service> to check <specific endpoint/limit>.
Provide via the ticket's `## Credentials` section below; this agent reads
it once, uses it for this ticket only, and never echoes it back into the
log or the ADR.

## Credentials

(CTO pastes the value here)
```
