package main

import (
	"encoding/json"
	"os"
)

type harnessConfig struct {
	Model    string `json:"model"`    // e.g. "claude-haiku-4-5-20251001"
	Effort   string `json:"effort"`   // "low" | "medium" | "high"
	Provider string `json:"provider"` // "claude" (default) or "openrouter"
}

type tessConfig struct {
	Enabled  bool   `json:"enabled"`
	Schedule string `json:"schedule"` // "HH:MM" in local time, e.g. "07:00"
	Model    string `json:"model"`
	Provider string `json:"provider"`
	Prompt   string `json:"prompt"`
}

type emailRoutingConfig struct {
	Enabled    bool   `json:"enabled"`
	GmailToken string `json:"gmail_token"` // OAuth Bearer token for zd.agents@gmail.com
}

type zdaiState struct {
	Harness      harnessConfig      `json:"harness"`
	Tess         tessConfig         `json:"tess"`
	EmailRouting emailRoutingConfig `json:"email_routing"`
}

func loadState(path string) (zdaiState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return zdaiState{}, err
	}
	var s zdaiState
	if err := json.Unmarshal(data, &s); err != nil {
		return zdaiState{}, err
	}
	if s.Harness.Model == "" {
		s.Harness.Model = "claude-haiku-4-5-20251001"
	}
	if s.Harness.Effort == "" {
		s.Harness.Effort = "medium"
	}
	return s, nil
}
