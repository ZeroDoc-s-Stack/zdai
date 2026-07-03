---
name: coding-standards
description: Reference skill holding team's per-stack coding standards ADRs (Go, Svelte, Docker, testing). Use starting new project in known stack, reviewing PR convention compliance, before writing new standards ADR for additional stack. Invoke whenever asked "what's our convention X", "set up new Go/Svelte project", audit needs baseline check drift against.
---

# Coding Standards

Holds team's pragmatic, evidence-based coding standards — one ADR per
stack, each derived from auditing team's actual shipped repos rather than
generic best-practice lists. These are descriptive first (what we already do)
before prescriptive (what to do going forward); gaps between two are called
out explicitly in each ADR's Consequences section rather than papered over.

## ADRs in skill

| Stack | ADR | Status |
|-------|-----|--------|
| Go | [[adr-go-standards]] | accepted |
| Docker | [[adr-docker-container-standards]] | proposed |
| Testing (cross-stack) | [[adr-testing-standards]] | proposed |
| Logging (cross-stack, Go-focused today) | [[adr-logging-standards]] | proposed |
| ID generation (cross-stack) | [[adr-id-generation-standards]] | proposed |
| Svelte | (pending — ticket "Define Svelte project standards") | — |
| Philosophy (cross-stack) | [[pragmatic-programmer-principles]] | proposed |

## Quick reference: templates and guides

- **Go project template:** `templates/Dockerfile.go`
- **Svelte project template:** `templates/Dockerfile.svelte`
- **Multi-service compose template:** `templates/docker-compose.yml.template`
- **zdharness containerization plan:** `ZDHARNESS_CONTAINERIZATION.md`

## Usage

- **Starting new project in stack**: read stack's ADR, follow
  recommendation section's default layout/tooling choice. Copy templates from
  `templates/` folder. Deviating is fine but should be a conscious choice, not
  an accident — note it in the new project's README or CLAUDE.md.
- **Reviewing PR/audit drift**: compare repo's structure, test
  patterns, tooling to the relevant ADR's "Current state" table. Flag
  deviations as audit findings, not automatic rejections — some
  deviation is expected for small/throwaway projects.
- **Adding new stack's ADR**: follow the same format as `adr-go-standards.md`
  (Context / Current state across repos / Recommendation /
  Consequences). Never import generic internet style guides. Update the
  table above to link to the new ADR.

## Revising existing ADR

Per Researcher persona's ADR ownership model: never edit a published
ADR in place once `status: proposed` or later. If standards need to
change, write a new ADR with `status: supersedes [[old-adr-slug]]` and
flip the old one to `superseded by [[new-adr-slug]]`. This keeps the
reasoning behind earlier decisions visible instead of silently overwritten.
