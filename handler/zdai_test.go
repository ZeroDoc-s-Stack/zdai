package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/zerodoc-s-stack/zdai/proto"
)

// newTestHandler returns a Zdai handler wired with no-op or stub functions.
func newTestHandler() *Zdai {
	now := time.Now()
	finished := now.Add(time.Second)
	return &Zdai{
		EmailRoutingEnabled: true,
		RunCycleFn:          func(trigger string) {},
		DispatchTicketFn:    func(ctx context.Context, path string) error { return nil },
		RegisterEmailThreadFn: func(ticketPath, gmailThreadID string) error {
			if ticketPath == "conflict" {
				return errors.New("already mapped")
			}
			return nil
		},
		ListRunsFn: func() []RunRecord {
			return []RunRecord{
				{ID: "a", Trigger: "test", StartedAt: now, FinishedAt: &finished, Status: "done"},
				{ID: "b", Trigger: "api", StartedAt: now, Status: "running"},
			}
		},
		GetRunFn: func(id string) (RunRecord, bool) {
			if id == "a" {
				return RunRecord{ID: "a", Trigger: "test", StartedAt: now, Status: "done"}, true
			}
			return RunRecord{}, false
		},
	}
}

func TestHealthCall(t *testing.T) {
	h := newTestHandler()
	resp := &pb.HealthResponse{}
	if err := h.HealthCall(context.Background(), &pb.HealthRequest{}, resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Errorf("got status %q, want %q", resp.Status, "ok")
	}
}

func TestListRunsCall(t *testing.T) {
	h := newTestHandler()
	resp := &pb.ListRunsResponse{}
	if err := h.ListRunsCall(context.Background(), &pb.ListRunsRequest{}, resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Runs) != 2 {
		t.Fatalf("got %d runs, want 2", len(resp.Runs))
	}
	// newest first: "b" should be index 0 after reverse
	if resp.Runs[0].Id != "b" {
		t.Errorf("expected newest-first ordering, got first id=%q", resp.Runs[0].Id)
	}
}

func TestGetRunCall(t *testing.T) {
	h := newTestHandler()

	t.Run("found", func(t *testing.T) {
		resp := &pb.GetRunResponse{}
		if err := h.GetRunCall(context.Background(), &pb.GetRunRequest{Id: "a"}, resp); err != nil {
			t.Fatal(err)
		}
		if !resp.Found || resp.Run.Id != "a" {
			t.Errorf("expected run a, got found=%v id=%q", resp.Found, resp.Run.GetId())
		}
	})

	t.Run("not found", func(t *testing.T) {
		resp := &pb.GetRunResponse{}
		if err := h.GetRunCall(context.Background(), &pb.GetRunRequest{Id: "missing"}, resp); err != nil {
			t.Fatal(err)
		}
		if resp.Found {
			t.Error("expected not found")
		}
	})
}

func TestDispatchCall(t *testing.T) {
	triggered := make(chan string, 1)
	h := newTestHandler()
	h.RunCycleFn = func(trigger string) { triggered <- trigger }

	resp := &pb.DispatchResponse{}
	if err := h.DispatchCall(context.Background(), &pb.DispatchRequest{Trigger: "test-trigger"}, resp); err != nil {
		t.Fatal(err)
	}
	if resp.Message != "dispatch started" {
		t.Errorf("got %q", resp.Message)
	}
	select {
	case got := <-triggered:
		if got != "test-trigger" {
			t.Errorf("got trigger %q, want %q", got, "test-trigger")
		}
	case <-time.After(time.Second):
		t.Error("RunCycleFn not called")
	}
}

func TestDispatchCall_DefaultTrigger(t *testing.T) {
	triggered := make(chan string, 1)
	h := newTestHandler()
	h.RunCycleFn = func(trigger string) { triggered <- trigger }

	if err := h.DispatchCall(context.Background(), &pb.DispatchRequest{}, &pb.DispatchResponse{}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-triggered:
		if got != "api" {
			t.Errorf("empty trigger should default to %q, got %q", "api", got)
		}
	case <-time.After(time.Second):
		t.Error("RunCycleFn not called")
	}
}

func TestAgentRunCall_EmptyPath(t *testing.T) {
	h := newTestHandler()
	resp := &pb.AgentRunResponse{}
	if err := h.AgentRunCall(context.Background(), &pb.AgentRunRequest{Path: ""}, resp); err != nil {
		t.Fatal(err)
	}
	if resp.Message != "path is required" {
		t.Errorf("got %q", resp.Message)
	}
}

func TestAgentRunCall_WithPath(t *testing.T) {
	dispatched := make(chan string, 1)
	h := newTestHandler()
	h.DispatchTicketFn = func(_ context.Context, path string) error {
		dispatched <- path
		return nil
	}

	resp := &pb.AgentRunResponse{}
	if err := h.AgentRunCall(context.Background(), &pb.AgentRunRequest{Path: "some/ticket.md"}, resp); err != nil {
		t.Fatal(err)
	}
	if resp.Message != "agent run started" {
		t.Errorf("got %q", resp.Message)
	}
	select {
	case got := <-dispatched:
		if got != "some/ticket.md" {
			t.Errorf("got path %q", got)
		}
	case <-time.After(time.Second):
		t.Error("DispatchTicketFn not called")
	}
}

func TestRegisterEmailThreadCall(t *testing.T) {
	h := newTestHandler()

	t.Run("success", func(t *testing.T) {
		resp := &pb.RegisterEmailThreadResponse{}
		err := h.RegisterEmailThreadCall(context.Background(), &pb.RegisterEmailThreadRequest{
			TicketPath:    "TaskNotes/AI/Tickets/foo.md",
			GmailThreadId: "thread123",
		}, resp)
		if err != nil {
			t.Fatal(err)
		}
		if resp.TicketPath != "TaskNotes/AI/Tickets/foo.md" {
			t.Errorf("got ticket_path %q", resp.TicketPath)
		}
	})

	t.Run("conflict", func(t *testing.T) {
		resp := &pb.RegisterEmailThreadResponse{}
		err := h.RegisterEmailThreadCall(context.Background(), &pb.RegisterEmailThreadRequest{
			TicketPath:    "conflict",
			GmailThreadId: "thread456",
		}, resp)
		if err == nil {
			t.Error("expected error for conflicting mapping")
		}
	})

	t.Run("routing disabled", func(t *testing.T) {
		h2 := newTestHandler()
		h2.EmailRoutingEnabled = false
		resp := &pb.RegisterEmailThreadResponse{}
		if err := h2.RegisterEmailThreadCall(context.Background(), &pb.RegisterEmailThreadRequest{
			TicketPath:    "any/path",
			GmailThreadId: "thread789",
		}, resp); err != nil {
			t.Fatal("expected no error when routing disabled, got:", err)
		}
	})
}
