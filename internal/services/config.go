package services

import (
	"encoding/json"
	"os"
)

type HarnessConfig struct {
	Model    string `json:"model"`    // e.g. "claude-haiku-4-5-20251001"
	Effort   string `json:"effort"`   // "low" | "medium" | "high"
	Provider string `json:"provider"` // "claude" (default) or "openrouter"
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
	return s, nil
}
