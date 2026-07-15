# zdai

Claude Code agent-harness service. Runs a background dispatch scheduler and
exposes a gRPC API (go-micro v5) for triggering agent runs, querying run
history, and email-driven ticket unblocking.

## Architecture

- **Scheduler** — fires at :07/:22/:37/:52 every hour (08:00–22:00 local);
  dispatches eligible `agent-queue` and `agent-request` vault tasks.
- **Email routing** — polls registered Gmail threads for replies on `blocked`
  tickets; transitions them to `in-progress` and redispatches.
- **Tess** — daily note trigger via `claude --agent tess`.
- **gRPC API** — go-micro v5 service registered in Consul.

State is persisted in `$STATE_DIR` (default `~/.local/state/zdai`):
- `runs.log` — agent invocation log (5 MB cap, rotates to `.1`)
- `zdai-state.json` — harness config (model, effort, Tess schedule, email routing)
- `email-thread-snapshots.json` — Gmail thread↔ticket mappings
- `run.lock` — single-instance flock

## Build / Run / Test / Deploy

```sh
# Build binary
make build

# Run (requires .dev.env or vault env vars)
make run

# Test (vet + unit tests)
make test

# Deploy to prod Nomad cluster
make deploy
```

See `Makefile` for all targets including `proto` (regenerate gRPC stubs) and `vendor`.

## Configuration

Copy `.env.example` to `.dev.env` and fill in values. Secrets are loaded from
Vault (`zdkey/<env>/zdai`) via AppRole at startup.

Key env vars:

| Var | Description |
|-----|-------------|
| `VAULT_ADDRESS` | Vault server URL |
| `APPROLE_ID` / `APPROLE_SECRET` | Vault AppRole credentials |
| `VAULT_DIR` | Obsidian vault root (default `/mnt/local/syncthing/data1`) |
| `STATE_DIR` | State directory (default `~/.local/state/zdai`) |
| `MICRO_PORT` | go-micro gRPC port (default 3001) |
| `CONSUL_ADDRESS` | Consul agent address |

## Container

```sh
# Build image (uses Dockerfile, deps vendored)
podman build -t zdai:latest .

# Run locally (mounts vault read-only, state rw, Claude session rw)
podman run --rm \
  -v /mnt/local/syncthing/data1:/vault:ro \
  -v ~/.local/state/zdai:/state \
  -v ~/.claude:/root/.claude \
  -e ANTHROPIC_API_KEY \
  -e VAULT_ADDRESS \
  -e APPROLE_ID \
  -e APPROLE_SECRET \
  --network=host \
  zdai:latest
```

`GITHUB_TOKEN` is required on first run when `/root/.claude/agents/` is not
present — the entrypoint clones `ZeroDoctor/zdclaude` to populate it.

## Project Layout

```
cmd/zdai/           entry point (main + micro setup)
internal/
  controllers/      gRPC request handlers
  logger/           shared logrus instance
  models/           data types (Run, RunRecord, etc.)
  services/         business logic (dispatch, scheduler, email routing, vault scanning)
package/grpc/       generated gRPC stubs (public, importable by callers)
nomad/              Nomad job specs (zdai.hcl prod, zdai.test.hcl test cluster)
.woodpecker/        Woodpecker CI pipeline definitions
```

## CI / Deploy

Woodpecker pipelines in `.woodpecker/`:
- `test.yaml` — vet + unit tests on every push to main/release
- `build-amd64.yaml` / `build-arm64.yaml` — multi-arch container builds
- `deploy.yaml` — manifest push + `nomad job run nomad/zdai.hcl` (main branch only)

Required Woodpecker org secrets: `github_user`, `github_token`, `docker_user`,
`docker_pass`, `zdai_id`, `zdai_secret`, `vault_address`.
