package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadState_HarnessPrompt exercises the new Prompt field — empty string and
// a non-empty value. Defaults (model/effort) are also exercised.
func TestLoadState_HarnessPrompt(t *testing.T) {
	cases := []struct {
		name       string
		json       string
		wantPrompt string
		wantModel  string
		wantEffort string
	}{
		{
			name:       "prompt set",
			json:       `{"harness":{"prompt":"Run the harness dispatch cycle."}}`,
			wantPrompt: "Run the harness dispatch cycle.",
			wantModel:  "openrouter/anthropic/claude-haiku-4.5", // default
			wantEffort: "medium",                              // default
		},
		{
			name:       "prompt empty — dispatch disabled",
			json:       `{"harness":{}}`,
			wantPrompt: "",
			wantModel:  "openrouter/anthropic/claude-haiku-4.5", // default
			wantEffort: "medium",
		},
		{
			name:       "bare claude-* model is normalized",
			json:       `{"harness":{"model":"claude-sonnet-4-6","effort":"high","prompt":"dispatch"}}`,
			wantPrompt: "dispatch",
			wantModel:  "anthropic/claude-sonnet-4-6",
			wantEffort: "high",
		},
		{
			name:       "already-prefixed model passes through",
			json:       `{"harness":{"model":"anthropic/claude-sonnet-4-6","prompt":"dispatch"}}`,
			wantPrompt: "dispatch",
			wantModel:  "anthropic/claude-sonnet-4-6",
			wantEffort: "medium",
		},
		{
			name:       "openrouter model passes through unchanged",
			json:       `{"harness":{"model":"openrouter/google/gemini-3.5-flash","prompt":"dispatch"}}`,
			wantPrompt: "dispatch",
			wantModel:  "openrouter/google/gemini-3.5-flash",
			wantEffort: "medium",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "zdai-state.json")
			if err := os.WriteFile(p, []byte(tc.json), 0o644); err != nil {
				t.Fatal(err)
			}
			s, err := LoadState(p)
			if err != nil {
				t.Fatalf("LoadState: %v", err)
			}
			if s.Harness.Prompt != tc.wantPrompt {
				t.Errorf("Prompt = %q, want %q", s.Harness.Prompt, tc.wantPrompt)
			}
			if s.Harness.Model != tc.wantModel {
				t.Errorf("Model = %q, want %q", s.Harness.Model, tc.wantModel)
			}
			if s.Harness.Effort != tc.wantEffort {
				t.Errorf("Effort = %q, want %q", s.Harness.Effort, tc.wantEffort)
			}
		})
	}
}

// TestLoadState_RoundTrip verifies that a ZdaiState with a harness prompt
// survives marshal → unmarshal intact.
func TestLoadState_RoundTrip(t *testing.T) {
	original := ZdaiState{
		Harness: HarnessConfig{
			Model:    "anthropic/claude-sonnet-4-6", // already normalized; round-trip must be stable
			Effort:   "high",
			Provider: "openrouter",
			Prompt:   "Run the harness dispatch cycle via Tess.",
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "zdai-state.json")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadState(p)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.Harness.Prompt != original.Harness.Prompt {
		t.Errorf("Prompt = %q, want %q", got.Harness.Prompt, original.Harness.Prompt)
	}
	if got.Harness.Model != original.Harness.Model {
		t.Errorf("Model = %q, want %q", got.Harness.Model, original.Harness.Model)
	}
}
