package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// shouldRunTess returns true if the Tess daily note should fire this cycle.
// It reads tessLastRunPath for the date of the last run (YYYY-MM-DD) and
// compares against today. The schedule string is "HH:MM" in local time; if
// the current time is before schedule, Tess is deferred to the next cycle
// that fires after that time.
func shouldRunTess(schedule, tessLastRunPath string) bool {
	today := time.Now().Format("2006-01-02")

	data, err := os.ReadFile(tessLastRunPath)
	if err == nil && strings.TrimSpace(string(data)) == today {
		return false // already ran today
	}

	if schedule == "" {
		return true
	}
	var hour, minute int
	if _, err := fmt.Sscanf(schedule, "%d:%d", &hour, &minute); err != nil {
		return true // malformed schedule; fire anyway
	}
	now := time.Now()
	scheduled := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	return !now.Before(scheduled)
}

// markTessRan writes today's date to tessLastRunPath so subsequent cycles this
// day skip the Tess trigger.
func markTessRan(tessLastRunPath string) error {
	today := time.Now().Format("2006-01-02")
	return os.WriteFile(tessLastRunPath, []byte(today), 0o644)
}

// runTess invokes `claude --agent tess` with the configured Tess prompt.
func runTess(ctx context.Context, cfg tessConfig, claudeBin, vaultDir, logPath string) error {
	if cfg.Prompt == "" {
		return fmt.Errorf("tess.prompt is empty in zdai-state.json")
	}
	p := persona{agent: "tess", model: cfg.Model}
	if p.model == "" {
		p.model = "claude-sonnet-4-6"
	}
	log.Infof("zdai: tess daily trigger → agent=tess model=%s", p.model)
	return invokeAgent(ctx, p, cfg.Prompt, vaultDir, claudeBin, "medium", cfg.Provider, logPath)
}
