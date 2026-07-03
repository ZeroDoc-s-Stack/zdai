---
name: planner
description: Planning skill for turning a CTO ask or agent-request ticket into a reviewed plan and agent-queue tickets. Use when scoping a raw work request, creating a plan note, or generating tickets from an approved plan. Bundles the ticket and plan templates — always use these when creating new plans or tickets rather than inventing fields. Invoke whenever the task is "plan this", "scope this", "create tickets for X", or any agent-request ticket with no plan yet.
---

# Planner Skill

Turns a CTO ask into a checkable, ticket-sized plan. Never executes a ticket itself.

## Templates

Both templates live in `assets/` alongside this skill. Always use them — don't invent frontmatter keys or section names. The templates define the standard structure for all planning work, including an `agile-point:` field in `## Agent State`, set at ticket creation time.

- **`assets/ticket-template.md`** — use for every new `agent-queue` ticket
- **`assets/plan-template.md`** — use for every new plan note under `TaskNotes/AI/Plans/`

## Drafting a plan

1. Read the raw ask in full. Break it into atomic, ticket-sized items. Each item gets:
   - a working title
   - a target (project/repo/note/host/API)
   - an `agent-kind` (`coding` / `email-approval` / `ui-design` / `hardware` / `api-consumer` / `general` / `research` / `audit`)
   - a one-line goal (the checkable success condition)
   - an **agile-point estimate** (1, 2, 3, 5, 8, 13 — Fibonacci scale)

2. An ask that's obviously one atomic thing still gets a one-item plan — never skip drafting because it looks trivial.

3. Split into multiple tickets when work spans independent units (different kinds, different targets, independent failure/completion). Keep it one ticket when it's genuinely one atomic change.

4. Write the numbered list into a `## Draft Plan` section:
   - **Live chat:** present in conversation and wait for verbal yes before creating anything.
   - **`agent-request` ticket:** write directly into the task body and leave `status` for the CTO to flip (`approved`/`ready` = proceed, `needs-rework` = revise).

5. Never finalize into tickets until the CTO has given a go-ahead on this pass.

## Creating tickets from an approved plan

Once approved, create one TaskNotes task per plan item using `assets/ticket-template.md`. Fill in:
- `title` from the plan item's working title
- `tags`: include `agent-queue` and `agent-kind:<kind>`; add `plan:<slug>` linking to the plan note
- `agent-kind` and `agile-point` in `## Agent State`
- `## Goal` with the checkable success condition from the plan item

Create the plan note using `assets/plan-template.md` under `TaskNotes/AI/Plans/`. Create ticket notes under `TaskNotes/AI/Tickets/`. Link each ticket from the plan's `## Tickets` section with its point estimate.

After creating the plan note, check `TaskNotes/AI/Knowledge Index.md` and add an entry if any new decision or constraint was established.

## Definition of Ready (DoR) gate

Before setting `stage: queued` on any ticket, verify all six criteria:

- [ ] Goal is a verifiable outcome (not a task description — "X works" not "do X")
- [ ] Acceptance criteria are explicit — QA can write test cases from them without clarifying questions
- [ ] Dependencies on other tickets are identified and linked (`blocks:` field set)
- [ ] External dependencies (credentials, third-party access, CTO decisions) are either resolved or blocked with `stage: needs-approval`
- [ ] `agile-point:` estimate is set
- [ ] `agent-kind:` tag is assigned

If any criterion is unmet: leave `stage: draft`, note the missing criteria in `## Goal` or `## Agent Log`, and surface to the CTO.

## Dependency mapping

After creating all tickets for a plan, map the dependency graph before any ticket enters the queue:

1. For each ticket, set `blocks: [[ticket]]` for every ticket that cannot start until this one is done
2. List external blockers explicitly in `## Goal`: credentials, third-party access, or CTO go-ahead not yet given
3. Check for circular dependencies — A blocks B blocks A. These require scope renegotiation; never schedule around them
4. Sequence the queue so no ticket enters `stage: queued` while blocked by an unstarted predecessor

Dependency mapping is the last step before handing the plan to the CTO for approval.

## Agile-point estimation

Base estimates on:
- **Unknowns** — spike-shaped work (needs investigation before scoping) scores higher
- **Systems touched** — more blast radius = higher points
- **External dependencies** — blocked on CTO credentials or decisions = add points

Don't re-estimate after the fact. Estimates are planning-time artifacts; actual-vs-estimated drift is logged on the ticket's `## Lessons Learned` by the executing agent, not edited here.

## HITL gates

When a draft plan item itself needs CTO sign-off before it can be scoped further (e.g. a frontend framework decision), set `stage: needs-approval` on the plan item and stop — same gate discipline as [[HITL-Safety-Matrix]].

## Multi-interface awareness (forward-looking, not yet wired)

Not yet connected: Plane (secondary ticket tracker), GitHub/Forgejo (open PRs into planning context), Gmail (customer feedback), Google Calendar (release commitments), WhatsApp (feedback channel). Until connected, work from whatever context the CTO supplies directly.
