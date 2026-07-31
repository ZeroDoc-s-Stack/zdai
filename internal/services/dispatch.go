package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zerodoc-s-stack/zdlib/base/logger"
	"gopkg.in/yaml.v3"
)

// log is the shared logrus instance for the services package.
var log *logrus.Logger = logger.Log

type persona struct {
	agent string
	model string
}

// personaByAgentKind maps the agent-kind tag value to the Claude persona and
// model to use. Mirrors the dispatch table in Projects/ZDProject/zdharness.md.
var personaByAgentKind = map[string]persona{
	"coding":       {"developer", "claude-sonnet-4-6"},
	"api-consumer": {"developer", "claude-sonnet-4-6"},
	"research":     {"researcher", "google/gemini-3.5-flash"},
	"general":      {"researcher", "google/gemini-3.5-flash"},
	"audit":        {"auditor", "google/gemini-pro-latest"},
	"qa":           {"qa", "claude-haiku-4-5-20251001"},
	"sre":          {"sre", "google/gemini-3.5-flash"},
	"tess":         {"tess", "claude-sonnet-4-6"},
}

// taskFrontmatter holds the subset of fields shared by ticket-reading helpers.
type taskFrontmatter struct {
	Status string `yaml:"status"`
}

// parseFrontmatter extracts the YAML frontmatter from a note file.
func parseFrontmatter(data []byte) (taskFrontmatter, bool) {
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return taskFrontmatter{}, false
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return taskFrontmatter{}, false
	}
	var fm taskFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return taskFrontmatter{}, false
	}
	return fm, true
}

// readAgentKind reads the agent dispatch hint from a ticket's frontmatter tags.
// "agent:<name>" (direct persona) takes precedence over "agent-kind:<kind>".
func readAgentKind(vaultDir, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(vaultDir, path))
	if err != nil {
		return "", err
	}
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return "", nil
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", nil
	}
	frontmatter := rest[:end]

	var directAgent, kindAgent string
	for _, line := range strings.Split(frontmatter, "\n") {
		trimmed := strings.TrimSpace(line)
		tag, ok := strings.CutPrefix(trimmed, "- ")
		if !ok {
			continue
		}
		tag = strings.TrimSpace(tag)
		if v, ok2 := strings.CutPrefix(tag, "agent:"); ok2 && directAgent == "" {
			directAgent = strings.TrimSpace(v)
		}
		if v, ok2 := strings.CutPrefix(tag, "agent-kind:"); ok2 && kindAgent == "" {
			kindAgent = strings.TrimSpace(v)
		}
	}
	if directAgent != "" {
		return directAgent, nil
	}
	return kindAgent, nil
}

// resolvePersona returns the persona for a ticket by checking the ticket file's
// frontmatter for an "agent:<name>" tag (direct persona override) or
// "agent-kind:<kind>" tag (dispatch table lookup). Returns false if no
// resolvable persona is found.
func resolvePersona(vaultDir, path string) (persona, bool) {
	kind, err := readAgentKind(vaultDir, path)
	if err != nil || kind == "" {
		return persona{}, false
	}
	if p, ok := personaByAgentKind[kind]; ok {
		return p, true
	}
	// Direct persona name not in the table (e.g. a custom agent); construct
	// with a safe default model so the invocation still proceeds.
	return persona{agent: kind, model: "claude-sonnet-4-6"}, true
}

// overrideModel applies the ZDAI_MODEL_OVERRIDE env var, forcing every
// dispatch onto one model without touching the persona table. Used to bypass
// models the proxy can't serve (e.g. the google/gemini-* entries).
func overrideModel(p persona) persona {
	if m := os.Getenv("ZDAI_MODEL_OVERRIDE"); m != "" {
		p.model = m
	}
	return p
}

// opencodeArgs builds the `opencode run` argv for a non-claude model.
// opencode has no --agent wired to ~/.claude/agents, so the persona is
// injected via the prompt instead.
func opencodeArgs(p persona, prompt string) []string {
	return []string{
		"run",
		"--model", "openrouter/" + p.model,
		fmt.Sprintf("First read ~/.claude/agents/%s.md and adopt that agent persona exactly. Then: %s", p.agent, prompt),
	}
}

func invokeAgent(ctx context.Context, p persona, prompt, vaultDir, claudeBin, opencodeBin, effort, provider, logPath string) error {
	var cmd *exec.Cmd
	binName := "claude"
	if isClaudeModel(p.model) {
		args := []string{
			"--print",
			"--dangerously-skip-permissions",
			"--model", p.model,
			"--effort", effort,
			"--agent", p.agent,
			prompt,
		}
		cmd = exec.CommandContext(ctx, claudeBin, args...)

		env := append(os.Environ(), "ANTHROPIC_BASE_URL="+baseURLForModel(p.model))
		if provider == "openrouter" {
			key := os.Getenv("OPENROUTER_API_KEY")
			if key == "" {
				return fmt.Errorf("provider=openrouter but OPENROUTER_API_KEY is not set")
			}
			env = append(env, "ANTHROPIC_API_KEY="+key)
		}
		cmd.Env = env
	} else {
		// Non-claude models run through opencode, which talks to OpenRouter
		// directly (picks up OPENROUTER_API_KEY from the environment).
		binName = "opencode"
		if os.Getenv("OPENROUTER_API_KEY") == "" {
			return fmt.Errorf("model %s dispatches via opencode but OPENROUTER_API_KEY is not set", p.model)
		}
		cmd = exec.CommandContext(ctx, opencodeBin, opencodeArgs(p, prompt)...)
	}
	cmd.Dir = vaultDir

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = 124
	} else if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	appendLog(logPath, truncate(out.String(), maxOutputChars), exitCode, duration)
	if exitCode != 0 {
		return fmt.Errorf("%s exited %d (agent=%s model=%s)", binName, exitCode, p.agent, p.model)
	}
	return nil
}

// DispatchTicket reads the ticket's agent-kind tag, resolves the persona, and
// invokes claude --agent <persona> with the ticket path as the prompt.
func DispatchTicket(ctx context.Context, path string, vaultDir string, opts DispatchOpts) error {
	p, ok := resolvePersona(vaultDir, path)
	if !ok {
		return fmt.Errorf("no agent-kind or agent tag found in frontmatter")
	}
	p = overrideModel(p)
	log.Infof("zdai: dispatch ticket %s → agent=%s model=%s", path, p.agent, p.model)
	prompt := fmt.Sprintf("Execute the ticket at: %s", path)
	return invokeAgent(ctx, p, prompt, opts.VaultDir, opts.ClaudeBin, opts.OpencodeBin, opts.Effort, opts.Provider, opts.LogPath)
}

