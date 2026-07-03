package main

import (
	"sync"
	"time"
)

const maxRuns = 100

type runStore struct {
	mu   sync.RWMutex
	runs []*Run
}

var store = &runStore{}

func (s *runStore) begin(trigger string) *Run {
	r := &Run{
		ID:        time.Now().UTC().Format("20060102T150405"),
		Trigger:   trigger,
		StartedAt: time.Now().UTC(),
		Status:    RunStatusRunning,
	}
	s.mu.Lock()
	s.runs = append(s.runs, r)
	if len(s.runs) > maxRuns {
		s.runs = s.runs[len(s.runs)-maxRuns:]
	}
	s.mu.Unlock()
	return r
}

func (s *runStore) finish(r *Run, status RunStatus) {
	t := time.Now().UTC()
	s.mu.Lock()
	r.FinishedAt = &t
	r.Status = status
	s.mu.Unlock()
}

func (s *runStore) addAgentRun(r *Run, ar AgentRun) {
	s.mu.Lock()
	r.AgentRuns = append(r.AgentRuns, ar)
	s.mu.Unlock()
}

func (s *runStore) list() []*Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Run, len(s.runs))
	copy(out, s.runs)
	return out
}

func (s *runStore) get(id string) *Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.runs {
		if r.ID == id {
			return r
		}
	}
	return nil
}
