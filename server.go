package main

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", handleHealth)

	v1 := r.Group("/v1")
	{
		v1.GET("/runs", handleListRuns)
		v1.GET("/runs/:id", handleGetRun)
		v1.POST("/dispatch", handleDispatch)
		v1.POST("/agents/run", handleAgentRun)
		v1.POST("/email/threads", handleRegisterEmailThread)
	}
	return r
}

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleListRuns(c *gin.Context) {
	all := store.list()
	// Return newest first.
	slices.Reverse(all)
	c.JSON(http.StatusOK, all)
}

func handleGetRun(c *gin.Context) {
	r := store.get(c.Param("id"))
	if r == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	c.JSON(http.StatusOK, r)
}

// handleDispatch triggers a full dispatch cycle immediately, independent of
// the scheduler. Returns the new run ID so callers can poll /v1/runs/:id.
func handleDispatch(c *gin.Context) {
	go runCycle("api")
	c.JSON(http.StatusAccepted, gin.H{"message": "dispatch started"})
}

// agentRunRequest is the body for POST /v1/agents/run.
type agentRunRequest struct {
	Path string `json:"path" binding:"required"` // vault-relative path to the ticket
}

// registerThreadRequest is the body for POST /v1/email/threads.
type registerThreadRequest struct {
	TicketPath    string `json:"ticket_path" binding:"required"`
	GmailThreadID string `json:"gmail_thread_id" binding:"required"`
}

// handleRegisterEmailThread registers a 1:1 mapping between a ticket path and
// a Gmail thread ID. Returns 409 if either side already maps to a different partner.
func handleRegisterEmailThread(c *gin.Context) {
	if _emailRouter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "email routing not enabled"})
		return
	}
	var req registerThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := _emailRouter.registerThread(req.TicketPath, req.GmailThreadID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"ticket_path":     req.TicketPath,
		"gmail_thread_id": req.GmailThreadID,
	})
}

// handleAgentRun runs a single agent on the given ticket path, bypassing
// the eligibility filter. Useful for manual re-runs or testing.
func handleAgentRun(c *gin.Context) {
	var req agentRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	go func() {
		r := store.begin("api")
		opts := globalOpts()
		if err := dispatchTicket(c.Request.Context(), req.Path, opts.vaultDir, opts); err != nil {
			store.finish(r, RunStatusFailed)
			return
		}
		store.finish(r, RunStatusDone)
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "agent run started", "path": req.Path})
}
