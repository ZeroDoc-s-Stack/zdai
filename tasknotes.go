// Pre-filters which agent-harness items are actually eligible for this cycle
// (per Harness/SKILL.md step 0 and 1.2/1.3) before the AI is invoked, so the
// prompt doesn't hand it the whole queue and a cycle with nothing eligible can
// skip AI invocation entirely.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const defaultMCPURL = "http://localhost:8080/mcp"

type taskSummary struct {
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	DateCreated string   `json:"dateCreated"`
	Tags        []string `json:"tags"`
}

var (
	agentStateHeadingRe = regexp.MustCompile(`(?m)^## Agent State\s*$`)
	agentStatusRe       = regexp.MustCompile(`(?m)^-\s*(?:stage|status):\s*(\S+)\s*$`)
)

func mcpToolCall(ctx context.Context, client *http.Client, mcpURL, name string, args map[string]any) (string, error) {
	reqBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": name, "arguments": args},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", mcpURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", err
	}

	// The MCP server replies with an SSE-framed body ("event: message\ndata:
	// {...}"); pull the JSON out of the "data:" line rather than parsing SSE
	// properly, since this server only ever sends a single message per call.
	payload := buf.Bytes()
	for _, line := range strings.Split(buf.String(), "\n") {
		if rest, ok := strings.CutPrefix(line, "data: "); ok {
			payload = []byte(rest)
			break
		}
	}

	var envelope struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", fmt.Errorf("decode mcp response: %w", err)
	}
	if envelope.Error != nil {
		return "", fmt.Errorf("mcp error: %s", envelope.Error.Message)
	}
	if len(envelope.Result.Content) == 0 {
		return "", fmt.Errorf("mcp response had no content")
	}
	return envelope.Result.Content[0].Text, nil
}

// queryByTag fetches every task carrying tag whose status is one of the four
// CTO-actionable values from SKILL.md's eligibility query, oldest first. The
// MCP query can't filter on the body's `## Agent State: status` (only the
// Go side can, by reading the file), so this is the cheap first pass.
func queryByTag(ctx context.Context, client *http.Client, mcpURL, tag string) ([]taskSummary, error) {
	args := map[string]any{
		"conjunction": "and",
		"children": []map[string]any{
			{"type": "condition", "id": "c1", "property": "tags", "operator": "contains", "value": tag},
			{"type": "group", "id": "g1", "conjunction": "or", "children": []map[string]any{
				{"type": "condition", "id": "c2", "property": "status", "operator": "is", "value": "open"},
				{"type": "condition", "id": "c3", "property": "status", "operator": "is", "value": "approved"},
				{"type": "condition", "id": "c4", "property": "status", "operator": "is", "value": "ready"},
				{"type": "condition", "id": "c5", "property": "status", "operator": "is", "value": "needs-rework"},
			}},
		},
	}
	text, err := mcpToolCall(ctx, client, mcpURL, "tasknotes_query_tasks", args)
	if err != nil {
		return nil, err
	}
	var result struct {
		All []taskSummary `json:"all"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("decode query_tasks result: %w", err)
	}

	var out []taskSummary
	for _, t := range result.All {
		// Template/example notes carry the same tags but have an unexpanded
		// "{{title}}" placeholder — skip them.
		if strings.Contains(t.Title, "{{") {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DateCreated < out[j].DateCreated })
	return out, nil
}

// readAgentStatus reads a ticket's `## Agent State` status line from disk.
// The body-text `stage`/`status` field (distinct from the TaskNotes frontmatter
// `status` field, which is the CTO approval gate) drives eligibility.
func readAgentStatus(vaultDir, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(vaultDir, path))
	if err != nil {
		return "", err
	}
	body := string(data)
	loc := agentStateHeadingRe.FindStringIndex(body)
	if loc == nil {
		return "", nil
	}
	section := body[loc[1]:]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	m := agentStatusRe.FindStringSubmatch(section)
	if m == nil {
		return "", nil
	}
	return m[1], nil
}

// readAgentKind reads the agent-kind tag from a ticket's YAML frontmatter.
// It scans the frontmatter tags array for a value matching "agent:<name>" or
// "agent-kind:<kind>" and returns the resolved tag value. "agent:<name>" takes
// precedence over "agent-kind:<kind>" as a direct persona override.
func readAgentKind(vaultDir, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(vaultDir, path))
	if err != nil {
		return "", err
	}
	body := string(data)

	// Find the YAML frontmatter block between the first two "---" lines.
	if !strings.HasPrefix(body, "---") {
		return "", nil
	}
	rest := body[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", nil
	}
	frontmatter := rest[:end]

	// Scan for tag values under the `tags:` key. We look for list items
	// (lines starting with "  - " or "- ") that match our prefixes.
	var directAgent string
	var kindAgent string
	for _, line := range strings.Split(frontmatter, "\n") {
		trimmed := strings.TrimSpace(line)
		tag, ok := strings.CutPrefix(trimmed, "- ")
		if !ok {
			continue
		}
		tag = strings.TrimSpace(tag)
		if v, ok := strings.CutPrefix(tag, "agent:"); ok && directAgent == "" {
			directAgent = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(tag, "agent-kind:"); ok && kindAgent == "" {
			kindAgent = strings.TrimSpace(v)
		}
	}
	if directAgent != "" {
		return directAgent, nil
	}
	return kindAgent, nil
}

// ticketEligible mirrors SKILL.md step 1.2/1.3: a queued ticket is always
// eligible; a stuck one only resumes on an explicit approved/ready/
// needs-rework status flip, never on agentStatus alone.
func ticketEligible(agentStatus, taskStatus string) bool {
	if taskStatus == "needs-rework" {
		return true
	}
	switch agentStatus {
	case "queued":
		return true
	case "blocked", "needs-handoff", "needs-approval":
		return taskStatus == "approved" || taskStatus == "ready"
	default:
		return false
	}
}

const eligibilityCap = 5

// eligibleWork queries both agent-request tasks and agent-queue tickets,
// applies the eligibility rule from SKILL.md, and returns at most
// eligibilityCap of each as vault-relative paths, oldest first.
func eligibleWork(ctx context.Context, client *http.Client, mcpURL, vaultDir string) (requests, tickets []string, err error) {
	requestCandidates, err := queryByTag(ctx, client, mcpURL, "agent-request")
	if err != nil {
		return nil, nil, fmt.Errorf("query agent-request: %w", err)
	}
	for _, t := range requestCandidates {
		requests = append(requests, t.Path)
		if len(requests) == eligibilityCap {
			break
		}
	}

	ticketCandidates, err := queryByTag(ctx, client, mcpURL, "agent-queue")
	if err != nil {
		return nil, nil, fmt.Errorf("query agent-queue: %w", err)
	}
	for _, t := range ticketCandidates {
		agentStatus, err := readAgentStatus(vaultDir, t.Path)
		if err != nil {
			continue
		}
		if !ticketEligible(agentStatus, t.Status) {
			continue
		}
		tickets = append(tickets, t.Path)
		if len(tickets) == eligibilityCap {
			break
		}
	}

	return requests, tickets, nil
}

func mcpHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
