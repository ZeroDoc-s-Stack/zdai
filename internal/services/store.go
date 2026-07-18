package services

import (
	"sync"
	"time"

	"github.com/zerodoc-s-stack/zdai/internal/models"
)

const maxRuns = 100

// RunStore is an in-process ring buffer of recent dispatch runs.
type RunStore struct {
	mu   sync.RWMutex
	runs []*models.Run
}

// Store is the package-level run history, initialised at startup.
var Store = &RunStore{}

func (s *RunStore) Begin(trigger string) *models.Run {
	r := &models.Run{
		ID:        time.Now().UTC().Format("20060102T150405"),
		Trigger:   trigger,
		StartedAt: time.Now().UTC(),
		Status:    models.RunStatusRunning,
	}
	s.mu.Lock()
	s.runs = append(s.runs, r)
	if len(s.runs) > maxRuns {
		s.runs = s.runs[len(s.runs)-maxRuns:]
	}
	s.mu.Unlock()
	return r
}

func (s *RunStore) Finish(r *models.Run, status models.RunStatus) {
	t := time.Now().UTC()
	s.mu.Lock()
	r.FinishedAt = &t
	r.Status = status
	s.mu.Unlock()
}

func (s *RunStore) AddAgentRun(r *models.Run, ar models.AgentRun) {
	s.mu.Lock()
	r.AgentRuns = append(r.AgentRuns, ar)
	s.mu.Unlock()
}

func (s *RunStore) List() []*models.Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Run, len(s.runs))
	copy(out, s.runs)
	return out
}

func (s *RunStore) Get(id string) *models.Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.runs {
		if r.ID == id {
			return r
		}
	}
	return nil
}
