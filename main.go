// zdai is the Claude Code agent-harness web service. It runs a background
// dispatch scheduler and exposes an HTTP API for triggering agent runs and
// querying run history. No MCP server required — vault tasks are read directly.
package main

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/zerodoc-s-stack/zdlib/base/logger"
)

var log = logger.Log

// defaultVaultDir is the Obsidian vault root on vp0dune (Syncthing target).
// Override via --vault-dir or VAULT_DIR env var for other hosts.
const defaultVaultDir = "/mnt/local/syncthing/data1"

func main() {
	vaultDir := flag.String("vault-dir", envOr("VAULT_DIR", defaultVaultDir), "Obsidian vault root")
	stateDir := flag.String("state-dir", envOr("STATE_DIR", ""), "state directory (run.lock, runs.log, zdai-state.json)")
	claudeBin := flag.String("claude-bin", "claude", "claude CLI binary")
	timeout := flag.Duration("timeout", 15*time.Minute, "max duration per claude invocation")
	port := flag.String("port", envOr("PORT", "8080"), "HTTP listen port")
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
			log.Fatal("zdai: another instance is already running")
		}
		log.Fatalf("zdai: flock: %v", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) //nolint:errcheck

	cfg, err := loadState(statePath)
	if err != nil {
		log.Fatalf("zdai: load zdai-state.json: %v", err)
	}

	rotateLogIfLarge(logPath)

	// Publish runtime opts so the scheduler and HTTP handlers share them.
	_opts = dispatchOpts{
		vaultDir:  *vaultDir,
		claudeBin: *claudeBin,
		timeout:   *timeout,
		logPath:   logPath,
		model:     cfg.Harness.Model,
		effort:    cfg.Harness.Effort,
		provider:  cfg.Harness.Provider,
	}

	startScheduler()

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: newRouter(),
	}
	log.Infof("zdai: listening on :%s", *port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("zdai: server: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
