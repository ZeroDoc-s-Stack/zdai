package services

import "strings"

// headroomBaseURL and headroomORBaseURL are the two routing targets for
// ANTHROPIC_BASE_URL. Claude-native models go through the direct headroom
// proxy; everything else (provider-prefixed models like "google/...",
// short non-claude aliases like "haiku", etc.) goes through the
// OpenRouter-compatible headroom-or endpoint.
const headroomBaseURL = "https://headroom.internal.zerodoc.dev"
const headroomORBaseURL = "https://headroom-or.internal.zerodoc.dev"

// isClaudeModel reports whether a model runs via the claude CLI. Only full
// claude-* IDs (e.g. "claude-sonnet-4-6", "claude-haiku-4-5-...") qualify;
// everything else (provider-prefixed names like "google/...", short aliases)
// is dispatched through the opencode CLI against OpenRouter.
func isClaudeModel(model string) bool {
	return strings.HasPrefix(model, "claude-")
}

// baseURLForModel picks the headroom endpoint for a claude CLI invocation.
func baseURLForModel(model string) string {
	if isClaudeModel(model) {
		return headroomBaseURL
	}
	return headroomORBaseURL
}
