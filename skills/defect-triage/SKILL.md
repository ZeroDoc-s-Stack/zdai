---
name: defect-triage
description: Defect classification, severity assessment, and routing skill. Use when a test failure, regression, or unexpected behavior needs to be classified and turned into an actionable ticket. Invoke whenever QA verification fails, when a bug is reported by the CTO or SRE, or when deciding how urgently a defect needs fixing. Produces a defect ticket with correct severity, priority, and reproduction steps.
---

# Defect Triage

Stateless skill for classifying defects and routing them to the right place at the right priority.

## Severity scale

Severity describes the technical impact — independent of when it gets fixed.

| Severity | Label | Criteria |
|----------|-------|---------|
| S1 | Critical | Production down, data loss or corruption, security breach |
| S2 | High | Core feature broken, no workaround; or significant performance regression affecting users |
| S3 | Medium | Feature broken but workaround exists; or regression in non-critical path |
| S4 | Low | Cosmetic issue, minor inconvenience, documentation error |

## Priority scale

Priority describes when it should be fixed — determined by severity + context.

| Priority | When to assign |
|----------|---------------|
| Urgent | S1 always; S2 with active user impact |
| High | S2 without active impact; S3 blocking another ticket |
| Normal | S3 standalone; S4 with visible user impact |
| Low | S4 cosmetic |

Severity and priority are independent. An S3 in a demo-critical path may warrant `urgent` priority. An S2 in a deprecated feature may drop to `normal`.

## Triage steps

1. **Reproduce.** Confirm the defect is reproducible with specific, minimal steps. If you can't reproduce it, mark it `needs-rework` and request more context — don't triage an unconfirmed defect.

2. **Classify.** Is this:
   - **Regression** — worked before, broken now (trace to the introducing commit with `git bisect` or log comparison)
   - **New defect** — never worked, or in new code
   - **Environment issue** — works locally, fails on a specific host
   - **Spec ambiguity** — behavior matches code but not the expectation; the spec was wrong

3. **Assign severity and priority** per the tables above.

4. **Write the defect ticket** using the `planner` skill's ticket template, with these fields in `## Request`:

```markdown
## Defect Report

**Type:** Regression / New defect / Environment / Spec ambiguity
**Severity:** S<1–4>
**Introduced by:** <commit hash or ticket link, if known>
**Affects:** <service/component/file>

### Reproduction steps

1. <exact steps>
2. ...

### Expected behavior

<what should happen>

### Actual behavior

<what happens instead, with log output if available>

### Environment

<host, OS, Go version, relevant config>
```

5. **Route:**
   - S1/S2: assign `agent-kind:coding`, priority `urgent`/`high`, tag `agent-issue:bug`. Notify CTO immediately via `## Agent Log` on the originating ticket.
   - S3: create `agent-kind:coding` ticket, normal priority queue.
   - S4: create ticket, low priority, batch with next sprint.
   - Environment issue: create `agent-kind:sre` ticket instead.
   - Spec ambiguity: create `agent-kind:general` ticket targeting the spec, not the code.

6. **Link.** Add a wikilink from the defect ticket back to the QA verification ticket that caught it. Add `artifact_path` on the QA ticket pointing to the defect ticket.
