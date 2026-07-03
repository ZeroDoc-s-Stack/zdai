# Stage 1: Build the zdai Go binary
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /zdai .

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

# Bake agent definitions and skills so the image is fully self-contained.
# Rebuild the image to pick up changes to agents or skills.
COPY agents/ /root/.claude/agents/
COPY skills/ /root/.claude/skills/

# /vault  — Obsidian vault (always volume-mounted; never baked)
# /scripts — ~/scripts (volume-mounted read-only)
# /state  — run.lock, runs.log, zdai-state.json, tess-last-run (volume-mounted)
VOLUME ["/vault", "/scripts", "/state"]

ENV VAULT_DIR=/vault \
    SCRIPTS_DIR=/scripts \
    STATE_DIR=/state \
    TASKNOTES_MCP_URL=http://host.containers.internal:8080/mcp

ENTRYPOINT ["/usr/local/bin/zdai", \
    "--vault-dir", "/vault", \
    "--state-dir", "/state"]
