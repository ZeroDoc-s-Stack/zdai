---
name: email-notify
description: Email the CTO (danielcastrolocal@gmail.com) when a ticket needs attention — handoff review, approval gate, or blocker. Use whenever stage flips to needs-handoff, needs-approval, or a cycle is stalled on a human decision.
---

# Email Notify

Creates a Gmail draft to the CTO. **Note:** Gmail MCP creates drafts only; the email sits in Drafts until sent.

## When to invoke

- Stage flips to `needs-handoff` (research/coding complete, CTO review required)
- Stage flips to `needs-approval` (ambiguity or design decision needed)
- Ticket is blocked on a human dependency and cycle cannot proceed

## Tool call

```
mcp__claude_ai_Gmail__create_draft({
  to: ["danielcastrolocal@gmail.com"],
  subject: "[zdharness] <Ticket Title> — <state>",
  body: "<body — see template below>"
})
```

## Subject format

`[zdharness] <Ticket Title> — needs-handoff`
`[zdharness] <Ticket Title> — needs-approval`
`[zdharness] <Ticket Title> — blocked`

## Body template

```
Ticket: <TaskNotes path>
State:  <needs-handoff | needs-approval | blocked>
Agent:  <researcher | developer | qa | sre | planner>

<One paragraph: what was completed or attempted, what artifact was produced
(with path), and exactly what action you need from me.>

Artifact: <artifact_path if applicable, else omit>
```

## Example

Subject: `[zdharness] Research current Go project standards — needs-handoff`

```
Ticket: TaskNotes/Tasks/Research current Go project standards.md
State:  needs-handoff
Agent:  researcher

Go standards ADR complete. Reviewed zdapi plus 6 cross-reference repos.
ADR is at +/Things/AI/Skills/coding-standards/adr-go-standards.md.
Please review and set ticket status to `approved` (accept) or `needs-rework`
(with feedback in ## User Input) to unblock the downstream coding tickets.

Artifact: +/Things/AI/Skills/coding-standards/adr-go-standards.md
```
