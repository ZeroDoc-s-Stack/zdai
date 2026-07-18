package controllers

import (
	"context"
	"time"

	"github.com/zerodoc-s-stack/zdlib/base/logger"
	"github.com/zerodoc-s-stack/zdai/internal/models"
	pb "github.com/zerodoc-s-stack/zdai/package/grpc"
)

var log = logger.Log

// Zdai implements the gRPC handler for the zdai service.
// Business logic (dispatch, store, email routing) lives in the services package;
// this handler delegates to injected function values set at startup.
type Zdai struct {
	// RunCycleFn triggers a full dispatch cycle.
	RunCycleFn func(trigger string)
	// DispatchTicketFn runs a single ticket by vault-relative path.
	DispatchTicketFn func(ctx context.Context, path string) error
	// RegisterEmailThreadFn maps a ticket path to a Gmail thread ID.
	RegisterEmailThreadFn func(ticketPath, gmailThreadID string) error
	// EmailRoutingEnabled indicates whether email routing is active.
	EmailRoutingEnabled bool
	// ListRunsFn returns current run records.
	ListRunsFn func() []models.RunRecord
	// GetRunFn returns a single run by ID, or false if not found.
	GetRunFn func(id string) (models.RunRecord, bool)
}

func (z *Zdai) HealthCall(_ context.Context, _ *pb.HealthRequest, resp *pb.HealthResponse) error {
	resp.Status = "ok"
	return nil
}

func (z *Zdai) ListRunsCall(_ context.Context, _ *pb.ListRunsRequest, resp *pb.ListRunsResponse) error {
	records := z.ListRunsFn()
	// Return newest first.
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
	for _, r := range records {
		finished := ""
		if r.FinishedAt != nil {
			finished = r.FinishedAt.UTC().Format(time.RFC3339)
		}
		resp.Runs = append(resp.Runs, &pb.RunRecord{
			Id:         r.ID,
			Trigger:    r.Trigger,
			StartedAt:  r.StartedAt.UTC().Format(time.RFC3339),
			FinishedAt: finished,
			Status:     r.Status,
		})
	}
	return nil
}

func (z *Zdai) GetRunCall(_ context.Context, req *pb.GetRunRequest, resp *pb.GetRunResponse) error {
	r, ok := z.GetRunFn(req.Id)
	if !ok {
		resp.Found = false
		return nil
	}
	finished := ""
	if r.FinishedAt != nil {
		finished = r.FinishedAt.UTC().Format(time.RFC3339)
	}
	resp.Found = true
	resp.Run = &pb.RunRecord{
		Id:         r.ID,
		Trigger:    r.Trigger,
		StartedAt:  r.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt: finished,
		Status:     r.Status,
	}
	return nil
}

func (z *Zdai) DispatchCall(_ context.Context, req *pb.DispatchRequest, resp *pb.DispatchResponse) error {
	trigger := req.Trigger
	if trigger == "" {
		trigger = "api"
	}
	go z.RunCycleFn(trigger)
	resp.Message = "dispatch started"
	return nil
}

func (z *Zdai) AgentRunCall(ctx context.Context, req *pb.AgentRunRequest, resp *pb.AgentRunResponse) error {
	if req.Path == "" {
		resp.Message = "path is required"
		return nil
	}
	go func() {
		if err := z.DispatchTicketFn(ctx, req.Path); err != nil {
			log.Errorf("zdai: agent run %s: %v", req.Path, err)
		}
	}()
	resp.Message = "agent run started"
	resp.Path = req.Path
	return nil
}

func (z *Zdai) RegisterEmailThreadCall(_ context.Context, req *pb.RegisterEmailThreadRequest, resp *pb.RegisterEmailThreadResponse) error {
	if !z.EmailRoutingEnabled {
		return nil // ponytail: caller checks resp fields; returning error would surface via micro error handling
	}
	if err := z.RegisterEmailThreadFn(req.TicketPath, req.GmailThreadId); err != nil {
		return err
	}
	resp.TicketPath = req.TicketPath
	resp.GmailThreadId = req.GmailThreadId
	return nil
}
