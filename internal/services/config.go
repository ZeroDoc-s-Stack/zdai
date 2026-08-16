package services

import (
	"encoding/json"
	"os"
	"strings"
)

type HarnessConfig struct {
	Model    string `json:"model"`    // e.g. "anthropic/claude-haiku-4-5-20251001"
	Effort   string `json:"effort"`   // "low" | "medium" | "high"
	Provider string `json:"provider"` // "claude" (default) or "openrouter"
	Prompt   string `json:"prompt"`   // dispatch prompt sent to tess each cycle; empty disables harness dispatch
}

type TessConfig struct {
	Enabled  bool   `json:"enabled"`
	Schedule string `json:"schedule"` // "HH:MM" in local time, e.g. "07:00"
	Model    string `json:"model"`
	Provider string `json:"provider"`
	Prompt   string `json:"prompt"`
}

type EmailRoutingConfig struct {
	Enabled    bool   `json:"enabled"`
	GmailToken string `json:"gmail_token"` // OAuth Bearer token for zd.agents@gmail.com
}

type ZdaiState struct {
	Harness      HarnessConfig      `json:"harness"`
	Tess         TessConfig         `json:"tess"`
	EmailRouting EmailRoutingConfig `json:"email_routing"`
}

// normalizeModel prefixes bare claude-* names with "anthropic/" so opencode
// receives a provider-qualified model string. Already-prefixed strings (contain
// "/") and non-claude names pass through unchanged.
// ponytail: one guard here covers harness + tess; no per-call-site patches needed.
func normalizeModel(m string) string {
	if !strings.Contains(m, "/") && strings.HasPrefix(m, "claude-") {
		return "anthropic/" + m
	}
	return m
}

func LoadState(path string) (ZdaiState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ZdaiState{}, err
	}
	var s ZdaiState
	if err := json.Unmarshal(data, &s); err != nil {
		return ZdaiState{}, err
	}
	if s.Harness.Model == "" {
		s.Harness.Model = "claude-haiku-4-5-20251001"
	}
	if s.Harness.Effort == "" {
		s.Harness.Effort = "medium"
	}
	s.Harness.Model = normalizeModel(s.Harness.Model)
	s.Tess.Model = normalizeModel(s.Tess.Model)
	return s, nil
}
