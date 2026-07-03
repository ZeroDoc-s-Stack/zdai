package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

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

// requestPersona is the persona used for all agent-request tasks regardless
// of their tags — requests are always handled by the planner.
var requestPersona = persona{"planner", "claude-haiku-4-5-20251001"}

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

func invokeAgent(ctx context.Context, p persona, prompt, vaultDir, claudeBin, effort, provider, logPath string) error {
	args := []string{
		"--print",
		"--dangerously-skip-permissions",
		"--model", p.model,
		"--effort", effort,
		"--agent", p.agent,
		prompt,
	}
	cmd := exec.CommandContext(ctx, claudeBin, args...)
	cmd.Dir = vaultDir

	env := append(os.Environ(), "ANTHROPIC_BASE_URL="+baseURLForModel(p.model))
	if provider == "openrouter" {
		key := os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			return fmt.Errorf("provider=openrouter but OPENROUTER_API_KEY is not set")
		}
		env = append(env, "ANTHROPIC_API_KEY="+key)
	}
	cmd.Env = env

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
		return fmt.Errorf("claude exited %d (agent=%s model=%s)", exitCode, p.agent, p.model)
	}
	return nil
}

// dispatchTicket reads the ticket's agent-kind tag, resolves the persona, and
// invokes claude --agent <persona> with the ticket path as the prompt.
func dispatchTicket(ctx context.Context, path string, vaultDir string, opts dispatchOpts) error {
	p, ok := resolvePersona(vaultDir, path)
	if !ok {
		return fmt.Errorf("no agent-kind or agent tag found in frontmatter")
	}
	log.Infof("zdai: dispatch ticket %s → agent=%s model=%s", path, p.agent, p.model)
	prompt := fmt.Sprintf("Execute the ticket at: %s", path)
	return invokeAgent(ctx, p, prompt, opts.vaultDir, opts.claudeBin, opts.effort, opts.provider, opts.logPath)
}

// dispatchRequest dispatches an agent-request task to the planner persona.
func dispatchRequest(ctx context.Context, path string, opts dispatchOpts) {
	p := requestPersona
	log.Infof("zdai: dispatch request %s → agent=%s model=%s", path, p.agent, p.model)
	prompt := fmt.Sprintf("Process the agent-request at: %s", path)
	if err := invokeAgent(ctx, p, prompt, opts.vaultDir, opts.claudeBin, opts.effort, opts.provider, opts.logPath); err != nil {
		log.Errorf("zdai: dispatch request %s: %v", path, err)
	}
}
