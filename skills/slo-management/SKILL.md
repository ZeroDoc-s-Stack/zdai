---
name: slo-management
description: SLO/SLI definition, error budget tracking, and alerting threshold management. Use whenever a service needs SLOs defined or reviewed, when an error budget needs calculating or checking, when alert thresholds need setting, or when deciding whether a change freeze should be triggered. Invoke at the start of a new service's reliability setup, during weekly reliability reviews, or when burn rate spikes.
---

# SLO Management

Stateless skill for defining, measuring, and enforcing service level agreements.

## Key concepts

- **SLI (Service Level Indicator):** The actual measured metric (e.g., % of requests returning 2xx in under 200ms).
- **SLO (Service Level Objective):** The target for that metric over a window (e.g., 99.9% over 30 days).
- **Error budget:** `1 - SLO target` expressed as allowed downtime or failure rate. The budget exists to be spent — it enables controlled risk-taking while staying honest.

## Standard SLIs for this stack

| Service type | SLI | Recommended SLO |
|-------------|-----|----------------|
| HTTP API (Go) | % requests with status < 500 and latency < 500ms | 99.5% over 30 days |
| Background job | % runs completing without error within deadline | 99% over 30 days |
| Nomad job | % scheduled allocations running healthy | 99.5% over 7 days |
| Sync (Syncthing) | % of time vault fully synced across devices | 99% over 7 days |

Adjust targets based on actual user impact — not all services have equal blast radius.

## Defining SLOs for a new service

1. Identify the user-facing behavior that matters most (availability, latency, correctness).
2. Pick 1–2 SLIs that directly measure it — resist the urge to track everything.
3. Set the SLO target conservatively first (lower than you think you need). Tighten it once you have baseline data.
4. Record the SLO in the service repo at `docs/slo.md` with: SLI definition, target, window, measurement source, and error budget calculation.

## Calculating the error budget

```
error_budget_minutes = window_minutes × (1 - slo_target)

# Example: 99.9% SLO over 30 days
window_minutes = 30 × 24 × 60 = 43,200 min
error_budget_minutes = 43,200 × 0.001 = 43.2 min/month
```

Track consumed budget as a running total. Report remaining budget as a percentage.

## Burn rate and alert thresholds

Fast burn (budget exhausted within hours) needs a page. Slow burn (budget exhausted within the window) needs a ticket.

| Burn rate multiplier | Meaning | Action |
|---------------------|---------|--------|
| > 14× | Budget exhausted in < 2 hours | Immediate incident — invoke `incident-response` |
| 2–14× | Budget exhausted in hours–days | Create high-priority ticket; consider change freeze |
| ≤ 2× | On track or slow burn | Monitor; log in weekly review |

Burn rate = (error rate observed) / (1 - SLO target)

## Change freeze trigger

Recommend a change freeze to the CTO (set `stage: needs-approval`) when:
- Error budget consumed > 80% for the current window, OR
- Burn rate > 2× for more than 6 consecutive hours

Freeze means: no new feature deployments to the affected service until budget recovers. Hotfixes for the active incident are exempt.

## Weekly reliability review

Each week, check per-service:
1. Current error budget remaining (%)
2. Burn rate over the past 7 days
3. Any SLO breaches or near-misses
4. Any toil that could be reduced

Append summary to `## Agent Log` on the SRE review ticket.
