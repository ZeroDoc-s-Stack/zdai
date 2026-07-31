package services

import (
	"context"
	"path/filepath"
	"time"

	"github.com/zerodoc-s-stack/zdai/internal/models"
)

// DispatchOpts carries the runtime parameters for a dispatch cycle.
// SetOpts() stores the value; GlobalOpts() retrieves it.
type DispatchOpts struct {
	VaultDir    string
	ClaudeBin   string
	OpencodeBin string
	Timeout     time.Duration
	LogPath   string
	Model     string
	Effort    string
	Provider  string
}

var _opts DispatchOpts

// SetOpts stores the dispatch options. Called once from main after flag parsing.
func SetOpts(opts DispatchOpts) { _opts = opts }

// GetOpts returns the current dispatch options.
func GetOpts() DispatchOpts { return _opts }

func globalOpts() DispatchOpts { return _opts }

// RunCycle executes one full dispatch cycle: Tess daily check + harness dispatch via Tess.
// It is called both by the scheduler and by POST /v1/dispatch.
func RunCycle(trigger string) {
	r := Store.Begin(trigger)
	status := models.RunStatusDone

	opts := globalOpts()
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cfg, err := LoadState(filepath.Join(opts.VaultDir, "..", "state", "zdai-state.json"))
	if err != nil {
		log.Errorf("zdai: load state: %v", err)
		Store.Finish(r, models.RunStatusFailed)
		return
	}

	// Email-driven ticket unblock check.
	if _emailRouter != nil {
		_emailRouter.checkBlockedTickets(ctx, opts.VaultDir, opts)
	}

	// Tess daily check.
	if cfg.Tess.Enabled {
		tessLastRun := filepath.Join(filepath.Dir(opts.LogPath), "tess-last-run")
		if shouldRunTess(cfg.Tess.Schedule, tessLastRun) {
			if err := runTess(ctx, cfg.Tess, opts.ClaudeBin, opts.OpencodeBin, opts.VaultDir, opts.LogPath); err != nil {
				log.Errorf("zdai: tess: %v", err)
			} else {
				_ = markTessRan(tessLastRun)
			}
		}
	}

	// Harness dispatch: invoke Tess with the configured prompt so it queries Plane and dispatches agents.
	// ponytail: no eligibleWork() vault scan — TaskNotes/AI/ was retired 2026-07-23; Tess owns Plane querying.
	if cfg.Harness.Prompt != "" {
		p := persona{agent: "tess", model: cfg.Harness.Model}
		p = overrideModel(p)
		log.Infof("zdai: harness dispatch → agent=tess model=%s", p.model)
		if err := invokeAgent(ctx, p, cfg.Harness.Prompt, opts.VaultDir, opts.ClaudeBin, opts.OpencodeBin, cfg.Harness.Effort, cfg.Harness.Provider, opts.LogPath); err != nil {
			log.Errorf("zdai: harness dispatch: %v", err)
			status = models.RunStatusFailed
		}
	} else {
		appendLog(opts.LogPath, "harness.prompt not configured; skipping dispatch", 0, 0)
	}

	Store.Finish(r, status)
}

// dispatchMinutes are the minute-offsets within each hour when a cycle runs.
var dispatchMinutes = map[int]bool{7: true, 22: true, 37: true, 52: true}

// StartScheduler runs a background goroutine that fires a dispatch cycle at
// :07, :22, :37, and :52 of every hour between 08:00 and 22:00 local time.
func StartScheduler() {
	go func() {
		for {
			now := time.Now()
			next := now.Truncate(time.Minute).Add(time.Minute)
			time.Sleep(time.Until(next))

			t := time.Now()
			h, m := t.Hour(), t.Minute()
			if h >= 8 && h < 22 && dispatchMinutes[m] {
				go RunCycle("scheduled")
			}
		}
	}()
}
