---
name: incident-response
description: Cross-functional incident response skill. Use when a coding change causes a regression or breakage — whether caught by the Senior Developer's own verification, the Auditor's drift review, or reported by the CTO. Covers detection, fix-forward vs. rollback triage, rollback mechanics, notification, and stage flip. Invoke whenever a change needs to be unwound or a regression needs to be triaged.
---

# Incident Response

Cross-functional — not owned by any single persona. Runs when a change causes a regression or breakage.

## Steps

1. **Detect.** A regression surfaces one of three ways: the Senior Developer's own post-change verification fails (see the `qa-testing` skill), a later ticket touching the same target fails for a reason traceable to a prior change, or the CTO reports it directly. Whoever notices logs it against the *original* ticket that introduced the change — not the ticket that happened to surface it.

2. **Triage — fix-forward vs. rollback.** Default to fix-forward (smallest correct patch) when the cause is understood and the fix is small and low-risk. Roll back when any of:
   - the cause isn't understood within one focused attempt,
   - the affected system is currently broken for users/other tickets (rollback restores service immediately; root-causing can happen after),
   - the fix would itself be a destructive or infra-mutating operation (see [[HITL-Safety-Matrix]]) — don't compound risk under incident pressure.

3. **Rollback mechanics.** Revert via an ordinary commit (`git revert`, not history rewrite — destructive git ops are never allowed regardless of incident pressure, per [[HITL-Safety-Matrix]]). For an infra change (Terraform/Ansible/Nomad), rolling back means re-applying the prior known-good config via the `devops-sre` skill — if that re-apply hits a HITL gate, it still needs `needs-approval`, incident or not.

4. **Notify.** Append a log entry to the *original* ticket's `## Agent Log` describing the regression and the action taken. If rollback was used, also note it in the new fix-forward ticket (if one is opened) via its `## Request` so the Senior Planner has the full picture when re-scoping.

5. **Stage flip.** Set the original ticket's `stage: failed` and tag `agent-issue:bug` if the change was rolled back and won't be reattempted as-is (a fresh ticket will redo it). Set `stage: needs-approval` instead if the fix-forward patch needs CTO sign-off per [[HITL-Safety-Matrix]]. Never leave a known-broken change at `stage: done`.

6. **Don't retry past the cap.** Same iteration discipline as any coding ticket — if triage + fix-forward takes more than the ticket's remaining iteration budget (cap 5), stop and set `stage: blocked` rather than burning the cap mid-incident.
