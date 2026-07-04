package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

// dispatchOpts carries the runtime parameters for a dispatch cycle.
// globalOpts() builds it from the parsed flags so both the scheduler
// and the HTTP handlers can share it.
type dispatchOpts struct {
	vaultDir  string
	claudeBin string
	timeout   time.Duration
	logPath   string
	model     string
	effort    string
	provider  string
}

var _opts dispatchOpts

func globalOpts() dispatchOpts { return _opts }

// runCycle executes one full dispatch cycle: Tess check + eligible work.
// It is called both by the scheduler and by POST /v1/dispatch.
func runCycle(trigger string) {
	r := store.begin(trigger)
	status := RunStatusDone

	opts := globalOpts()
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	cfg, err := loadState(filepath.Join(opts.vaultDir, "..", "state", "zdai-state.json"))
	if err != nil {
		log.Errorf("zdai: load state: %v", err)
		store.finish(r, RunStatusFailed)
		return
	}

	// Email-driven ticket unblock check.
	if _emailRouter != nil {
		_emailRouter.checkBlockedTickets(ctx, opts.vaultDir, opts)
	}

	// Tess daily check.
	if cfg.Tess.Enabled {
		tessLastRun := filepath.Join(filepath.Dir(opts.logPath), "tess-last-run")
		if shouldRunTess(cfg.Tess.Schedule, tessLastRun) {
			if err := runTess(ctx, cfg.Tess, opts.claudeBin, opts.vaultDir, opts.logPath); err != nil {
				log.Errorf("zdai: tess: %v", err)
			} else {
				_ = markTessRan(tessLastRun)
			}
		}
	}

	requests, tickets, err := eligibleWork(opts.vaultDir)
	if err != nil {
		log.Errorf("zdai: eligible work: %v", err)
		store.finish(r, RunStatusFailed)
		return
	}

	if len(requests) == 0 && len(tickets) == 0 {
		appendLog(opts.logPath, "no eligible work this cycle", 0, 0)
		store.finish(r, RunStatusDone)
		return
	}

	for _, path := range requests {
		dispatchRequest(ctx, path, opts)
	}
	for _, path := range tickets {
		if err := dispatchTicket(ctx, path, opts.vaultDir, opts); err != nil {
			log.Errorf("zdai: dispatch %s: %v", path, err)
			appendLog(opts.logPath, fmt.Sprintf("skipped %s: %v", path, err), 1, 0)
			status = RunStatusFailed
		}
	}

	store.finish(r, status)
}

// dispatchMinutes are the minute-offsets within each hour when a cycle runs.
var dispatchMinutes = map[int]bool{7: true, 22: true, 37: true, 52: true}

// startScheduler runs a background goroutine that fires a dispatch cycle at
// :07, :22, :37, and :52 of every hour between 08:00 and 22:00 local time.
func startScheduler() {
	go func() {
		for {
			now := time.Now()
			next := now.Truncate(time.Minute).Add(time.Minute)
			time.Sleep(time.Until(next))

			t := time.Now()
			h, m := t.Hour(), t.Minute()
			if h >= 8 && h < 22 && dispatchMinutes[m] {
				go runCycle("scheduled")
			}
		}
	}()
}
