# ADR: go-standards

**Status:** accepted
**Date:** 2026-06-29
**Ticket:** [[TaskNotes/Tasks/Research current Go project standards]]

## Context

The "Create coding standards" plan needs a Go baseline before Svelte,
Docker, and cross-stack testing standards can be defined (those tickets
depend on this one). Rather than import a generic Go style guide, this ADR
is built by auditing what the team's own Go repos under
`~/Documents/lang/go/` actually do — `zdapi` (primary target), plus
`zdlib`, `zdcli`, `zdmigration`, `zdharness`, `zdintegration`, `zdscraper`,
`zdworkflow/server`, and `env` as corroborating evidence, since a single
repo isn't enough to separate "team convention" from "one-off choice."

**Scope note on zdauth:** the ticket asked for zdauth specifically, but
`zdauth` is not a local checked-out repo anywhere under `~/Documents/lang`
— it's consumed by `zdapi` purely as a remote go-micro dependency
(`github.com/zerodoctor/zdauth`, referenced in `zdapi/go.mod` and called
via generated proto client in `zdapi/handler/auth.go`). Its source was not
available to review. Findings below about cross-service/proto conventions
are inferred from the consumer side (zdapi) only. If zdauth's source
becomes available (e.g. cloned locally), this ADR should be revisited —
write a new ADR that supersedes this one rather than editing it in place.

**Duplicate ticket note:** a second, overlapping ticket exists —
`TaskNotes/Tasks/Research current Go project standards from zdapi and
zdauth.md` (status: `queued`, same plan tag `coding-standards`). This ADR
satisfies both; the CTO should close the duplicate to avoid double work on
the next harness cycle.

## Current state across repos

### 1. Repository structure and module organization

| Repo | Layout | internal/ used? |
|------|--------|------------------|
| `zdapi` | Flat: `cmd/main.go`, `handler/`, `logger/` at root | No |
| `zdworkflow/server` | `cmd/`, `internal/`, `sql/` | Yes |
| `zdmigration` | `cmd/`, `internal/`, `schema/` | Yes |
| `zdintegration` | `internal/provider/`, `internal/worker/` | Yes |
| `zdscraper` | `internal/bible/` | Yes |
| `zdharness` | Flat, single package `main` at root | No |
| `zdlib` | Flat domain packages: `collection/`, `convert/`, `errors/`, `file/`, `validate/`, `util/`, plus `base/` for infra (`grpc`, `logger`, `metric`, `secret`) | No (it's a library, everything is meant to be imported) |
| `zdcli` | Flat domain folders: `command/`, `config/`, `db/`, `generate/`, `alert/` | No |

**Pattern:** `internal/` is used specifically when a binary has multiple
packages that should not be importable by other modules (`zdworkflow`,
`zdmigration`, `zdintegration`, `zdscraper` — all multi-file apps with
real internal complexity). Flat, no-`internal/` layout is used for (a)
small single-binary services with few packages (`zdapi`: 3 packages total)
and (b) libraries meant to be fully importable (`zdlib`, `zdcli`). Module
names consistently follow `github.com/zerodoctor/<repo-name>` (one
exception: `zdmedia`, `zdlib` import path uses `zerodoc-s-stack` org for
some packages — org naming is not fully unified, a real inconsistency
worth flagging rather than hiding).

Package naming: short, lower-case, no underscores, named after the domain
they own (`handler`, `logger`, `collection`, `convert`) — never `utils`
or `common` as a dumping-ground package name (`zdlib/util` is the closest
exception and is itself deliberately minimal — `util.go` + `retry.go`
only).

### 2. Testing approach

Coverage across the audited repos is **thin and inconsistent** — this is
the most important finding, not a strength to emulate uncritically:

- `zdapi`: one test file (`handler/handler_test.go`), one test function,
  covering a single helper (`PathGroups`). No tests for the gateway's
  actual HTTP handlers, auth flow, or error paths.
- `zdlib`: 2 test files for ~19 source files (`collection/set_test.go`,
  `base/secret/hash/hash_test.go`).
- `zdcli`: 3 test files across a much larger package tree.
- `zdharness`: 2 test files (`routing_test.go`, `tasknotes_test.go`) —
  small repo, but these are the **best-quality tests in the whole
  audit** (see below).

Two distinct test styles coexist:

- **Older/most repos** (`zdapi`, `zdlib`): sequential assertions with
  `t.Fail()` / `t.FailNow()` and manual `fmt.Printf("[ERROR] ...")`
  messages, no subtests, no `t.Run`. Example, `zdapi/handler/handler_test.go`:
  ```go
  func TestPathGroups(t *testing.T) {
      path := BASE_PATH_V1 + "/auth/gentoken"
      wantGroups := []string{"api"}
      wantTTL := TIME_DAY
      gotGroups, gotTTL := PathGroups(path)
      if wantTTL != int(gotTTL) {
          t.Logf("ttl does not match [want=%d] [got=%d]", wantTTL, gotTTL)
          t.Fail()
      }
      ...
  }
  ```
- **Newer repo** (`zdharness`, `zdscraper/internal/bible`): proper
  table-driven subtests using `t.Run` and `t.Errorf`/`t.Fatalf`, plus a
  doc comment on the test function explaining *what is and isn't being
  exercised*. Example, `zdharness/routing_test.go`:
  ```go
  // TestBaseURLForModel verifies model-prefix routing in isolation: claude-*
  // models must stay on the direct headroom proxy, while google/* and any
  // other non-claude-* prefix must go through the OpenRouter-compatible
  // headroom-or endpoint. No HTTP calls are made — this only inspects the
  // string returned by baseURLForModel.
  func TestBaseURLForModel(t *testing.T) {
      cases := []struct{ name, model, want string }{
          {"claude opus", "claude-opus-4", headroomBaseURL},
          {"google gemini", "google/gemini-2.5-pro", headroomORBaseURL},
          ...
      }
      for _, tc := range cases {
          t.Run(tc.name, func(t *testing.T) {
              got := baseURLForModel(tc.model)
              if got != tc.want {
                  t.Errorf("baseURLForModel(%q) = %q, want %q", tc.model, got, tc.want)
              }
          })
      }
  }
  ```
  This is the pattern to standardize on going forward — it's strictly
  better (per-case failure isolation, readable diff output, self-documenting
  intent) and costs nothing extra to write.

No repo has a stated coverage target. `Makefile` `test` targets that exist
(`zdapi`, `zdmigration`) just run `go test -v ./... -cover` — `-cover`
prints a percentage but nothing gates on it.

### 3. Tooling integration

- **No linter config anywhere** — `find ~/Documents/lang/go -iname
  ".golangci*"` returns nothing across all 10+ Go repos. No `go vet` or
  `gofmt -l` step in any Makefile either.
- **No CI pipeline files** found (no `.github/workflows`, no `.drone.yml`)
  in any locally-checked-out repo, despite `zdapi`'s Makefile referencing
  a `release` branch and registry push (`make docker`, `make nomad-test`)
  — CI likely exists only in the remote GitHub repo / a Drone instance not
  mirrored locally, or is fully manual.
  identical structure: a Go-builder stage (`golang:X-alpine`) that sets
  `GOPRIVATE`/`GONOSUMDB`/`GONOPROXY` for the `zerodoctor`/`zerodoc-s-stack`
  private module orgs, copies `go.mod`/`go.sum` first for layer caching,
  then a minimal `alpine` runtime stage copying only the binary + certs +
  timezone data. `zdapi/Dockerfile` is representative.
- **Makefile-as-task-runner** is the universal convention: every repo with
  a `Makefile` exposes `build`, `test`, `run`/`dev`, and deployment targets
  (`nomad-dev`, `nomad-test`) that source a `.env` file and `export` Nomad
  variables before `nomad job run`. No repo uses a more sophisticated build
  tool (Taskfile, mage, bazel) — Make + shell is the standing choice.
- **Private module access** is handled identically everywhere via git
  `insteadOf` URL rewriting + `GOPRIVATE`/`GONOSUMDB`/`GONOPROXY` env vars
  for the `github.com/zerodoctor/*` and `github.com/zerodoc-s-stack/*`
  prefixes (see `zdapi/README.md` and `zdapi/Dockerfile`) — this is a load-
  bearing convention since multiple internal libraries (`zdgo-util`,
  `zdmiddleware`, `zdauth`, `zdemail`, `zdvolume`) are pulled as private
  GitHub modules, not vendored.
- **Secrets**: HashiCorp Vault AppRole login at startup (`zdapi/cmd/main.go`
  `loadEnv()`) reading a per-env KV path (`zdkey/<env>/zdapi`) and exporting
  results as process env vars — not `.env`-file-only secret management in
  production; `.env` files are for local/dev convenience and Nomad var
  plumbing only.

### 4. Documentation standards

Two tiers exist, and the gap between them is the clearest "before/after"
in this audit:

- **Minimal tier** (`zdapi/README.md`, most older repos): just private-
  module setup instructions and a couple of `make` deployment commands.
  No architecture explanation, no "why," no usage beyond copy-paste shell
  blocks.
- **Mature tier** (`zdmigration/CLAUDE.md`, `zdharness/README.md`): a
  short prose summary of purpose, an "Architecture" section walking through
  each source file and *why* it exists (not just what it's called), a
  worked example for non-obvious behavior (zdmigration's up/down ordering
  example), a "Known issue" section documenting a real limitation instead
  of hiding it, and exact `make` commands mapped to what they do. Example
  structure from `zdmigration/CLAUDE.md`:
  ```
  ## Architecture
  - `cmd/main.go` - entrypoint. Parses flags...
  - `internal/envgen.go` - `EnsureEnvFile(path)` generates...
  - `internal/migration.go` - core logic: ...
  ## Schema layout
  ## Adding a new user/service
  ## Running
  ## Known issue
  ```
  This pattern (file-by-file architecture walkthrough + rationale + known
  limitations) is the most replicable and valuable documentation
  convention found and should be the template for new project READMEs/
  CLAUDE.md files, Go or otherwise.

Inline comments are sparse in the older repos (zdapi has almost none
beyond what's needed to follow control flow) and purposeful in the newer
ones (zdharness's test doc-comments explain intent and explicitly state
what's *not* covered, e.g. "No HTTP calls are made"). No repo uses
`godoc`-style package documentation (`// Package x ...` comments) anywhere
audited.

### 5. Why these patterns work for this team (rationale)

- **Flat layout for small services, `internal/` for multi-package apps**
  is a low-ceremony rule that scales with actual complexity instead of
  cargo-culting `internal/` onto a 3-package gateway. It matches how the
  team actually grows projects: start flat, add `internal/` once a second
  consumer-facing concern appears.
  team's deploy target (Nomad + Consul + Vault, not Kubernetes) — pulling
  in a heavier build tool would add ceremony without matching the actual
  ops surface.
- **Vault AppRole over `.env`-only secrets in prod** fits a team running
  its own Nomad cluster with Vault already in the stack — it's the
  pragmatic choice given existing infra, not a generic "best practice."
- **The documentation gap is a real risk, not a style nit**: `zdapi` is a
  production-facing gateway with effectively no test coverage of its
  actual HTTP handlers and a README that only covers deployment. The
  `zdharness`/`zdmigration` tier shows the team already knows how to do
  this well on newer projects — the gap is about consistency over time,
  not capability.

## Recommendation

Adopt the following as the Go standard for new projects and as the bar to
raise older ones toward (not a blocking requirement to retrofit everything
immediately):

1. **Layout**: start flat (`cmd/`, plus domain packages at root). Introduce
   `internal/` only once the project has packages that must not be
   importable by other modules — don't add it preemptively.
2. **Testing**: use table-driven subtests with `t.Run` and a doc comment
   stating what is/isn't exercised, per the `zdharness/routing_test.go`
   pattern. Minimum bar for new code: every exported function with
   non-trivial branching gets at least one table-driven test. No fixed
   coverage percentage gate yet — `go test -cover` stays informational
   until a follow-up testing-standards ADR sets a number (this feeds the
   "Define testing standards for developer persona" ticket directly).
3. **Tooling**: adopt `golangci-lint` with a default/lenient ruleset and a
   `make lint` target in new repos — currently zero repos have this, so
   even a permissive baseline is a net improvement. Add `go vet` to `make
   test` immediately (zero-cost, currently missing everywhere).
4. **Documentation**: every new repo gets a CLAUDE.md or README following
   the `zdmigration` template — prose summary, file-by-file architecture
   section with rationale, usage/make-target mapping, and a "Known issues"
   section maintained honestly rather than omitted.
5. **Module/secrets conventions**: keep the existing `GOPRIVATE`-rewrite +
   Vault AppRole pattern — it's consistent, fits the infra, and isn't worth
   disrupting.
6. Unify the module org prefix (`zerodoctor` vs `zerodoc-s-stack`) the next
   time any of the affected repos (`zdmedia`, `zdlib`) get touched for
   other reasons — not urgent enough to justify a standalone migration
   ticket today, but worth a one-line note in that repo's next PR.

## Consequences

- New Go projects get a consistent, low-ceremony starting layout instead
  of each developer/agent guessing; the `internal/` decision becomes a
  one-line rule instead of bikeshedding.
- Adding `golangci-lint` and `go vet` to `make test` will surface latent
  issues in `zdapi`/`zdlib`/`zdcli` the first time it's run there — expect
  a small one-time cleanup pass per repo, not a blocking migration.
- Raising the testing bar going forward does not retroactively fix
  `zdapi`'s near-zero handler test coverage; that's a separate, larger
  effort and should become its own ticket if the CTO wants it prioritized
  — this ADR only sets the bar for new/touched code.
- This ADR is Go-only. The Svelte and Docker standards tickets that depend
  on this one can reuse the documentation-template recommendation (item 4)
  verbatim — it's stack-agnostic — but must independently verify tooling
  and testing conventions for their own ecosystems rather than assuming
  Go's choices transfer.
- If zdauth's source is later reviewed directly, expect this ADR to need
  a superseding revision specifically for cross-service/proto conventions,
  since that section was inferred rather than observed directly.
