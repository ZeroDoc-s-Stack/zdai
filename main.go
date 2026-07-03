// zdai is the Claude Code agent-harness orchestrator. It pre-filters eligible
// agent-request and agent-queue tasks via the TaskNotes MCP server, dispatches
// each directly to its persona (developer, researcher, auditor, qa, sre, tess)
// via `claude --agent`, and fires a daily Tess note independently of the ticket
// queue. One invocation = one full dispatch cycle; repetition is the systemd
// timer's job.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/zerodoc-s-stack/zdlib/base/logger"
)

var log = logger.Log

const defaultVaultDir = "/mnt/v1drive/syncthing/data1"
const defaultStateDir = "" // resolved to ~/.local/state/zdai at runtime

func main() {
	vaultDir := flag.String("vault-dir", defaultVaultDir, "working directory for claude invocations (vault root)")
	stateDir := flag.String("state-dir", defaultStateDir, "state directory for run.lock, runs.log, zdai-state.json, tess-last-run")
	claudeBin := flag.String("claude-bin", "claude", "claude CLI binary to invoke")
	timeout := flag.Duration("timeout", 15*time.Minute, "max duration per claude invocation")
	mcpURL := flag.String("mcp-url", defaultMCPURL, "TaskNotes MCP server URL for pre-filtering eligible tickets")
	flag.Parse()

	logger.LoadText(true)
	logger.LogStart("zdai")
	defer logger.LogStop("zdai")

	if *stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("zdai: resolve home dir: %v", err)
		}
		*stateDir = filepath.Join(home, ".local", "state", "zdai")
	}
	if err := os.MkdirAll(*stateDir, 0o755); err != nil {
		log.Fatalf("zdai: create state dir: %v", err)
	}

	lockPath := filepath.Join(*stateDir, "run.lock")
	logPath := filepath.Join(*stateDir, "runs.log")
	statePath := filepath.Join(*stateDir, "zdai-state.json")

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		log.Fatalf("zdai: open lock file: %v", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			appendLog(logPath, "previous run still active, skipping", -1, 0)
			os.Exit(0)
		}
		log.Fatalf("zdai: flock: %v", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	state, err := loadState(statePath)
	if err != nil {
		log.Fatalf("zdai: load zdai-state.json: %v", err)
	}

	rotateLogIfLarge(logPath)

	opts := dispatchOpts{
		vaultDir:  *vaultDir,
		claudeBin: *claudeBin,
		timeout:   *timeout,
		logPath:   logPath,
		model:     state.Harness.Model,
		effort:    state.Harness.Effort,
		provider:  state.Harness.Provider,
	}

	// Tess daily note fires before ticket dispatch so it always runs even when
	// ticket dispatch is lengthy.
	if state.Tess.Enabled {
		tessLastRunPath := filepath.Join(*stateDir, "tess-last-run")
		if shouldRunTess(state.Tess.Schedule, tessLastRunPath) {
			ctx, cancel := context.WithTimeout(context.Background(), *timeout)
			if err := runTess(ctx, state.Tess, *claudeBin, *vaultDir, logPath); err != nil {
				log.Errorf("zdai: tess daily run: %v", err)
			} else {
				if err := markTessRan(tessLastRunPath); err != nil {
					log.Errorf("zdai: mark tess ran: %v", err)
				}
			}
			cancel()
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	requests, tickets, err := eligibleWork(ctx, mcpHTTPClient(), *mcpURL, *vaultDir)
	if err != nil {
		log.Fatalf("zdai: pre-filter eligible work: %v", err)
	}
	if len(requests) == 0 && len(tickets) == 0 {
		appendLog(logPath, "no eligible agent-request tasks or agent-queue tickets this cycle; skipped claude invocation", 0, 0)
		return
	}

	// Dispatch requests (always planner) then tickets (per agent-kind tag).
	for _, path := range requests {
		dispatchRequest(ctx, path, opts)
	}
	for _, path := range tickets {
		if err := dispatchTicket(ctx, path, *vaultDir, opts); err != nil {
			log.Errorf("zdai: dispatch ticket %s: %v", path, err)
			appendLog(logPath, fmt.Sprintf("skipped %s: %v", path, err), 1, 0)
		}
	}
}
