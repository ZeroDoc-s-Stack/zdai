---
name: devops-sre
description: Infrastructure operations skill for this team's stack. Use for any Ansible, Terraform, Nomad, Consul, Vault, or Cloudflare work. Enforces dry-run-before-apply discipline and HITL gates for all mutating operations. Invoke whenever a ticket involves infra changes, host provisioning, secrets rotation, DNS/CDN config, or any operation against a live environment.
---

# DevOps/SRE Skill

Infrastructure operations for this team's stack. The core discipline: **plan before you apply, gate before you mutate.**

## Host resolution

All target host names resolve via the Ansible inventory:

```
~/Documents/lang/ansible/zdansible/.inventory.toml
```

Short names (`zp0dune`, `vp0dune`, `mdune`, etc.) map to the actual host there. Never guess a hostname — read the inventory. If a ticket names a host not in the inventory, set `stage: needs-approval` and ask the CTO before proceeding.

## Stack and tooling

| Tool | Purpose | Dry-run command |
|------|---------|----------------|
| Ansible | Host provisioning, config mgmt | `ansible-playbook --check --diff` |
| Terraform | Cloud infra state | `terraform plan` |
| Nomad | Workload scheduling | `nomad plan` |
| Consul | Service discovery / KV | read-only queries before any write |
| Vault | Secrets management | `vault read` before any write/rotate |
| Cloudflare | DNS, CDN, WAF rules | preview via API dry-run where available |

## Run discipline

**Always run the dry-run first.** Record the output (or a summary of it) in the ticket's `## Agent Log`. If the dry-run output contains unexpected diffs — things the ticket didn't mention — stop, log the discrepancy, set `stage: needs-approval`, and surface it to the CTO. Don't apply a change whose plan shows surprises.

**HITL gates (per [[HITL-Safety-Matrix]]):**

| Operation | Gate |
|-----------|------|
| Dry-run / plan / `--check` | None — proceed |
| `terraform apply`, `ansible-playbook` (no `--check`) | `needs-approval` before running |
| Nomad job run / Consul KV write / Vault write or rotate | `needs-approval` before running |
| Cloudflare DNS / WAF change | `needs-approval` before applying |

After CTO approval (`status` flipped to `approved`/`ready`), run the mutating command. Log the result in `## Agent Log`. If the apply fails, invoke the `incident-response` skill — don't retry blind.

## Vault / secrets handling

Read secrets only when the ticket genuinely requires them. Never echo a secret value into `## Agent Log` or any other field. If a rotation or new secret is needed and the scope requires CTO-held credentials, follow the `## Credentials Needed` / `## Credentials` split from `Harness/playbooks/api-consumer.md`.

## Ansible specifics (zdansible)

Playbooks live at `~/Documents/lang/ansible/zdansible/`. Role structure follows the repo's existing conventions — read the existing roles before writing new tasks. Variable files: `group_vars/`, `host_vars/` per standard Ansible layout. `vars.secrets.yaml` is credential-bearing — read only when directly needed, never log values.

## Terraform specifics

State is remote — don't assume local `.tfstate` is authoritative. Run `terraform init` if the workspace is fresh. Always run `terraform plan -out=tfplan` and apply the plan file rather than re-planning at apply time, so what gets applied is exactly what was reviewed.

## Nomad / Consul / Vault specifics

Job definitions live alongside their service repos under `~/Documents/lang/<service>/`. Check the existing job spec before modifying — Nomad job names and task group names are how Consul service registrations are named; changing them without coordinating the service discovery side breaks other services.

Vault paths follow `<mount>/<service>/<key>` convention — read existing paths before writing new ones to avoid orphaned secrets.

## Ticket state flow

```
stage: queued → [dry-run] → stage: needs-approval → [CTO approves] → [apply] → stage: done
                                                                                 or stage: failed (log why)
```

If the apply succeeds, append the result summary to `## Agent Log` and set `stage: done`. If it partially succeeds (some tasks changed, some failed), treat it as failed — don't leave live state in an ambiguous half-applied condition without surfacing it.

