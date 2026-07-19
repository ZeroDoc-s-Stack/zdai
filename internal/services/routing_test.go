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

// TestOpencodeDispatchSelection verifies which models go to opencode vs the
// claude CLI, and that opencode argv carries the openrouter-prefixed model
// and persona-injected prompt.
func TestOpencodeDispatchSelection(t *testing.T) {
	if isClaudeModel("google/gemini-3.5-flash") || isClaudeModel("haiku") {
		t.Error("non-claude model classified as claude")
	}
	if !isClaudeModel("claude-sonnet-4-6") {
		t.Error("claude-sonnet-4-6 not classified as claude")
	}

	args := opencodeArgs(persona{agent: "researcher", model: "google/gemini-3.5-flash"}, "Execute the ticket at: X")
	if args[0] != "run" {
		t.Errorf("args[0] = %q, want \"run\"", args[0])
	}
	got, ok := lookupFlag(args, "--model")
	if !ok || got != "openrouter/google/gemini-3.5-flash" {
		t.Errorf("--model = %q, want openrouter/google/gemini-3.5-flash", got)
	}
	prompt := args[len(args)-1]
	if !strings.Contains(prompt, "researcher.md") || !strings.Contains(prompt, "Execute the ticket at: X") {
		t.Errorf("prompt missing persona or original task: %q", prompt)
	}
}

func lookupFlag(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func TestOverrideModel(t *testing.T) {
	p := persona{agent: "researcher", model: "google/gemini-3.5-flash"}

	if got := overrideModel(p); got.model != "google/gemini-3.5-flash" {
		t.Errorf("unset override changed model to %q", got.model)
	}

	t.Setenv("ZDAI_MODEL_OVERRIDE", "claude-sonnet-4-6")
	if got := overrideModel(p); got.model != "claude-sonnet-4-6" {
		t.Errorf("override not applied, model = %q", got.model)
	}
	if got := overrideModel(p); got.agent != "researcher" {
		t.Errorf("override changed agent to %q", got.agent)
	}
}
