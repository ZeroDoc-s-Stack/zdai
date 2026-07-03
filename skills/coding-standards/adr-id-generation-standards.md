# ADR: id-generation-standards

**Status:** proposed
**Date:** 2026-07-02
**Ticket:** N/A — direct CTO request (no TaskNotes ticket filed for this round)

## Context

CTO ask: establish an ID generation standard, ground it in a survey of ID
methods[^1], then set a default for the team's actual contexts — Go
services, SQLite-backed storage, and ticket IDs in Markdown frontmatter —
with justification. **This ADR sets the standard for new work only.** It
does not retrofit any existing ID. If retrofitting later looks valuable,
file a ticket in `TaskNotes/AI/Tickets` rather than doing it under this ADR.

Per this skill's own convention ("descriptive first, prescriptive second"),
the recommendation below is grounded in auditing the team's actual repos
under `~/Documents/lang/go/`, using the survey article only to name and
compare methods the team doesn't currently use.

## Method survey (per situation)[^1]

| Method | Generation | Properties | Best for |
|---|---|---|---|
| **UUIDv4** | Secure random, 128-bit | Practically collision-free, no coordination, not sortable, poor DB index locality at scale | General-purpose resource IDs, simplicity over performance |
| **UUIDv1** | Timestamp + MAC address | Time-sortable, but leaks machine identity | Rarely — the leak is a real drawback |
| **UUIDv5** | Deterministic hash of namespace + name | Same input → same UUID | Idempotent/derived IDs (e.g. dedup keys) |
| **ULID** | Timestamp + random, base32, 128-bit | Lexicographically sortable, human-readable, DB-friendly, larger than Snowflake | Logs, event streams, anything needing chronological order without a separate index |
| **Snowflake** | Timestamp + machine ID + sequence, 64-bit int | Compact, fast, time-sortable, but needs clock sync + machine-ID coordination | High-throughput distributed systems, microservices |
| **NanoID** | Secure random, custom length/alphabet | Short, URL-safe, not sortable | Public-facing/URL IDs, frontend-generated IDs |
| **Random/hash-based (crypto/rand, SHA-256)** | Cryptographic randomness or hash of input | High entropy, collision-resistant, not sortable, not index-friendly | Auth tokens, session IDs, password-reset links — security boundary, not a resource ID |
| **DB auto-increment** | Sequential integer from the database | Simplest, fastest, but single point of failure, predictable (enumeration risk), doesn't scale horizontally | Small single-DB internal tools only |

## Current state across repos (audit)

`github.com/google/uuid` (UUIDv4 via `uuid.New()`/`uuid.NewString()`) is
already the near-universal team default:

| Repo | Uses `google/uuid`? | Where |
|---|---|---|
| `zdworkflow/server` | yes | every model ID (`model.go`) — `uuid.UUID` typed columns throughout |
| `zdauth` | yes | resource IDs **and** session tokens (`uuid.New().String()` in `GenTokenCall`) |
| `zdapi` | yes (indirect) | — |
| `zdintegration` | yes (indirect) | sqlite + postgres repositories |
| `zdscraper` | yes (indirect) | — |
| `zdlib` | yes (indirect) | shared dependency baseline |
| `zdcli` | different lib (`nu7hatch/gouuid`) | TUI, not a service — out of scope, matches `adr-logging-standards`'s treatment of `zdcli` |

No repo uses ULID, Snowflake, NanoID, or DB auto-increment for
externally-visible IDs today. One gap found: `zdauth` uses UUIDv4 for
**session tokens**, not just resource IDs — the survey's own guidance[^1]
treats tokens as a distinct, security-sensitive category from resource IDs.
Noted here as a gap for future *new* token-issuance code to avoid, not a
retrofit of `zdauth`.

## Recommendation

1. **Default: UUIDv4 via `google/uuid` for all new Go/SQLite resource IDs
   and cross-service identifiers.** Matches practice already present in 6
   of 7 active Go repos, needs no new dependency, works identically as a
   SQLite/Postgres `TEXT` column, requires zero coordination between
   services. This is the same "shared, not duplicated" reasoning
   `adr-logging-standards` applied to `zdlib/base/logger` — don't introduce
   a second ID scheme when the existing one already covers the case.
2. **Exception — chronologically-queried, high-volume rows (execution
   steps, workflow node runs, anything with a "most recent N" access
   pattern): ULID instead of UUIDv4**, only when a service's query pattern
   genuinely needs sort-by-creation without a separate `created_at` index
   scan. Opt-in per repo when the pattern is real today — not applied
   speculatively anywhere right now.
3. **Ticket/artifact IDs in Markdown frontmatter (agent-queue tickets,
   ADRs, plans), if/when an explicit `id:` field is introduced: ULID.**
   Today TaskNotes identifies tickets by file path/title, not a numeric ID
   field, so this rule has no immediate code to apply to — it's forward
   guidance for the day one is added. ULID's lexicographic sort means a
   plain alphabetical sort of IDs matches creation order (useful for
   scanning `AI/Tickets/` by age), and its 26-char Crockford base32 form is
   shorter and more diff/eyeball-friendly in frontmatter than a 36-char
   UUID.
4. **Auth/session/reset tokens: crypto-random opaque tokens (`crypto/rand`
   or an equivalent secure generator), not UUIDv4**, for new token-issuance
   code going forward. UUIDv4 is cryptographically fine as randomness
   source, but a dedicated token generator keeps tokens visually and
   semantically distinct from resource IDs and matches the survey's
   security guidance[^1]. `zdauth`'s existing `uuid.New()`-based tokens are
   not touched by this ADR — no retrofit.
5. **Never use bare DB auto-increment for anything exposed outside a
   single service's internals.** No repo currently does this; documented
   as a guardrail against future regression, not a fix.

## Consequences

- No code changes required today — rule 1 formalizes existing practice.
- ULID adoption under rules 2/3 introduces a new dependency (e.g.
  `github.com/oklog/ulid/v2`) only for repos/fields that opt in; it is not
  added to `zdlib` speculatively.
- The `zdauth` UUIDv4-as-token gap (rule 4) is left in place; retrofitting
  it would be a security-relevant behavior change belonging in its own
  ticket, not silently bundled into this standards ADR.
- No existing ID of any kind — resource, token, or ticket — is changed by
  this ADR. If retrofitting is wanted later, file a ticket in
  `TaskNotes/AI/Tickets` referencing this ADR.

[^1]: https://akshatjme.medium.com/the-complete-guide-to-unique-id-generators-methods-explained-fb8bd3c3886f
