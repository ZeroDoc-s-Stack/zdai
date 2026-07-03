# zdharness Containerization Plan

**Status:** Phase 1 (Foundation) — to be implemented after Docker standards ADR approval

See also: [[adr-docker-container-standards.md]] — "zdharness containerization" section for architecture context.

## Current state

`zdharness` is an OS-level scheduler binary that:
- Runs as a systemd user timer (`~/.config/systemd/user/zdharness.timer`)
- Fires once per cycle and exits immediately (no daemon mode)
- Reads harness prompt from `.cron-state.json` in the Obsidian vault
- Invokes `claude --print` headlessly via subprocess
- Logs results to `~/.local/state/zdharness/runs.log`

## Architecture constraints for containerization

1. **Vault access required:** zdharness reads `.cron-state.json` at
   `/mnt/v1drive/syncthing/data1/+/Things/AI/Harness/.cron-state.json` —
   must be bind-mounted into container or accessible via network.

2. **Claude CLI dependency:** zdharness shells out to `claude --print`; the
   Claude binary or its socket/API endpoint must be available inside the
   container.

3. **State persistence:** runs.log grows across invocations and must survive
   container restarts (named volume or bind mount).

4. **Timer orchestration:** the container doesn't schedule itself; systemd
   timer on the host invokes `podman run ...` once per cycle (stateless invocation).

5. **No networking required:** zdharness is a local orchestrator, not a networked
   service. Network isolation is not a goal.

## Phase 1: Build Dockerfile and mount configuration

### 1.1 Dockerfile for zdharness

Location: `zdharness/Dockerfile` (in the actual repo)

```dockerfile
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-linkmode=external -extldflags=-static" \
    -o /app/zdharness ./main.go

# Runtime: minimal Alpine with zdharness binary
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/zdharness /usr/local/bin/zdharness

# Mount points documented but not enforced in Dockerfile
# (caller is responsible for -v flags)
# /vault — Obsidian vault sync folder
# /state — state logs directory

ENTRYPOINT ["/usr/local/bin/zdharness"]
CMD ["--harness-dir=/vault/+/Things/AI/Harness", "--timeout=15m"]
```

### 1.2 Build and test locally

```bash
# Build image
docker build -t zdharness:latest -f Dockerfile .

# Test: run with explicit mount points
docker run -it \
  -v /mnt/v1drive/syncthing/data1:/vault \
  -v ~/.local/state/zdharness:/state \
  zdharness:latest

# Expected output: runs once, logs to /state/runs.log, exits with status code of claude invocation
```

### 1.3 Configuration file (optional)

Create `zdharness.docker.yml` or `.dockerignore` documenting mount points and environment:

```yaml
# zdharness.docker.yml — mount and run configuration
# Reference for Phase 2 systemd integration

image: zdharness:latest
container_name: zdharness-harness

mounts:
  vault:
    host_path: /mnt/v1drive/syncthing/data1
    container_path: /vault
    mode: ro  # read-only vault (harness reads config, not writes)
  
  state:
    host_path: ~/.local/state/zdharness
    container_path: /state
    mode: rw  # read-write state logs

environment:
  HARNESS_DIR: /vault/+/Things/AI/Harness
  STATE_LOG: /state/runs.log
  TIMEOUT: 15m

# Claude CLI: to be resolved in Phase 2
# Option A: bind-mount host's Claude installation
# Option B: install Claude inside container (requires authentication)
# Option C: use Claude socket if available
```

### 1.4 Testing checklist (Phase 1 completion)

- [ ] Dockerfile builds without errors
- [ ] Image runs locally with explicit mounts
- [ ] Container reads `.cron-state.json` from vault mount
- [ ] Container can invoke `claude --print` (mock or real)
- [ ] Logs appear in `/state/runs.log` on host
- [ ] Container exits with correct status code
- [ ] Image does not contain secrets or credentials

## Phase 2: Systemd integration

**Ticket:** TBD (separate ticket after Phase 1 approval)

### 2.1 Update systemd service

Current: `~/.config/systemd/user/zdharness.service`
```ini
[Unit]
Description=zdharness harness timer
After=network.target

[Service]
Type=oneshot
ExecStart=/home/zerodoc/Documents/lang/go/zdharness/zdharness
# ... environment / working directory
```

New (with podman):
```ini
[Unit]
Description=zdharness harness timer
After=network.target podman.socket

[Service]
Type=oneshot
ExecStart=/usr/bin/podman run --rm \
  -v /mnt/v1drive/syncthing/data1:/vault:ro \
  -v %h/.local/state/zdharness:/state:rw \
  zdharness:latest
# Note: %h expands to $HOME at runtime
```

### 2.2 End-to-end test

- [ ] Systemd timer fires; podman container starts
- [ ] Container reads vault config, invokes Claude
- [ ] Results logged to host's state directory
- [ ] Timer repeats on schedule without stale container cleanup

### 2.3 Cleanup and recovery

- [ ] Document how to clear stale containers: `podman ps -a | grep zdharness`
- [ ] Document how to verify image is current: `podman images`
- [ ] Document how to force rebuild: `podman build --no-cache`

## Phase 3: Polish (optional)

- Publish image to internal container registry (if registry configured)
- Add optional `--push` flag to CI/CD pipeline
- Optimize image size if needed (current estimate: ~50MB)
- Document air-gapped scenarios (build once, distribute image as tarball)

## Open questions for CTO

1. **Claude CLI access inside container:** how should the container invoke
   `claude --print`?
   - Option A: Bind-mount `~/.local/share/claude` (or equivalent) from host?
   - Option B: Install Claude inside image (requires auth token, may bloat image)?
   - Option C: Use Claude socket/API if available?

2. **Vault write access:** should zdharness be able to write to vault
   (e.g., to update `.cron-state.json` after a run)? Currently read-only in
   Phase 1 plan.

3. **Secrets in Phase 2:** if Claude CLI requires a token or session,
   how do we pass it into the container without baking it into the image?
   (Systemd environment variables? Volume-mounted `.env` file? Vault AppRole?)

4. **Image registry:** where should the built image live?
   - GitHub Container Registry (ghcr.io)?
   - Internal Nexus or Docker Hub?
   - Local only (no push)?

## Non-goals

- Kubernetes orchestration (out of scope)
- Multi-instance scheduling (timer already manages that)
- Hot reload (container is stateless, timer handles scheduling)
- Volume backups (state logs are ephemeral; only runs.log matters, stored on host)
