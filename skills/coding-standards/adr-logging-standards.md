# ADR: logging-standards

**Status:** proposed
**Date:** 2026-07-02
**Ticket:** N/A — direct CTO request (no TaskNotes ticket filed for this round)

## Context

CTO ask: establish a logging standard, make lifecycle logging (startup +
shutdown) mandatory for every service, consolidate Go logging setup so it's
shared via `zdlib` instead of hand-rolled per repo, wire the standard into
the `developer` persona, then bring existing services into compliance.

This is a cross-stack ADR in spirit (like `adr-testing-standards.md`) but in
practice today the team's services are all Go, so the concrete rules below
are Go/logrus-specific. Per this skill's own usage note ("never import
generic internet style guides"), the recommendation below is grounded in
auditing the team's actual repos under `~/Documents/lang/go/`, not lifted
from the internet. [betterstack's logging guide][^1] is cited only where it
corroborates or sharpens a specific practice already found in the audit.

## Current state across repos

`zdlib/base/logger/logger.go` already exists and is the right foundation:
a shared `Log *logrus.Logger`, `Load()` (JSON formatter, `InfoLevel` —
prod), `LoadText(showCaller bool)` (colored text formatter, `DebugLevel` —
dev), and `GinLogger(...)` (Gin access-log middleware). The problem isn't a
missing library, it's that several repos never adopted it and instead
copy-pasted their own near-identical `logger` package.

| Repo | Logrus? | Uses `zdlib/base/logger`? | Startup log | Shutdown/completion log | Gap |
|---|---|---|---|---|---|
| `zdapp` | yes | yes | yes | yes, via `zdlib/util.OnExitWithContext` | none — **reference implementation** |
| `zdscraper` | yes | yes | yes | yes | none — compliant |
| `zdapi` | yes | **own duplicate `logger` pkg** | yes | dead code: `log.Info("shutting down...")` sits after the blocking `router.Run()` call, which only returns on a listener error, never on a shutdown signal | duplicate logger package; shutdown log never actually fires |
| `zdauth` | yes | **own duplicate `logger` pkg**, near byte-identical to zdapi's pre-fix version | yes | yes, via `zdutil.OnExit` (a *different* helper, from `zdgo-util` not `zdlib`) | duplicate logger package |
| `zdworkflow/server` | yes | yes | **missing** | **missing** | uses the shared logger but never calls it for lifecycle events |
| `zdmigration` | yes | **own duplicate `logger` pkg** | yes | **missing** (one-shot CLI, no completion line) | duplicate logger package; no completion log |
| `zdintegration` | yes | **no** — raw `logrus.New()` + manual `JSONFormatter` inline in `main()` | yes (ad hoc string) | **missing** | not on the shared setup at all; no zdlib dependency |
| `zdharness` | **no** — stdlib `"log"` | no | **missing** | **missing** (has an unrelated custom `runs.log` audit-trail file, which is a different mechanism and out of scope here) | not on logrus; no lifecycle logs |
| `zdcli` | n/a | n/a | n/a | n/a | out of scope — interactive TUI, not a service with a start/stop lifecycle |
| `zdmiddleware` | yes, already | separate concern: a colored Gin access-log formatter, not a logger-*setup* duplicate | n/a | n/a | out of scope — doesn't violate "shared setup," it's a different feature (request access logs) already on logrus |

The three duplicated `logger` packages (`zdapi`, `zdauth`, `zdmigration`)
are functionally near-identical: `logrus.New()` in an `init()`, a
`TextFormatter` with `ForceColors`/`FullTimestamp`, a `CallerPrettyfier`,
and `InfoLevel`/`DebugLevel` selected by `os.Getenv("ENV")`. This is exactly
`zdlib/base/logger.LoadText()`, reinvented three times with minor
formatting drift (e.g. differing `CallerPrettyfier` string layouts) — a
concrete instance of "shared, not duplicated" logging setup.

`zdapp`'s shutdown idiom — a goroutine running
`util.OnExitWithContext(ctx, func(s os.Signal, i ...any) { log.Warn(...); log.Info("shutting down...") })`
started before the blocking `Run()`/`router.Run()` call — is the existing
convention to replicate for services missing it. Note it logs on signal
receipt but does not itself force the blocking listener call to return;
that's a pre-existing characteristic of this pattern across the codebase,
not something this ADR changes (fixing it would be a runtime/reliability
behavior change, not a logging one).

## Recommendation

1. **Logrus only, for every Go service.** No new logging library gets
   introduced (`slog`, `zap`, `zerolog`, etc.) even though betterstack's own
   benchmark notes logrus costs ~20% more throughput than `slog`[^1] — this
   team has already standardized on logrus via `zdlib`, and introducing a
   second logging library would itself violate "shared, not duplicated."
   Treat the perf note as a known, accepted tradeoff, not a reason to
   diverge.
2. **One shared setup: `zdlib/base/logger`.** No repo defines its own
   `logger` package. Services call `logger.Load()` (prod) or
   `logger.LoadText(showCaller)` (dev), matching `zdapp`'s
   `if env == "" || env == "local" { logger.LoadText() }` pattern. Existing
   duplicate `logger` packages (`zdapi`, `zdauth`, `zdmigration`) are
   deleted, not deprecated-in-place.
3. **Every service logs the start and end of its lifecycle.** Two new
   shared helpers added to `zdlib/base/logger`:
   ```go
   func LogStart(service string) {
       Log.Infof("starting %s [env=%s] [commit=%s]...", service, os.Getenv("ENV"), os.Getenv("GIT_COMMIT"))
   }
   func LogStop(service string) {
       Log.Infof("%s stopped", service)
   }
   ```
   Long-running services call `LogStart` at the top of `main()` and call
   `LogStop` from the existing `OnExitWithContext` signal-handler goroutine
   (the `zdapp` idiom). One-shot CLI tools (`zdmigration`, `zdharness`) call
   `LogStart` at the top of `main()` and `LogStop` (or an equivalent
   completion line) right before returning/exiting — no signal handling
   needed since the process life span *is* the work.
4. **Never log secret values, only their location.** Cross-checked against
   the audit: `zdapi`/`zdauth`'s `loadEnv()` already only logs the Vault KV
   *path* (`"reading secrets...zdkey/%s/zdapi"`), never the values —
   betterstack's citation of real leaked-secret incidents[^1] is why this
   stays a hard rule, not just a nice-to-have. Nothing to fix here; call it
   out so future changes don't regress it.
5. **Structured fields over freeform strings** for anything beyond a
   simple lifecycle line — already team practice via `logrus.Fields` in
   `GinLogger`/`GinLogrus`; no change needed, just documented as the
   standard.
6. **Level discipline**: `InfoLevel` default in prod, `DebugLevel` in
   dev/local, selected by `ENV` — already universal team practice via
   `Load()`/`LoadText()`; documented, not changed.

## Consequences

- Three duplicate `logger` packages (`zdapi`, `zdauth`, `zdmigration`) are
  deleted; those repos take on an explicit `zdlib` dependency (`zdintegration`
  gains one for the first time).
- `zdharness` is the one repo that needs an actual library swap (stdlib
  `log` → shared logrus `Log`) rather than just adopting a shared setup it
  already half-uses.
- `zdmiddleware`'s `GinLogrus` (a separate, colored access-log formatter,
  already on logrus) is left untouched — it's a different feature from the
  lifecycle/setup logging this ADR targets, not a violation of it. Worth a
  follow-up ADR someday if the team wants one canonical Gin access-log
  middleware instead of two (`zdlib.GinLogger` and `zdmiddleware.GinLogrus`),
  but that's a bigger, separately-scoped consolidation.
- This ADR does not touch `zdcli` (interactive TUI — no service lifecycle
  to log) or attempt to fix the pre-existing "shutdown log fires but the
  blocking listener doesn't actually stop" characteristic of the
  `OnExitWithContext` pattern — that's a reliability concern for a future
  SRE-owned ADR, not a logging one.

[^1]: https://betterstack.com/community/guides/logging/logging-best-practices/
