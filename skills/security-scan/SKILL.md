---
name: security-scan
description: Static security analysis skill for code changes. Use after every coding change before setting stage:done — runs language-appropriate security scanners, reports findings by severity, and gates the ticket on high/critical results. Invoke whenever a PR is about to be opened, when a dependency is added or upgraded, or when asked to audit a codebase for vulnerabilities. Never optional — security scanning is a default step in the Developer's execution order.
---

# Security Scan

Static security analysis for this team's stack. Runs after implementation, before verification. A clean scan is a prerequisite for `stage: done` — not a nice-to-have.

## Scan commands by stack

| Stack | Tool | Command |
|-------|------|---------|
| Go | `govulncheck` | `govulncheck ./...` |
| Go | `gosec` | `gosec -fmt sarif ./...` (install: `go install github.com/securego/gosec/v2/cmd/gosec@latest`) |
| Go dependencies | `go mod tidy` check | `go mod verify` — flags tampered dependencies |
| Svelte / Node | `npm audit` | `npm audit --audit-level=moderate` |
| Svelte / Node deps | `npm audit fix` | only with explicit CTO approval — can silently break things |
| Ansible | `ansible-lint --profile security` | if ansible-lint installed |
| Terraform | `tfsec` | `tfsec .` (install: `brew install tfsec` or `go install`) |
| Secrets in any file | `trufflesecurity/trufflehog` | `trufflehog filesystem --directory=. --only-verified` |

Run the relevant tool(s) for the stacks touched by the current ticket. Don't run all tools on every ticket — scope to what changed.

## Severity triage

| Finding level | Action |
|--------------|--------|
| Critical / High | Block ticket — set `stage: needs-approval`, log the finding, surface to CTO before proceeding |
| Medium | Log in `## Agent Log`, create a follow-on `agent-kind:coding` ticket tagged `agent-issue:security`, proceed with current ticket |
| Low / Informational | Log briefly in `## Agent Log`, no blocking action |

Never suppress or filter out findings to make the scan appear clean — log everything found, then triage per the table above.

## Pre-existing findings

Before blocking on a finding, check if it was pre-existing:
1. `git stash` the current changes
2. Re-run the scan on the base branch
3. If the finding exists on base too — log it as pre-existing, create a follow-on security ticket, and don't block the current ticket for it

Only new findings introduced by this ticket's changes are blocking.

## Dependency additions

Any time a new dependency is added (`go get`, `npm install`, etc.):
1. Run `govulncheck` / `npm audit` immediately after adding
2. Check the dependency's last commit date and open issue count (a dependency unmaintained for >12 months is a risk flag — log it)
3. Verify the license is compatible with this team's projects

## Log format

Append to `## Agent Log`:

```
Security scan — <date>
Tools: <tools run>
New findings: <count by severity, or "none">
Pre-existing: <count, or "none">
Blocking: yes (see: <finding summary>) / no
Follow-on ticket: [[link]] / none
```
