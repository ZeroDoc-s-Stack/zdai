#!/bin/sh
# Bootstrap ~/.claude from the zerodoctor/zdclaude repo if not already present.
# GITHUB_TOKEN must be set for the clone to succeed.
# On subsequent runs the directory is reused (no re-clone); pull to update.
set -e

CLAUDE_DIR="${CLAUDE_DIR:-/root/.claude}"
ZDCLAUDE_REPO="${ZDCLAUDE_REPO:-https://github.com/ZeroDoctor/zdclaude}"

if [ ! -d "${CLAUDE_DIR}/agents" ]; then
    if [ -z "${GITHUB_TOKEN}" ]; then
        echo "zdai: CLAUDE_DIR=${CLAUDE_DIR}/agents not found and GITHUB_TOKEN is not set — cannot clone zdclaude" >&2
        exit 1
    fi
    echo "zdai: cloning zdclaude into ${CLAUDE_DIR}..."
    AUTH_URL=$(echo "${ZDCLAUDE_REPO}" | sed "s|https://|https://x-access-token:${GITHUB_TOKEN}@|")
    git clone --depth=1 "${AUTH_URL}" "${CLAUDE_DIR}"
    echo "zdai: zdclaude clone complete"
fi

exec /usr/local/bin/zdai "$@"
