# Dockerfile for Go projects
# Template for: single-service Go apps with optional private module deps
# See: adr-docker-container-standards.md — Recommendation section

# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache \
    ca-certificates \
    gcc \
    git \
    musl-dev

WORKDIR /app

# Copy only mod files first for layer caching
COPY go.mod go.sum ./

# For private modules: pass build args at build time
# docker build --build-arg API_USERNAME=... --build-arg API_TOKEN=... .
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

# Build: use CGO_ENABLED=1 for static linking, CGO_ENABLED=0 for pure Go
# For scratch images (no runtime), use: -linkmode=external -extldflags=-static
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-linkmode=external -extldflags=-static" \
    -o /app/binary ./cmd

# Runtime stage: use scratch for minimal images, or alpine for debugging access
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/binary /binary

# Optional: if the binary needs timezone data
# COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

EXPOSE 8080
ENTRYPOINT ["/binary"]

# To use alpine runtime instead of scratch:
# FROM alpine:latest
# RUN apk add --no-cache ca-certificates
# COPY --from=builder /app/binary /binary
# EXPOSE 8080
# ENTRYPOINT ["/binary"]
