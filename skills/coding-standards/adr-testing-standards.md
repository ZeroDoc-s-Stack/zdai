# ADR: testing-standards

**Status:** proposed
**Date:** 2026-06-29
**Ticket:** [[TaskNotes/Tasks/Define testing standards for developer persona]]

## Context

The "Create coding standards" plan's foundational ADRs (`adr-go-standards.md` and
`pragmatic-programmer-principles.md`) revealed testing as the highest-leverage
low-cost improvement opportunity across the codebase. This ADR applies those
findings to the developer persona's Definition of Done and execution order,
establishing:

1. When tests are mandatory vs. optional
2. Test structure and organization by project kind
3. Test-running gates in the developer's workflow
4. Integration with the harness-coordinator's quality checkpoints

This is the first "cross-stack" standards ADR (applies to Go, Svelte, Docker
tests uniformly) rather than a stack-specific one, because the developer's
**execution order** and **gate logic** are stack-agnostic — the *what to test*
is detailed below by stack; the *when and how to verify* is uniform.

## Findings from upstream ADRs

### From `adr-go-standards.md`

- **Gap found:** `zdapi` (production gateway) has near-zero HTTP handler test
  coverage, flagged as "a real risk, not a style nit."
- **Best practice demonstrated:** `zdharness` and `zdmigration` use table-driven
  subtests with `t.Run`, paired with doc comments explaining what is and isn't
  being tested.
- **Recommendation already in place:** new Go code must follow table-driven
  subtests pattern; no fixed coverage percentage yet.
- **Broken window:** zero linter config (`go vet`, `golangci-lint`) across 10+
  repos — cheap fix, currently missing everywhere.

### From `pragmatic-programmer-principles.md`

- **Tracer bullets principle:** starting lean is correct; not circling back to
  harden coverage (like `zdapi` didn't) is the failure mode.
- **Broken windows principle (highest-impact):** `zdworkflow/client` has no test
  runner installed (`vitest`/`@testing-library/svelte`) and no CI gate on
  `svelte-check` — these are cheap to add and normalize the current gap.
- **DRY for documentation:** the `zdmigration` README template should be reused
  across stacks for test documentation, not reinvented per project.
- **Pragmatism over dogma:** default to "new code gets tested," not "achieve
  80% coverage" — state the default explicitly, then allow exceptions under
  rare circumstances.

## Current state by project kind

| Project Kind | Test Coverage | Test Organization | CI/Makefile Gate | Gap / Risk |
|--------------|----------------|-------------------|-------------------|-----------|
| Go (new: `zdharness`, `zdscraper`) | Thin but present | Table-driven `t.Run` with doc comments | `make test` exists | None — these set new standard |
| Go (old: `zdapi`, `zdlib`, `zdcli`) | Near-zero to ~10% | Mix of basic functions and missing suites | `make test` exists but incomplete | `zdapi` is production-facing with near-zero handler coverage |
| Svelte (`zdworkflow/client`) | Zero | No test files exist | No `svelte-check` gate found in CI/Makefile | Broken window: no test runner installed; no check step |
| Docker (build-test layer) | N/A (infra concern) | Alpine multi-stage standard | Layer present | Not tested per se; verified via successful builds |

## Recommendation

### 1. Developer's test-writing requirements

**New code is mandatory to include tests. Bug fixes must include regression tests.** 
Define "tests" contextually:

#### Go projects

- **Unit tests:** Table-driven subtests with `t.Run` (per `adr-go-standards.md`
  recommendation #2). Minimum coverage: all exported functions with branching
  logic get at least one table case per branch.
- **Integration tests:** Not required in this ADR; documented separately if
  needed. Current team practice shows weak integration-test discipline;
  adding it here would conflict with the pragmatism-over-dogma principle.
- **HTTP handler tests (for services):** If a Go service exposes HTTP endpoints,
  at least one happy-path and one error-path test per handler. `zdapi` should
  retroactively adopt this (separate ticket); new services must do this
  from day one.
- **Coverage percentage:** No fixed target. `go test -cover` output is
  informational; future ADR may set a percentage if team decides it matters.

#### Svelte/frontend projects

- **Unit tests:** One test file per component (`.test.ts`/`.spec.ts` naming).
  Minimum: each exported prop/event/slot gets at least one test case.
- **End-to-end tests:** Optional at this ADR-writing time. `zdworkflow/client`
  currently has zero test infrastructure; adding E2E first would skip unit
  testing entirely. Recommend unit tests first, E2E as follow-up.
- **Test runner:** `vitest` (modern, Svelte-compatible, already in SvelteKit
  ecosystem). Do not invent per-project test tooling choices.
- **Check gate:** `svelte-check` must run in CI and `Makefile` (broken-windows
  fix). Tests must run before build succeeds.

#### Docker/infrastructure

- Test targets are the build step itself (layer build succeeds, final image
  has correct structure). Covered implicitly by successful `docker build`.
  No standalone test files needed; verify via build or runtime sanity check.

### 2. Test-first practice for bug fixes

Bug fixes are **code changes**, so they follow the same test requirement as new
features. Additionally:

1. **Reproduce before fixing:** Write a test that fails on the current code,
   demonstrating the bug. This test becomes part of the permanent suite.
2. **Fix and verify:** Make the minimal change to fix the bug. Re-run the test
   to confirm it passes.
3. **No skipped tests:** Do not use `t.Skip()` or `.skip()` for regression
   tests — they exist to prevent future regressions.

This is already the de-facto practice in newer repos (`zdharness`); document
it as mandatory here so older repos (`zdapi`) adopt it when their handlers
are hardened.

### 3. Test organization by project kind

#### Go services and libraries

```
<project>/
  cmd/              (if applicable)
  internal/         (if applicable)
  handler/
    handler.go
    handler_test.go         ← table-driven tests for each handler
    handler_integration.go  ← optional; for multi-handler flows
  logger/
    logger.go
    logger_test.go
  ...
```

Pattern: `*_test.go` lives in the same package as the code it tests.
Sub-packages get their own `*_test.go` files. No separate `tests/` folder.

#### Svelte/SvelteKit projects

```
<project>/
  src/
    routes/
      +page.svelte
      +page.test.ts           ← per-route unit tests
    components/
      Button.svelte
      Button.test.ts
    lib/
      utils.ts
      utils.test.ts
  vitest.config.ts            ← test configuration
```

Pattern: `*.test.ts` lives in the same folder as the component/lib/route.
All tests discovered via glob pattern in vitest config.

### 4. Test-running gates in developer's execution order

**Update developer persona's "Default execution order" section:**

```
For every coding ticket:
  1. ticket-queue (pick up)
  2. Implement the change
  3. Write tests (mandatory; see Definition of Done)
  4. Run `make test` (Go) or `npm test` (Svelte) — developer self-check
  5. security-scan (automatic; runs before next step)
  6. qa-testing (independent verification; QA persona)
  7. ticket-queue (close)
```

Tests must pass locally before security-scan is invoked. If a test fails,
fix the code/test and re-run; do not skip or commit broken tests.

**For bug fixes specifically:**

```
  1. Reproduce: Write failing test demonstrating the bug
  2. Fix: Minimal code change to make test pass
  3. Verify: Run full test suite; confirm no regressions
  4. Document: Note in CHANGELOG or commit message which bug is fixed
```

### 5. Update harness-coordinator Definition of Done

Add to the developer persona's success criteria (currently in
`developer-persona.md`):

```
stage: done when:
  - ## Goal fully met
  - Tests pass: all new code and bug fixes include tests
  - For bug fixes: regression test is included
  - security-scan passes (no new high/critical findings)
  - qa-testing passes
  - No outstanding questions
```

Enforce this via `ticket-queue` validation: if a developer-kind ticket
closes without tests, and the Goal requires them, flag as `needs-rework`
with reason "Tests missing or not documented."

### 6. Integration with harness-coordinator

The coordinator currently dispatches to `qa-testing` persona after developer
closes a ticket. The QA step includes independent test verification:

- QA runs the full test suite on the developer's branch
- QA verifies test organization matches this ADR (files in correct locations,
  naming conventions, at least one test per acceptance criterion)
- QA flags missing tests as a defect, routes to developer for rework

No change needed to coordinator logic; clarifying that this ADR establishes
the "what QA checks for" part of the quality gate.

## Consequences

- **New Go code:** Higher test bar immediately (table-driven subtests,
  doc comments on intent). Older code (`zdapi`, `zdlib`, `zdcli`) are
  grandfathered; if touched for other reasons, adopt new standard.
- **Svelte/frontend projects:** Must install `vitest` + `@testing-library/svelte`
  and commit at least one test file before merging. `zdworkflow/client` will
  need this retroactively (separate ticket).
- **Broken-windows fix:** Adding `go vet`, `golangci-lint`, and `svelte-check`
  CI gates is prerequisite for this standards ADR to be enforceable — these
  gate *quality* of tests, not just presence. Recommend the Go linting
  additions happen this cycle (cheap, high-impact); Svelte check gate
  can follow in next Svelte standards ticket.
- **Test coverage percentage:** Explicitly not mandated. `go test -cover`
  and `vitest --coverage` outputs are informational only. If team later
  decides a percentage target matters (e.g., 60% minimum), that's a new ADR
  superseding this one, not a hidden expectation.
- **Retroactive fixes:** `zdapi`'s near-zero handler coverage is documented
  in `adr-go-standards.md` Consequences as "separate, larger effort should
  become its own ticket if CTO wants it prioritized." This ADR doesn't
  retroactively mandate it.
- **Performance of test infrastructure:** `vitest` and `go test` are both
  fast enough for pre-commit hooks; no team is blocked by test performance
  concerns at current codebase size. Revisit if any test suite exceeds
  30s wall-clock time.

## Related decisions

- **`adr-go-standards.md` supersedes this if Go testing patterns change.**
  If that ADR updates testing recommendations, this one should be reviewed
  for consistency and updated if divergent.
- **Svelte and Docker standards ADRs** will build on this one's cross-stack
  philosophy (test-first for new code, broken-windows fixes first) but
  specify stack-specific tooling and patterns.
- **The developer persona's execution order** is the primary enforcement point.
  Any changes to it should be approved by CTO + team, not developer's own call.
