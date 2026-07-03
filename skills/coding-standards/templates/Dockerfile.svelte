# Dockerfile for SvelteKit projects (SSR)
# Template for: SvelteKit SSR-built apps with Node.js adapter
# See: adr-docker-container-standards.md — Recommendation section

# Build stage: compile SvelteKit app to Node.js adapter output
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files first for layer caching
COPY package*.json ./

# Use npm ci for reproducible installs (requires package-lock.json in repo)
RUN npm ci

# Copy source code
COPY . .

# Build the SvelteKit app (outputs to build/ directory)
RUN npm run build

# Runtime stage: minimal image with prod deps only
FROM node:20-alpine

WORKDIR /app

# Copy built artifacts from builder
COPY --from=builder /app/build ./build
COPY --from=builder /app/package*.json ./

# Install production dependencies only
RUN npm ci --production

EXPOSE 5173

# SvelteKit SSR adapter produces a Node.js server
CMD ["node", "build"]

# Notes:
# - Assumes package.json has a "build" script (default for SvelteKit)
# - Node adapter must be installed: npm install -D @sveltejs/adapter-node
# - If using a different adapter (e.g., static), adjust EXPOSE and CMD accordingly
# - For development, use docker-compose with volume mounts instead of building this image
