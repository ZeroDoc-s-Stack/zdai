package services

import (
	"testing"
)

func TestOverrideModel(t *testing.T) {
	p := persona{agent: "researcher", model: "openrouter/google/gemini-3.5-flash"}

	if got := overrideModel(p); got.model != "openrouter/google/gemini-3.5-flash" {
		t.Errorf("unset override changed model to %q", got.model)
	}

	t.Setenv("ZDAI_MODEL_OVERRIDE", "anthropic/claude-sonnet-4-6")
	if got := overrideModel(p); got.model != "anthropic/claude-sonnet-4-6" {
		t.Errorf("override not applied, model = %q", got.model)
	}
	if got := overrideModel(p); got.agent != "researcher" {
		t.Errorf("override changed agent to %q", got.agent)
	}

	// Matches the live Nomad job's ZDAI_MODEL_OVERRIDE=claude-sonnet-4-6 (bare,
	// no provider prefix) — the exact value that broke Tess/harness dispatch in prod.
	t.Setenv("ZDAI_MODEL_OVERRIDE", "claude-sonnet-4-6")
	if got := overrideModel(p); got.model != "anthropic/claude-sonnet-4-6" {
		t.Errorf("bare override not normalized, model = %q", got.model)
	}
}
