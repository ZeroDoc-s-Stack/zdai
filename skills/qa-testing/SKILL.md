---
name: qa-testing
description: Independent test execution skill. Use for running E2E, integration, and syntax-verification loops on any code change — separate from the developer's own self-check. Invoke whenever a coding ticket is ready for independent verification, when a regression needs reproducing, or when a CI failure needs diagnosis. Also use when asked to verify any change across ~/Documents/lang/* projects before a PR is opened or merged.
---

# QA Testing

Independent verification of code changes. Reads the change fresh and runs the actual test suite — not just type-checking.

## What counts as verified

- The target project's existing test suite passes (not just type-checks or lints).
- If the ticket describes a specific behavior, that behavior is manually exercisable and works.
- No adjacent behavior visibly regressed (run the full suite, not just tests near the change).

Passing type-checking or a build step alone is **not** verification. State explicitly in the log if the project has no test suite — don't silently skip and claim success.

## Per-stack test commands

| Stack | Command | Notes |
|-------|---------|-------|
| Go | `go test ./...` | Add `-race` if the project uses goroutines |
| Go (vet) | `go vet ./...` | Run after test pass, not instead of |
| Svelte / frontend | check `package.json` `"test"` script | May be Vitest, Playwright, or absent |
| Lua scripts | check for `_test.lua` or inline `assert` blocks | Run with `lua <file>` |
| Ansible | `ansible-lint` if installed; `--check --diff` dry-run | |
| Terraform | `terraform validate` + `terraform plan` | |

If the project has no test script, note it in `## Agent Log` and do a best-effort manual smoke test of the specific change instead.

## Scope per ticket

Read the ticket's `## Request` and `## Goal` before running anything. Verify the exact behavior the ticket describes, plus the surrounding area. Don't just run `go test ./...` and call it done if the ticket is about a specific endpoint — exercise that endpoint too.

## Reporting

Append one structured block to `## Agent Log`:

```
QA pass — <date>
Suite: <command run>
Result: PASS / FAIL
Coverage note: <what was exercised beyond the suite, or "suite only">
Regressions: none / <description if any>
```

If the suite fails, report the failing test name and the error output (truncated if long). Don't edit or suppress test output — surface it verbatim so the Senior Developer and CTO can read the actual failure.

## On failure

If a test fails:
1. Determine if the failure is pre-existing (existed before this ticket's changes) or newly introduced.
2. Check with `git stash` + rerun to confirm — if the failure exists on the base branch too, log it as pre-existing and note it doesn't block the current ticket's merge.
3. If newly introduced by this ticket: invoke `defect-triage` to classify and file a defect ticket. Set this ticket to `stage: needs-rework`. Don't attempt the fix — verification, not correction.

## Ticket state flow

```
stage: queued → [run suite + manual check] → stage: done (PASS) or stage: needs-rework (FAIL → defect-triage)
```
