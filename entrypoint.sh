#!/bin/sh
# Bootstrap Claude Code config from zerodoctor/zdclaude into /root/.claude.
#
# Merges agents/, skills/, settings.json from the zdclaude repo without
# touching existing auth credentials (sessions/, .credentials, etc.) so that
# a persistent /root/.claude volume survives the clone.
#
# Requires GITHUB_TOKEN when /root/.claude/agents/ is not already present.
set -e

CLAUDE_DIR="${CLAUDE_DIR:-/root/.claude}"
ZDCLAUDE_REPO="${ZDCLAUDE_REPO:-https://github.com/ZeroDoctor/zdclaude}"
CLONE_TMP="/tmp/zdclaude-clone"

if [ ! -d "${CLAUDE_DIR}/agents" ]; then
    if [ -z "${GITHUB_TOKEN}" ]; then
        echo "zdai: ${CLAUDE_DIR}/agents not found and GITHUB_TOKEN not set — cannot clone zdclaude" >&2
        exit 1
    fi
    echo "zdai: cloning zdclaude..."
    AUTH_URL=$(echo "${ZDCLAUDE_REPO}" | sed "s|https://|https://x-access-token:${GITHUB_TOKEN}@|")
    rm -rf "${CLONE_TMP}"
    git clone --depth=1 "${AUTH_URL}" "${CLONE_TMP}"

    # Merge into CLAUDE_DIR — preserve any existing auth files
    mkdir -p "${CLAUDE_DIR}"
    cp -r "${CLONE_TMP}/agents"   "${CLAUDE_DIR}/agents"
    cp -r "${CLONE_TMP}/skills"   "${CLAUDE_DIR}/skills"
    [ -f "${CLONE_TMP}/settings.json" ] && \
        cp "${CLONE_TMP}/settings.json" "${CLAUDE_DIR}/settings.json"
    rm -rf "${CLONE_TMP}"
    echo "zdai: zdclaude merged into ${CLAUDE_DIR}"
fi

exec /usr/local/bin/zdai "$@"
