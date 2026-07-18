# Stage 1: Build the zdai Go binary.
# Private deps (zdlib) resolve via API_USERNAME/API_TOKEN build args — same
# pattern as zdapi. A local vendor/ dir, if present, is used automatically.
FROM golang:1.25-alpine AS build
ARG API_USERNAME=
ARG API_TOKEN=
WORKDIR /src
RUN apk add --no-cache git && \
    git config --global --add url.https://${API_USERNAME}:${API_TOKEN}@github.com.insteadOf https://github.com && \
    go env -w GOPRIVATE=github.com/zerodoctor/*,github.com/zerodoc-s-stack/* && \
    go env -w GONOSUMDB=github.com/zerodoctor/*,github.com/zerodoc-s-stack/* && \
    go env -w GONOPROXY=github.com/zerodoctor/*,github.com/zerodoc-s-stack/*
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY package/ package/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /zdai ./cmd/zdai/

# Stage 2: Install the claude CLI via npm
FROM node:20-alpine AS claude-install
RUN npm install -g @anthropic-ai/claude-code

# Stage 3: Runtime image
FROM alpine:3.22
# bash/curl/jq: required by ~/scripts (zdscripts) helpers, notably
# sh/vault-agent.sh for non-interactive Vault access via the zdagent approle.
RUN apk add --no-cache nodejs ca-certificates git bash curl jq

# zdai binary
COPY --from=build /zdai /usr/local/bin/zdai

# claude CLI (node_modules + wrapper script)
COPY --from=claude-install /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY --from=claude-install /usr/local/bin/claude /usr/local/bin/claude

# Entrypoint: merges zerodoctor/zdclaude (agents/skills) into /root/.claude without
# wiping existing auth credentials, then exec's zdai. GITHUB_TOKEN is required on
# first run if /root/.claude/agents/ is not already present.
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# /vault        — Obsidian vault  (/mnt/local/syncthing/data1 on vp0dune)
# /state        — run.lock, runs.log, zdai-state.json, tess-last-run
# /root/.claude — persistent Claude Max session + agents/skills from zdclaude
VOLUME ["/vault", "/state", "/root/.claude"]

ENV VAULT_DIR=/vault \
    STATE_DIR=/state \
    PORT=8080 \
    ZDCLAUDE_REPO=https://github.com/ZeroDoctor/zdclaude

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
