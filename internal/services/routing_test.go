package services

import (
	"strings"
	"testing"
)

// TestBaseURLForModel verifies model-prefix routing: full claude-* IDs go to
// the direct headroom proxy; everything else (provider-prefixed, short aliases,
// non-claude prefixes) goes through headroom-or.
func TestBaseURLForModel(t *testing.T) {
	cases := []struct {
		name  string
		model string
		want  string
	}{
		{"claude opus full id", "claude-opus-4", headroomBaseURL},
		{"claude sonnet full id", "claude-3-5-sonnet", headroomBaseURL},
		{"claude haiku full id", "claude-haiku-4-5-20251001", headroomBaseURL},
		{"google gemini", "google/gemini-2.5-pro", headroomORBaseURL},
		{"openai gpt", "openai/gpt-4o", headroomORBaseURL},
		{"bare non-claude prefix", "llama-3.1", headroomORBaseURL},
		{"short alias haiku", "haiku", headroomORBaseURL},
		{"short alias sonnet", "sonnet", headroomORBaseURL},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := baseURLForModel(tc.model)
			if got != tc.want {
				t.Errorf("baseURLForModel(%q) = %q, want %q", tc.model, got, tc.want)
			}
		})
	}
}

// TestANTHROPIC_BASE_URLInEnv exercises the env-slice construction that
// invokeAgent uses for ANTHROPIC_BASE_URL, without executing any command.
func TestANTHROPIC_BASE_URLInEnv(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		provider string
		want     string
	}{
		{"google model", "google/gemini-2.5-pro", "", headroomORBaseURL},
		{"non-claude prefix", "openai/gpt-4o", "", headroomORBaseURL},
		{"claude full id", "claude-opus-4", "", headroomBaseURL},
		{"google + openrouter provider", "google/gemini-2.5-pro", "openrouter", headroomORBaseURL},
		{"claude + openrouter provider", "claude-sonnet-4-6", "openrouter", headroomBaseURL},
	}

	baseEnv := []string{"PATH=/usr/bin", "HOME=/home/test"}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := append(append([]string{}, baseEnv...), "ANTHROPIC_BASE_URL="+baseURLForModel(tc.model))
			if tc.provider == "openrouter" {
				env = append(env, "ANTHROPIC_API_KEY=test-key")
			}

			got, ok := lookupEnv(env, "ANTHROPIC_BASE_URL")
			if !ok {
				t.Fatalf("ANTHROPIC_BASE_URL not found in env: %v", env)
			}
			if got != tc.want {
				t.Errorf("ANTHROPIC_BASE_URL = %q, want %q (model=%q provider=%q)", got, tc.want, tc.model, tc.provider)
			}
		})
	}
}

func lookupEnv(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix), true
		}
	}
	return "", false
}
