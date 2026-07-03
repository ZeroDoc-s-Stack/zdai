# ADR: docker-container-standards

**Status:** proposed
**Date:** 2026-06-29
**Ticket:** [[TaskNotes/Tasks/Define Docker container standards]]

## Context

The "Create coding standards" plan requires Docker/container standards as a
foundation for:

1. All new projects must be containerizable (mandatory requirement, not
   optional deployment convenience).
2. A phased containerization plan for `zdharness` — the OS-level scheduler
   binary that orchestrates the agent-harness system — to run inside a podman
   container instead of directly on the host.

This ADR draws on actual patterns found in the team's checked-out Go and
Svelte projects (`zdapi`, `zdintegration`, `zdworkflow/server/client`) and
applies the pragmatic-programmer principles from [[pragmatic-programmer-principles]]:
DRY for infrastructure conventions, orthogonal concerns in multi-service setups,
and "good enough" container design (no Kubernetes-grade orchestration for
development or single-service deployments).

## Current state: container patterns across projects

### Go projects

| Repo | Pattern | Private modules? | Secrets handling |
|------|---------|------------------|------------------|
| `zdapi` | Multi-stage builder (golang:1.24-alpine → alpine); explicit `GOPRIVATE`/`GONOSUMDB`/`GONOPROXY` via git insteadOf; ARG-based build args for `API_USERNAME`/`API_TOKEN` | Yes | Build-time git auth via ARG |
| `zdintegration` | Multi-stage builder (golang:1.25-alpine → scratch); static linking (`CGO_ENABLED=1 -linkmode=external`); no secret args needed | No (vendored or standard libs) | N/A (scratch image) |

**Pattern analysis:**

- **Builder stage:** golang:1.x-alpine + `apk add` (gcc, musl-dev, protoc, git,
  ca-certificates, tzdata as needed per repo)
- **Dependencies:** `COPY go.mod go.sum` → `go mod download` (layer caching);
  `COPY .` only after, so code changes don't invalidate mod cache
- **Build:** `CGO_ENABLED=1 GOOS=linux go build` + static linking for scratch
  images, or dynamic linking for alpine runtime
- **Runtime stage:** scratch (minimal) or alpine:latest (certs/shells available);
  copy only binary + certs + tzdata from builder
- **Secrets:** ARG-based build args (API_USERNAME/API_TOKEN) passed at build
  time, used only for git auth during build. No secrets baked into images.
- **Entry point:** `ENTRYPOINT ["/binary"]` with optional `CMD`; no shell
  wrappers

### Svelte/Node projects

| Repo | Pattern | SSR vs. SPA | Prod optimization |
|------|---------|------------|-------------------|
| `zdworkflow/client` | Multi-stage builder (node:20-alpine → node:20-alpine); `npm ci` in builder, `npm ci --production` in runtime | SSR (SvelteKit) | Copy only `build/` + prod deps; minimal runtime image |

**Pattern analysis:**

- **Builder stage:** node:X-alpine + `npm ci` (reproducible, uses package-lock.json)
- **Build command:** `npm run build` (project-specific, e.g. SvelteKit's Vite build)
- **Runtime stage:** node:X-alpine with prod deps only (`npm ci --production`)
- **Exposed port:** matches dev server port (5173 for Vite, customizable)
- **Entry point:** `CMD ["node", "build"]` for SSR-built artifact

### Multi-service orchestration

| Project | Services | Orchestration | Dev vs. prod |
|---------|----------|----------------|--------------|
| `zdworkflow` | postgres + backend (Go) + frontend (Svelte) | docker-compose.yml (3.8) | Same compose for dev; health checks on postgres |

**Compose pattern analysis:**

- **Version:** 3.8 (widely supported, no Compose v2 required)
- **Services:** each with `image:` or `build:` (build only for dev, image refs for prod)
- **Networking:** docker-compose provides service-name DNS resolution automatically
- **Health checks:** postgres uses `pg_isready` with 5s interval/timeout, 5 retries;
  backend `depends_on: postgres: condition: service_healthy`
- **Volumes:** named volumes for persistent data (postgres_data); bind mounts for
  init scripts (backend/schema.sql)
- **Environment:** inline `environment:` map; can be overridden via `.env` file
  or `-e` flags
- **Restart policy:** `unless-stopped` (restart on crash, unless explicitly stopped)

## Registry and build-push conventions

**Current practice (inferred from CI/CD mentions in repos):**

- Private module auth via git `insteadOf` URL rewriting + `GOPRIVATE` env vars
  (consistent across `zdapi`, `zdintegration`, etc.)
- No explicit Docker registry pattern documented; assumed to be internal or
  GitHub Container Registry
- Build secrets (git tokens) not baked into images; passed at build time via
  ARG and used only during `RUN` steps (ephemeral, not in final image)

**Recommendation:** use `--build-arg` at build time, never `ENV` for secrets;
container registries TBD (CTO decision on internal vs. GitHub Container Registry).

## zdharness containerization: current state and target

### Current state

- **Type:** single Go binary (no dynamic deps except runtime libs)
- **Invocation:** systemd timer fires `zdharness` once per cycle
- **State:** reads/writes to `~/.local/state/zdharness/runs.log` + reads
  `.cron-state.json` from vault sync folder
- **Dependencies:** Claude CLI (`claude --print`), Obsidian vault via Syncthing
- **Runtime environment:** Linux host with systemd, Syncthing sync folder
  mounted at `/mnt/v1drive/syncthing/data1`

### Architecture constraints for containerization

1. **Vault sync folder must be mounted:** zdharness reads `.cron-state.json` from
   the Obsidian vault; it must be available inside the container (bind mount or
   shared volume).
2. **Claude CLI must be available:** zdharness invokes `claude --print`; either
   the Claude binary lives in the container, or the container has access to the
   host's Claude installation (bind mount `~/.local/share/claude` or similar).
3. **State log must persist:** the `runs.log` file grows across invocations;
   either stored in a named volume or bind-mounted from host.
4. **Systemd timer replaces:** the container doesn't need to daemonize or
   schedule itself — the host's podman/systemd integration handles the timer
   (podman's systemd-native support or a wrapper script).
5. **No network isolation required initially:** zdharness is a local orchestrator,
   not a service. Network is out of scope for phase 1.

### Target state: phased approach

**Phase 1 (Foundation):**
- Build Dockerfile for zdharness that matches Go patterns (golang:1.25-alpine →
  alpine, static linking).
- Store base image + vault mount + state log mount config in a
  `zdharness.podman.yml` (compose file or equivalent podman config snippet).
- Document mount points: vault at `/vault`, state logs at `/state`.
- CTO verifies container runs locally with explicit mount flags (`podman run -v
  ...`) before automation.

**Phase 2 (Systemd integration):**
- Write systemd `.service` that wraps `podman run ...` with the necessary mount
  and network flags.
- Update existing `~/.config/systemd/user/zdharness.service` to use `podman run`
  instead of direct binary invocation.
- Test one full harness cycle inside container.

**Phase 3 (Polish):**
- Add optional container image publication to internal registry (TBD).
- Document recovery steps (e.g., clearing stale container or volume state).

## Recommendation

### Mandatory for all new projects

1. **All new projects must be dockerizable.** This means:
   - A `Dockerfile` present (or explicitly documented as "not applicable" with
     CTO approval, e.g., pure static/documentation projects).
   - Follows the Go or Svelte patterns below, whichever applies.
   - Built and tested locally before PR; CI gates on successful build.

2. **Repository structure:**
   - `Dockerfile` at project root (or `server/Dockerfile` and
     `client/Dockerfile` for multi-part projects like `zdworkflow`).
   - Optional: `docker-compose.yml` if the project has multiple services (DB +
     app, frontend + backend, etc.) needed for local dev.

3. **Go project template:**
   ```dockerfile
   # Build stage
   FROM golang:1.25-alpine AS builder

   RUN apk add --no-cache \
       ca-certificates \
       gcc \
       musl-dev \
       git

   WORKDIR /app

   # Copy only mod files first for layer caching
   COPY go.mod go.sum ./

   # For private modules, git auth happens here:
   # Pass --build-arg API_USERNAME=... --build-arg API_TOKEN=... at build time
   ARG API_USERNAME=
   ARG API_TOKEN=

   RUN if [ ! -z "$API_USERNAME" ]; then \
         git config --global --add url.https://${API_USERNAME}:${API_TOKEN}@github.com.insteadOf https://github.com && \
         go env -w GOPRIVATE=github.com/zerodoctor/*,github.com/zerodoc-s-stack/* && \
         go env -w GONOSUMDB=github.com/zerodoctor/*,github.com/zerodoc-s-stack/* && \
         go env -w GONOPROXY=github.com/zerodoctor/*,github.com/zerodoc-s-stack/*; \
       fi

   RUN go mod download

   # Copy source
   COPY . .

   # Build
   RUN CGO_ENABLED=1 GOOS=linux go build \
       -ldflags="-linkmode=external -extldflags=-static" \
       -o /app/binary ./cmd

   # Runtime stage: use scratch for minimal images
   FROM scratch

   COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
   COPY --from=builder /app/binary /binary

   EXPOSE 8080
   ENTRYPOINT ["/binary"]
   ```

   **Rationale (pragmatic programmer principles):**
   - DRY: single Dockerfile pattern reused across all Go repos; no per-repo
     variation unless justified.
   - Orthogonality: build stage is isolated from runtime; changes to Go source
     don't invalidate cert copying.
   - Good enough: scratch image is minimal but handles the most common case
     (binary-only Go apps). Alpine runtime is acceptable if you need shell
     access for debugging or dynamic libs.

4. **Svelte/Node project template:**
   ```dockerfile
   # Build stage
   FROM node:20-alpine AS builder

   WORKDIR /app

   COPY package*.json ./
   RUN npm ci

   COPY . .
   RUN npm run build

   # Runtime stage
   FROM node:20-alpine

   WORKDIR /app

   COPY --from=builder /app/build ./build
   COPY --from=builder /app/package*.json ./

   RUN npm ci --production

   EXPOSE 5173
   CMD ["node", "build"]
   ```

   **Rationale:**
   - Matches `zdworkflow/client` pattern exactly (two-stage, `npm ci` for
     reproducibility).
   - SvelteKit SSR builds to a Node.js adapter; the runtime image only needs
     Node + production deps.

5. **Multi-service compose template** (use only if project has 2+ services):
   ```yaml
   version: '3.8'

   services:
     # Example: postgres DB
     db:
       image: postgres:15-alpine
       container_name: <project>-db
       environment:
         POSTGRES_DB: ${DB_NAME:-mydb}
         POSTGRES_USER: ${DB_USER:-postgres}
         POSTGRES_PASSWORD: ${DB_PASS:-postgres}
       ports:
         - "5432:5432"
       volumes:
         - <project>_data:/var/lib/postgresql/data
         - ./schema.sql:/docker-entrypoint-initdb.d/schema.sql
       healthcheck:
         test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-postgres}"]
         interval: 5s
         timeout: 5s
         retries: 5

     # Example: backend service
     backend:
       build:
         context: ./backend
         dockerfile: Dockerfile
       container_name: <project>-backend
       environment:
         DATABASE_URL: postgres://${DB_USER:-postgres}:${DB_PASS:-postgres}@db:5432/${DB_NAME:-mydb}?sslmode=disable
         PORT: 8080
       ports:
         - "8080:8080"
       depends_on:
         db:
           condition: service_healthy
       restart: unless-stopped

   volumes:
     <project>_data:
   ```

   **Rationale:**
   - Service names resolve via Docker DNS (e.g., `db` from `backend` container).
   - Health checks gate startup dependencies (`depends_on: condition:
     service_healthy`).
   - Environment variables can be overridden at runtime via `.env` or CLI.

6. **Build and registry conventions:**
   - **Build locally:** `docker build -t <project>:<version> .` or `docker build
     --build-arg API_USERNAME=... --build-arg API_TOKEN=... .` for private
     modules.
   - **Test locally:** `docker run -it <project>:<version>` or `docker-compose
     up` for multi-service projects.
   - **Registry:** TBD by CTO (internal Nexus, GitHub Container Registry, or
     other); once decided, add a `.dockerignore` file to exclude `node_modules`,
     `.git`, test files, etc., and update CI to publish built images.
   - **Secrets in build args:** never stored as `ENV` in final image; passed
     only at build time and used in ephemeral `RUN` steps.

### zdharness containerization plan

**Phase 1 (This ticket):**

1. Create `Dockerfile` for zdharness:
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

   FROM alpine:latest

   RUN apk add --no-cache ca-certificates

   COPY --from=builder /app/zdharness /usr/local/bin/zdharness

   # Mount points for state and vault
   VOLUME ["/vault", "/state"]

   # Entry point: zdharness with flags for mounted paths
   ENTRYPOINT ["/usr/local/bin/zdharness", \
               "--harness-dir=/vault/+/Things/AI/Harness", \
               "--timeout=15m"]
   ```

2. Document mount/run instructions:
   - Vault: `podman run -v /mnt/v1drive/syncthing/data1:/vault ...`
   - State logs: `podman run -v ~/.local/state/zdharness:/state ...`
   - Claude CLI availability: document the bind mount path (e.g., `~/.local/share/claude`)
     or build Claude into the image (out of scope for phase 1).

3. Create `zdharness.podman.yml` (compose-like config or docs) with the full
   invocation command and mount flags.

**Phase 2+ (separate ticket):**
- Systemd integration: modify `~/.config/systemd/user/zdharness.service` to use
  `podman run` instead of direct binary invocation.
- End-to-end test: run a full harness cycle in the container.

## Implementation notes

- This ADR does not mandate any specific container registry or CI/CD integration
  (that's a separate CTO decision and may vary by project).
- The Dockerfiles above are templates, not rigid rules. Projects may adjust
  (e.g., different Node version, different Go version, additional build deps),
  but must justify the deviation in their README or CLAUDE.md.
- For multi-service projects, docker-compose is the standard. For single-service
  projects, a Dockerfile alone is sufficient.
- Container image size is not explicitly gated (e.g., "keep images under 100MB")
  because the team's current practice has not enforced this; if size becomes a
  concern (e.g., slow registry pushes), that's a separate optimization ticket.

## Consequences

- All new projects now have a containerization requirement, which enables:
  - Portable local dev across machines (compose up, no platform-specific
    install steps).
  - CI/CD standardization (same build pattern across all languages).
  - zdharness containerization as a phased roadmap.
- The templates above (Go, Svelte, compose) are stable enough to be added to
  new-project scaffolding or copied directly into existing repos.
- Documentation templates (from [[adr-go-standards]] rec #4) should include a
  "Containerization" section pointing to these templates; Svelte and testing
  standards ADRs can cross-reference this ADR instead of re-documenting.
- If a project cannot be dockerized for technical reasons (e.g., hardware-dependent
  binary, requires host networking), document that explicitly in the repo's README
  with CTO sign-off, rather than silently exempting it.

## Audit notes

- No external API or container registry access was needed for this ADR — it
  draws on checked-out repos, existing Dockerfiles, and the team's running
  `zdharness` binary.
- `zdauth` is not containerized locally (consumed as remote module); if its
  source later becomes available, it should be audited for any
  database-container or service-discovery patterns not evident in `zdapi` (its
  consumer).
