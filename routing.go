package main

import "strings"

// headroomBaseURL and headroomORBaseURL are the two routing targets for
// ANTHROPIC_BASE_URL. Claude-native models go through the direct headroom
// proxy; everything else (provider-prefixed models like "google/...",
// short non-claude aliases like "haiku", etc.) goes through the
// OpenRouter-compatible headroom-or endpoint.
const headroomBaseURL = "https://headroom.internal.zerodoc.dev"
const headroomORBaseURL = "https://headroom-or.internal.zerodoc.dev"

// baseURLForModel picks the headroom endpoint based on model name.
// Only full claude-* IDs (e.g. "claude-sonnet-4-6", "claude-haiku-4-5-...") go
// to the direct Anthropic proxy. Short aliases ("haiku", "sonnet") and
// provider-prefixed names ("google/...", "openai/...") route through headroom-or.
func baseURLForModel(model string) string {
	if strings.HasPrefix(model, "claude-") {
		return headroomBaseURL
	}
	return headroomORBaseURL
}
