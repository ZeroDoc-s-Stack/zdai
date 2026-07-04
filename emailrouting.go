package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const systemEmailSender = "zd.agents@gmail.com"
const gmailAPIBase = "https://gmail.googleapis.com/gmail/v1"

// threadMsg is the minimal metadata for one message in a Gmail thread.
// Body content is never stored here — only the ID, sender address, and timestamp.
type threadMsg struct {
	MessageID string    // Gmail message ID (not the RFC 2822 Message-ID header)
	Sender    string    // From address, lower-cased and addr-only
	SentAt    time.Time // from internalDate (ms since epoch)
}

// threadSnapshot records the last-handled state of the Gmail thread
// paired with a ticket. Persisted across cycles so a handled reply is
// never re-triggered even after a service restart.
type threadSnapshot struct {
	TicketPath    string    `json:"ticket_path"`
	GmailThreadID string    `json:"gmail_thread_id"`
	HandledUpTo   string    `json:"handled_up_to"`   // MessageID of last dispatched reply; "" = none
	LastCheckedAt time.Time `json:"last_checked_at"` // UTC
}

// latestUnhandledReply returns the newest non-system reply that arrived
// AFTER the last outbound system message AND has not yet been dispatched.
//
// Algorithm:
//  1. Find lastOutboundIdx: the index of the last message from systemEmailSender.
//     If no outbound exists, return (zero, false) — there is nothing to reply to yet.
//  2. Find handledUpToIdx: the index of the message whose ID matches handledUpTo.
//     -1 when handledUpTo is empty (no previous dispatch).
//  3. minIdx = max(lastOutboundIdx, handledUpToIdx). Every message at index ≤ minIdx
//     is already covered by prior dispatches.
//  4. Walk messages at index > minIdx; collect non-system ones; return the last
//     (newest) such message.
//
// Isolation guarantee: only the single newest unhandled reply is returned.
// Historical reply loops from previous blocked cycles are excluded because
// HandledUpTo advances to the reply MessageID on each dispatch.
func latestUnhandledReply(msgs []threadMsg, handledUpTo string) (threadMsg, bool) {
	// Step 1: require at least one outbound system message.
	lastOutboundIdx := -1
	for i, m := range msgs {
		if m.Sender == systemEmailSender {
			lastOutboundIdx = i
		}
	}
	if lastOutboundIdx == -1 {
		return threadMsg{}, false
	}

	// Step 2.
	handledUpToIdx := -1
	if handledUpTo != "" {
		for i, m := range msgs {
			if m.MessageID == handledUpTo {
				handledUpToIdx = i
				break
			}
		}
	}

	// Step 3.
	minIdx := max(lastOutboundIdx, handledUpToIdx)

	// Step 4: newest non-system message after minIdx.
	var latest threadMsg
	found := false
	for i := minIdx + 1; i < len(msgs); i++ {
		if msgs[i].Sender != systemEmailSender {
			latest = msgs[i]
			found = true
		}
	}
	return latest, found
}

// -----------------------------------------------------------------------------
// Gmail fetcher
// -----------------------------------------------------------------------------

// gmailFetcher abstracts Gmail API calls so business logic is testable without HTTP.
type gmailFetcher interface {
	threadMessages(ctx context.Context, threadID string) ([]threadMsg, error)
}

// httpGmailFetcher calls the Gmail REST API using net/http.
// It requests only message metadata — never fetches body content.
type httpGmailFetcher struct {
	token string
	hc    *http.Client
}

func newHTTPGmailFetcher(token string) *httpGmailFetcher {
	return &httpGmailFetcher{
		token: token,
		hc:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (f *httpGmailFetcher) threadMessages(ctx context.Context, threadID string) ([]threadMsg, error) {
	url := fmt.Sprintf(
		"%s/users/me/threads/%s?format=metadata&metadataHeaders=From",
		gmailAPIBase, threadID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+f.token)

	resp, err := f.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gmail API status %d for thread %s", resp.StatusCode, threadID)
	}

	var tr gmailThreadResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return parseThreadResponse(tr)
}

// gmailThreadResponse and friends capture only the JSON fields we need.
type gmailThreadResponse struct {
	ID       string         `json:"id"`
	Messages []gmailMsgMeta `json:"messages"`
}

type gmailMsgMeta struct {
	ID           string          `json:"id"`
	InternalDate string          `json:"internalDate"` // ms since epoch, as string
	Payload      gmailMsgPayload `json:"payload"`
}

type gmailMsgPayload struct {
	Headers []gmailHeader `json:"headers"`
}

type gmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// parseThreadResponse converts the raw Gmail thread response to []threadMsg.
// Messages with a malformed internalDate are skipped (best-effort).
func parseThreadResponse(tr gmailThreadResponse) ([]threadMsg, error) {
	out := make([]threadMsg, 0, len(tr.Messages))
	for _, m := range tr.Messages {
		ms, err := strconv.ParseInt(m.InternalDate, 10, 64)
		if err != nil {
			continue
		}
		sender := extractFromAddress(m.Payload.Headers)
		out = append(out, threadMsg{
			MessageID: m.ID,
			Sender:    sender,
			SentAt:    time.UnixMilli(ms).UTC(),
		})
	}
	return out, nil
}

// extractFromAddress finds the From header and returns the lower-cased
// email address portion, falling back to the raw value on parse failure.
func extractFromAddress(headers []gmailHeader) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, "From") {
			if addr, err := mail.ParseAddress(h.Value); err == nil {
				return strings.ToLower(addr.Address)
			}
			return strings.ToLower(strings.TrimSpace(h.Value))
		}
	}
	return ""
}

// -----------------------------------------------------------------------------
// emailRouter
// -----------------------------------------------------------------------------

// emailRouter manages thread↔ticket mappings and drives the unblock workflow.
// One instance lives for the lifetime of the process.
type emailRouter struct {
	mu        sync.Mutex
	fetcher   gmailFetcher
	snapFile  string                     // path to JSON file storing snapshots
	snapshots map[string]threadSnapshot  // keyed by ticket path
}

// _emailRouter is the package-level instance initialised in main (nil when disabled).
var _emailRouter *emailRouter

func newEmailRouter(fetcher gmailFetcher, snapFile string) (*emailRouter, error) {
	r := &emailRouter{
		fetcher:   fetcher,
		snapFile:  snapFile,
		snapshots: make(map[string]threadSnapshot),
	}
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load snapshots: %w", err)
	}
	return r, nil
}

func (r *emailRouter) load() error {
	data, err := os.ReadFile(r.snapFile)
	if err != nil {
		return err
	}
	var snaps []threadSnapshot
	if err := json.Unmarshal(data, &snaps); err != nil {
		return err
	}
	for _, s := range snaps {
		r.snapshots[s.TicketPath] = s
	}
	return nil
}

func (r *emailRouter) save() error {
	snaps := make([]threadSnapshot, 0, len(r.snapshots))
	for _, s := range r.snapshots {
		snaps = append(snaps, s)
	}
	data, err := json.Marshal(snaps)
	if err != nil {
		return err
	}
	// ponytail: 0o600 — snapshot file contains no secrets, but thread IDs are internal
	return os.WriteFile(r.snapFile, data, 0o600)
}

// registerThread establishes the strict 1:1 mapping between a ticket path and
// a Gmail thread ID. Returns an error if either side already maps to a
// different partner (prevents accidental cross-wiring).
func (r *emailRouter) registerThread(ticketPath, gmailThreadID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if snap, ok := r.snapshots[ticketPath]; ok && snap.GmailThreadID != gmailThreadID {
		return fmt.Errorf("ticket %q already mapped to thread %q", ticketPath, snap.GmailThreadID)
	}
	for path, snap := range r.snapshots {
		if snap.GmailThreadID == gmailThreadID && path != ticketPath {
			return fmt.Errorf("thread %q already mapped to ticket %q", gmailThreadID, path)
		}
	}

	r.snapshots[ticketPath] = threadSnapshot{
		TicketPath:    ticketPath,
		GmailThreadID: gmailThreadID,
	}
	return r.save()
}

// checkBlockedTickets is called each dispatch cycle. For every ticket that is
// blocked and has a registered Gmail thread, it fetches the thread, checks for
// a new reply, and if found transitions the ticket blocked→in-progress before
// dispatching the agent.
func (r *emailRouter) checkBlockedTickets(ctx context.Context, vaultDir string, opts dispatchOpts) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for ticketPath, snap := range r.snapshots {
		status, err := readTicketFrontmatterStatus(vaultDir, ticketPath)
		if err != nil {
			log.Errorf("zdai: email-routing: read status %s: %v", ticketPath, err)
			continue
		}
		if status != "blocked" {
			continue
		}

		msgs, err := r.fetcher.threadMessages(ctx, snap.GmailThreadID)
		if err != nil {
			// ponytail: log only thread ID and ticket path, never message content
			log.Errorf("zdai: email-routing: fetch thread %s (ticket %s): %v",
				snap.GmailThreadID, ticketPath, err)
			continue
		}

		reply, ok := latestUnhandledReply(msgs, snap.HandledUpTo)
		if !ok {
			continue
		}

		// Transition blocked → in-progress BEFORE dispatch (SIAA state machine).
		if err := setTicketFrontmatterStatus(vaultDir, ticketPath, "in-progress"); err != nil {
			log.Errorf("zdai: email-routing: set in-progress %s: %v", ticketPath, err)
			continue
		}
		log.Infof("zdai: email-routing: unblocked ticket %s via thread %s msg %s sent-at %s",
			ticketPath, snap.GmailThreadID, reply.MessageID, reply.SentAt.Format(time.RFC3339))

		// Advance snapshot before dispatch so a crash/timeout cannot re-trigger
		// the same reply on the next cycle.
		snap.HandledUpTo = reply.MessageID
		snap.LastCheckedAt = time.Now().UTC()
		r.snapshots[ticketPath] = snap
		if err := r.save(); err != nil {
			log.Errorf("zdai: email-routing: save snapshots: %v", err)
		}

		if err := r.dispatchUnblockedTicket(ctx, ticketPath, reply, opts); err != nil {
			log.Errorf("zdai: email-routing: dispatch %s: %v", ticketPath, err)
		}
	}
}

// dispatchUnblockedTicket resolves the ticket's persona and invokes claude with
// a prompt that names the new reply by message ID only. Body retrieval is the
// agent's responsibility — we never include or log body content here.
func (r *emailRouter) dispatchUnblockedTicket(ctx context.Context, ticketPath string, reply threadMsg, opts dispatchOpts) error {
	p, ok := resolvePersona(opts.vaultDir, ticketPath)
	if !ok {
		return fmt.Errorf("no agent-kind tag in %s", ticketPath)
	}
	// ponytail: sender address is routing metadata, not body content — safe to include
	prompt := fmt.Sprintf(
		"Execute the ticket at: %s\n\n"+
			"Email unblock context: a new reply has arrived on this ticket's Gmail thread "+
			"(message ID: %s, from: %s, at: %s). This is the only new content since the "+
			"last outbound system message. Retrieve and process this reply to continue work.",
		ticketPath, reply.MessageID, reply.Sender, reply.SentAt.UTC().Format(time.RFC3339),
	)
	return invokeAgent(ctx, p, prompt, opts.vaultDir, opts.claudeBin, opts.effort, opts.provider, opts.logPath)
}

// -----------------------------------------------------------------------------
// Ticket frontmatter helpers
// -----------------------------------------------------------------------------

// readTicketFrontmatterStatus reads the status field from a ticket's YAML frontmatter.
func readTicketFrontmatterStatus(vaultDir, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(vaultDir, path))
	if err != nil {
		return "", err
	}
	fm, ok := parseFrontmatter(data)
	if !ok {
		return "", fmt.Errorf("no frontmatter in %s", path)
	}
	return fm.Status, nil
}

// setTicketFrontmatterStatus rewrites the status field in a ticket's YAML
// frontmatter without a full YAML round-trip, preserving all other formatting.
func setTicketFrontmatterStatus(vaultDir, path, newStatus string) error {
	fullPath := filepath.Join(vaultDir, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return fmt.Errorf("no frontmatter in %s", path)
	}
	// rest starts immediately after the opening "---"
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return fmt.Errorf("unterminated frontmatter in %s", path)
	}

	lines := strings.Split(rest[:end], "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "status:") {
			lines[i] = "status: " + newStatus
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("status field not found in frontmatter of %s", path)
	}

	updated := "---" + strings.Join(lines, "\n") + rest[end:]
	return os.WriteFile(fullPath, []byte(updated), 0o644)
}
