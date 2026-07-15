package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/zerodoc-s-stack/zdai/internal/models"
)

// DispatchOpts carries the runtime parameters for a dispatch cycle.
// SetOpts() stores the value; GlobalOpts() retrieves it.
type DispatchOpts struct {
	VaultDir  string
	ClaudeBin string
	Timeout   time.Duration
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

// RunCycle executes one full dispatch cycle: Tess check + eligible work.
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
			if err := runTess(ctx, cfg.Tess, opts.ClaudeBin, opts.VaultDir, opts.LogPath); err != nil {
				log.Errorf("zdai: tess: %v", err)
			} else {
				_ = markTessRan(tessLastRun)
			}
		}
	}

	requests, tickets, err := eligibleWork(opts.VaultDir)
	if err != nil {
		log.Errorf("zdai: eligible work: %v", err)
		Store.Finish(r, models.RunStatusFailed)
		return
	}

	if len(requests) == 0 && len(tickets) == 0 {
		appendLog(opts.LogPath, "no eligible work this cycle", 0, 0)
		Store.Finish(r, models.RunStatusDone)
		return
	}

	for _, path := range requests {
		dispatchRequest(ctx, path, opts)
	}
	for _, path := range tickets {
		if err := DispatchTicket(ctx, path, opts.VaultDir, opts); err != nil {
			log.Errorf("zdai: dispatch %s: %v", path, err)
			appendLog(opts.LogPath, fmt.Sprintf("skipped %s: %v", path, err), 1, 0)
			status = models.RunStatusFailed
		}
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
