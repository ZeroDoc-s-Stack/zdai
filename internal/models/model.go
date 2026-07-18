package models

import "time"

type RunStatus string

const (
	RunStatusRunning RunStatus = "running"
	RunStatusDone    RunStatus = "done"
	RunStatusFailed  RunStatus = "failed"
)

type Run struct {
	ID         string     `json:"id"`
	Trigger    string     `json:"trigger"` // "scheduled", "api", "tess"
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Status     RunStatus  `json:"status"`
	AgentRuns  []AgentRun `json:"agent_runs"`
}

type AgentRun struct {
	Path      string        `json:"path"`
	Persona   string        `json:"persona"`
	Model     string        `json:"model"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration_ms"` // milliseconds for JSON
	ExitCode  int           `json:"exit_code"`
	Output    string        `json:"output"` // last 4000 chars
}

// RunRecord is a transfer object passed between the service store and the
// gRPC controller to avoid carrying proto types into business logic.
type RunRecord struct {
	ID         string
	Trigger    string
	StartedAt  time.Time
	FinishedAt *time.Time
	Status     string
}
