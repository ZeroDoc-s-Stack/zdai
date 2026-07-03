---
name: test-strategy
description: Test strategy planning, coverage analysis, and shift-left ticket review. Use before a coding ticket is picked up by the Developer to verify it has testable success criteria. Use when evaluating test coverage across a project, planning what tests to write for a new feature, or deciding which testing layer (unit/integration/E2E) owns a given scenario. Invoke whenever asked to review a ticket for testability, assess coverage, or define what "done" looks like from a quality perspective.
---

# Test Strategy

Stateless skill for planning what to test, at which layer, and how to verify it's enough.

## Shift-left ticket review

Before the Developer picks up a `coding` ticket, QA reviews it for:

1. **Testable success criteria** — can `## Goal` be verified by running a specific command or checking a specific output? If not, flag it: "Goal is not verifiable as stated — add acceptance criteria."
2. **Edge cases missing from scope** — what inputs/states weren't mentioned? Empty inputs, auth failures, concurrent requests, large payloads.
3. **Test level assignment** — for each scenario, decide which layer owns it:

   | Scenario type | Test layer |
   |--------------|-----------|
   | Single function, isolated logic | Unit |
   | Multiple components interacting | Integration |
   | Full user journey, cross-service | E2E |
   | Production monitoring | Synthetic/shift-right |

4. **Measurable coverage target** — state a specific threshold: "this change should add ≥ 80% branch coverage on `pkg/auth`." Vague coverage goals are ignored.

If any of 1–4 are missing or insufficient: set ticket to `needs-rework` with specific questions. A ticket that can't be verified shouldn't be started.

## Test pyramid targets

| Layer | Target ratio | Owner | Speed |
|-------|-------------|-------|-------|
| Unit | 70% of test count | Developer | ms |
| Integration | 20% | Developer + QA | seconds |
| E2E | 10% | QA | minutes |

If the current project inverts this (heavy E2E, sparse unit) — flag it in the quality assessment. An inverted pyramid is fragile and slow.

## Coverage analysis for a project

1. Run the coverage tool for the stack:
   - Go: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
   - Frontend (Vitest): check `coverage/` output after `npm run test -- --coverage`
2. Identify uncovered packages/modules — list them with their coverage %.
3. Triage by risk:
   - **High risk, low coverage** → create test ticket (priority: high)
   - **Low risk, low coverage** → note in audit artifact, no immediate ticket
   - **High coverage, complex logic** → verify tests are meaningful (not just line-touching)
4. Write findings to the audit artifact or ticket `## Agent Log`.

## Regression scope for a change

When a PR touches file `X`:
1. Find all functions modified in `X`.
2. Identify all callers of those functions (one level up).
3. The regression test scope is: the modified functions + their callers + any E2E paths that exercise them.
4. Document this scope in the QA pass block (see `qa-testing` skill's reporting format).

## Definition of Done (quality gate)

A ticket is QA-approved when:
- The stated `## Goal` is met and manually verifiable
- The regression scope (defined above) passes
- Coverage did not decrease from baseline
- No new test skips or `t.Skip()` without a linked ticket explaining why
