// zdai is the Claude Code agent-harness service. It runs a background
// dispatch scheduler and exposes a gRPC API (go-micro v5) for triggering
// agent runs and querying run history.
package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/zerodoc-s-stack/zdai/handler"
	"github.com/zerodoc-s-stack/zdai/logger"
	zdutil "github.com/zerodoctor/zdgo-util"
)

var log = logger.Log

// defaultVaultDir is the Obsidian vault root on vp0dune (Syncthing target).
// Override via --vault-dir or VAULT_DIR env var for other hosts.
const defaultVaultDir = "/mnt/local/syncthing/data1"

func loadEnv() {
	log.SetLevel(logrus.InfoLevel)

	env := os.Getenv("ENV")
	if env != "prod" && env != "test" {
		env = "dev"
		log.SetLevel(logrus.DebugLevel)
	}
	os.Setenv("ENV", env)

	log.Infof("loading api [env=%s]...", env)
	godotenv.Load("." + env + ".env") //nolint:errcheck

	client, err := vault.New(
		vault.WithAddress(os.Getenv("VAULT_ADDRESS")),
		vault.WithRequestTimeout(30*time.Second),
	)
	if err != nil {
		log.Panicf("vault client init: %v", err)
	}

	log.Infof("approle login...")
	resp, err := client.Auth.AppRoleLogin(
		context.Background(),
		schema.AppRoleLoginRequest{
			RoleId:   os.Getenv("APPROLE_ID"),
			SecretId: os.Getenv("APPROLE_SECRET"),
		},
	)
	if err != nil {
		log.Panicf("vault approle login: %v", err)
	}

	log.Infof("reading secrets zdkey/%s/zdai...", env)
	secret, err := client.Secrets.KvV2Read(
		context.Background(),
		env+"/zdai",
		vault.WithMountPath("zdkey"),
		vault.WithToken(resp.Auth.ClientToken),
	)
	if err != nil {
		log.Panicf("vault read secrets: %v", err)
	}

	for k, v := range secret.Data.Data {
		os.Setenv(k, v.(string))
	}
}

func main() {
	vaultDir := flag.String("vault-dir", envOr("VAULT_DIR", defaultVaultDir), "Obsidian vault root")
	stateDir := flag.String("state-dir", envOr("STATE_DIR", ""), "state directory (run.lock, runs.log, zdai-state.json)")
	claudeBin := flag.String("claude-bin", "claude", "claude CLI binary")
	timeout := flag.Duration("timeout", 15*time.Minute, "max duration per claude invocation")
	flag.Parse()

	loadEnv()

	log.Infof("zdai: starting")
	defer log.Infof("zdai: stopped")

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
		if err == syscall.EWOULDBLOCK {
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

	_opts = dispatchOpts{
		vaultDir:  *vaultDir,
		claudeBin: *claudeBin,
		timeout:   *timeout,
		logPath:   logPath,
		model:     cfg.Harness.Model,
		effort:    cfg.Harness.Effort,
		provider:  cfg.Harness.Provider,
	}

	if cfg.EmailRouting.Enabled {
		snapFile := filepath.Join(*stateDir, "email-thread-snapshots.json")
		fetcher := newHTTPGmailFetcher(cfg.EmailRouting.GmailToken)
		r, err := newEmailRouter(fetcher, snapFile)
		if err != nil {
			log.Fatalf("zdai: init email router: %v", err)
		}
		_emailRouter = r
		log.Infof("zdai: email routing enabled, snapshots at %s", snapFile)
	}

	startScheduler()

	h := &handler.Zdai{
		EmailRoutingEnabled: cfg.EmailRouting.Enabled,
		RunCycleFn:          runCycle,
		DispatchTicketFn: func(ctx context.Context, path string) error {
			opts := globalOpts()
			r := store.begin("api")
			if err := dispatchTicket(ctx, path, opts.vaultDir, opts); err != nil {
				store.finish(r, RunStatusFailed)
				return err
			}
			store.finish(r, RunStatusDone)
			return nil
		},
		RegisterEmailThreadFn: func(ticketPath, gmailThreadID string) error {
			if _emailRouter == nil {
				return nil
			}
			return _emailRouter.registerThread(ticketPath, gmailThreadID)
		},
		ListRunsFn: func() []handler.RunRecord {
			runs := store.list()
			out := make([]handler.RunRecord, len(runs))
			for i, r := range runs {
				out[i] = handler.RunRecord{
					ID:         r.ID,
					Trigger:    r.Trigger,
					StartedAt:  r.StartedAt,
					FinishedAt: r.FinishedAt,
					Status:     string(r.Status),
				}
			}
			return out
		},
		GetRunFn: func(id string) (handler.RunRecord, bool) {
			r := store.get(id)
			if r == nil {
				return handler.RunRecord{}, false
			}
			return handler.RunRecord{
				ID:         r.ID,
				Trigger:    r.Trigger,
				StartedAt:  r.StartedAt,
				FinishedAt: r.FinishedAt,
				Status:     string(r.Status),
			}, true
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := StartMicro(ctx, h); err != nil {
		log.Fatalf("zdai: failed to start micro: %v", err)
	}

	zdutil.OnExit(func(s os.Signal, i ...interface{}) {
		cancel()
		<-ctx.Done()
		log.Warn("zdai: shutting down...")
		log.Info("zdai: server exiting...")
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
