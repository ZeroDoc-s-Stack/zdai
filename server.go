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
