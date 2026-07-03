# Stage 1: Build the zdai Go binary
# Dependencies are vendored so the build works offline (zdlib is private).
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor vendor
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /zdai .

# Stage 2: Install the claude CLI via npm
FROM node:20-alpine AS claude-install
RUN npm install -g @anthropic-ai/claude-code

# Stage 3: Runtime image
FROM alpine:3.22
RUN apk add --no-cache nodejs ca-certificates git

# zdai binary
COPY --from=build /zdai /usr/local/bin/zdai

# claude CLI (node_modules + wrapper script)
COPY --from=claude-install /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY --from=claude-install /usr/local/bin/claude /usr/local/bin/claude

# Entrypoint: clones zerodoctor/zdclaude into /root/.claude on first run,
# then execs zdai. GITHUB_TOKEN must be provided at runtime via the Nomad job.
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# /vault   — Obsidian vault  (/mnt/local/syncthing/data1 on vp0dune)
# /scripts — ~/scripts clone  (/mnt/local/zdai/scripts or direct mount)
# /state   — run.lock, runs.log, zdai-state.json, tess-last-run
# /root/.claude — zdclaude clone (persistent volume; populated by entrypoint on first run)
VOLUME ["/vault", "/scripts", "/state", "/root/.claude"]

ENV VAULT_DIR=/vault \
    SCRIPTS_DIR=/scripts \
    STATE_DIR=/state \
    TASKNOTES_MCP_URL=http://host.containers.internal:8080/mcp \
    ZDCLAUDE_REPO=https://github.com/ZeroDoctor/zdclaude

ENTRYPOINT ["/usr/local/bin/entrypoint.sh", \
    "--vault-dir", "/vault", \
    "--state-dir", "/state"]
