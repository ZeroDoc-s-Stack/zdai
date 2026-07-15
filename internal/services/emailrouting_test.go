package services

// Tests for reply-detection and latest-reply-isolation logic in emailrouting.go.
// Does NOT exercise HTTP or file I/O — only the pure algorithmic functions.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helpers --------------------------------------------------------------------

func msg(id, sender string, offsetSec int) threadMsg {
	return threadMsg{
		MessageID: id,
		Sender:    sender,
		SentAt:    time.Date(2026, 7, 4, 10, 0, offsetSec, 0, time.UTC),
	}
}

const sys = systemEmailSender
const user = "danielcastrolocal@gmail.com"
const user2 = "danielrcastro10@gmail.com"

// --- latestUnhandledReply ---------------------------------------------------

func TestLatestUnhandledReply(t *testing.T) {
	cases := []struct {
		name        string
		msgs        []threadMsg
		handledUpTo string
		wantID      string // "" means no reply expected
	}{
		{
			name:   "empty thread",
			msgs:   nil,
			wantID: "",
		},
		{
			name:   "only inbound, no outbound",
			msgs:   []threadMsg{msg("i1", user, 0)},
			wantID: "",
		},
		{
			name:   "only outbound, no inbound",
			msgs:   []threadMsg{msg("s1", sys, 0)},
			wantID: "",
		},
		{
			name:   "inbound before outbound, not after",
			msgs:   []threadMsg{msg("i1", user, 0), msg("s1", sys, 10)},
			wantID: "",
		},
		{
			name:   "outbound then inbound — new reply",
			msgs:   []threadMsg{msg("s1", sys, 0), msg("i1", user, 10)},
			wantID: "i1",
		},
		{
			name:   "multiple inbound after outbound — returns newest",
			msgs:   []threadMsg{msg("s1", sys, 0), msg("i1", user, 10), msg("i2", user2, 20)},
			wantID: "i2",
		},
		{
			name:        "inbound already handled via HandledUpTo",
			msgs:        []threadMsg{msg("s1", sys, 0), msg("i1", user, 10)},
			handledUpTo: "i1",
			wantID:      "",
		},
		{
			name:        "second inbound after handled first — returns second",
			msgs:        []threadMsg{msg("s1", sys, 0), msg("i1", user, 10), msg("i2", user, 20)},
			handledUpTo: "i1",
			wantID:      "i2",
		},
		{
			name: "outbound after inbound resets window — inbound before second outbound ignored",
			//  i1(user) s1(sys) i2(user) s2(sys) i3(user)
			// lastOutboundIdx = s2 (idx 3); handledUpTo = ""; minIdx = 3
			// only i3 (idx 4) qualifies
			msgs: []threadMsg{
				msg("i1", user, 0),
				msg("s1", sys, 10),
				msg("i2", user, 20),
				msg("s2", sys, 30),
				msg("i3", user, 40),
			},
			wantID: "i3",
		},
		{
			name: "multiple dispatch cycles: HandledUpTo from first cycle, new reply in second",
			// Cycle 1: [s1, i1] → handledUpTo = i1
			// Cycle 2: [s1, i1, s2, i2] → lastOutboundIdx = s2(idx 2), handledUpToIdx = i1(idx 1)
			//   minIdx = 2, look at i2(idx 3) → returns i2
			msgs: []threadMsg{
				msg("s1", sys, 0),
				msg("i1", user, 10),
				msg("s2", sys, 20),
				msg("i2", user, 30),
			},
			handledUpTo: "i1",
			wantID:      "i2",
		},
		{
			name: "HandledUpTo points past last outbound — still isolates correctly",
			// handledUpTo = i2 (idx 3), lastOutboundIdx = s1 (idx 0)
			// minIdx = 3 (handledUpTo wins); only i3 (idx 4) qualifies
			msgs: []threadMsg{
				msg("s1", sys, 0),
				msg("i1", user, 10),
				msg("i2", user, 20),
				msg("i3", user, 30),
			},
			handledUpTo: "i2",
			wantID:      "i3",
		},
		{
			name: "HandledUpTo not found in thread — falls back to lastOutbound boundary",
			// handledUpToIdx = -1 (unknown ID), lastOutboundIdx = s1 (idx 0)
			// minIdx = 0; i1 (idx 1) qualifies
			msgs:        []threadMsg{msg("s1", sys, 0), msg("i1", user, 10)},
			handledUpTo: "does-not-exist",
			wantID:      "i1",
		},
		{
			name: "system messages interspersed — last outbound is what matters",
			// s1 i1 s2 i2 s3 — last outbound is s3 (idx 4), no inbound after it
			msgs: []threadMsg{
				msg("s1", sys, 0),
				msg("i1", user, 10),
				msg("s2", sys, 20),
				msg("i2", user, 30),
				msg("s3", sys, 40),
			},
			wantID: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := latestUnhandledReply(tc.msgs, tc.handledUpTo)
			if tc.wantID == "" {
				if ok {
					t.Errorf("expected no reply, got MessageID=%q", got.MessageID)
				}
				return
			}
			if !ok {
				t.Errorf("expected reply %q, got none", tc.wantID)
				return
			}
			if got.MessageID != tc.wantID {
				t.Errorf("got MessageID=%q, want %q", got.MessageID, tc.wantID)
			}
		})
	}
}

// --- setTicketFrontmatterStatus ---------------------------------------------

// writeTestTicket creates a temp vault dir containing a ticket file and
// returns the vault dir and vault-relative path.
func writeTestTicket(t *testing.T, content string) (vaultDir, relPath string) {
	t.Helper()
	dir := t.TempDir()
	relPath = "ticket.md"
	if err := os.WriteFile(filepath.Join(dir, relPath), []byte(content), 0o644); err != nil {
		t.Fatalf("write test ticket: %v", err)
	}
	return dir, relPath
}

func TestSetTicketFrontmatterStatus(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		newStatus string
		wantErr   bool
		wantLine  string // expected "status: <x>" line after mutation
	}{
		{
			name:      "blocked to in-progress",
			input:     "---\ntitle: Foo\nstatus: blocked\ntags:\n  - agent-queue\n---\n\n## Goal\n",
			newStatus: "in-progress",
			wantLine:  "status: in-progress",
		},
		{
			name:      "open to in-progress",
			input:     "---\ntitle: Bar\nstatus: open\n---\n\nbody",
			newStatus: "in-progress",
			wantLine:  "status: in-progress",
		},
		{
			name:      "preserves rest of frontmatter",
			input:     "---\ntitle: Bar\nstatus: blocked\npriority: high\n---\n\nbody",
			newStatus: "done",
			wantLine:  "status: done",
		},
		{
			name:      "no frontmatter returns error",
			input:     "## Goal\n\nno frontmatter",
			newStatus: "in-progress",
			wantErr:   true,
		},
		{
			name:      "missing status field returns error",
			input:     "---\ntitle: Foo\n---\n\nbody",
			newStatus: "in-progress",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vaultDir, relPath := writeTestTicket(t, tc.input)
			err := setTicketFrontmatterStatus(vaultDir, relPath, tc.newStatus)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			data, _ := os.ReadFile(filepath.Join(vaultDir, relPath))
			content := string(data)
			if !strings.Contains(content, tc.wantLine) {
				t.Errorf("file does not contain %q\nfull content:\n%s", tc.wantLine, content)
			}
			// Verify frontmatter is still well-formed by parsing it.
			fm, ok := parseFrontmatter(data)
			if !ok {
				t.Error("frontmatter became unparseable after mutation")
			}
			if fm.Status != tc.newStatus {
				t.Errorf("parseFrontmatter returned status=%q, want %q", fm.Status, tc.newStatus)
			}
		})
	}
}

// --- registerThread 1:1 enforcement ----------------------------------------

func TestRegisterThread_OneToOneEnforcement(t *testing.T) {
	// Use an in-memory-only router (no real file I/O).
	tmp := t.TempDir()
	snapFile := filepath.Join(tmp, "snaps.json")
	r := &EmailRouter{
		fetcher:   nil, // not needed for registration tests
		snapFile:  snapFile,
		snapshots: make(map[string]threadSnapshot),
	}

	// First registration succeeds.
	if err := r.registerThread("tickets/a.md", "thread-1"); err != nil {
		t.Fatalf("first register: %v", err)
	}

	// Re-registering same pair is idempotent.
	if err := r.registerThread("tickets/a.md", "thread-1"); err != nil {
		t.Errorf("idempotent re-register: %v", err)
	}

	// Different thread for same ticket is rejected.
	if err := r.registerThread("tickets/a.md", "thread-2"); err == nil {
		t.Error("expected error when ticket already mapped to different thread, got nil")
	}

	// Same thread for different ticket is rejected.
	if err := r.registerThread("tickets/b.md", "thread-1"); err == nil {
		t.Error("expected error when thread already mapped to different ticket, got nil")
	}

	// Second ticket with its own thread succeeds.
	if err := r.registerThread("tickets/b.md", "thread-2"); err != nil {
		t.Fatalf("second distinct register: %v", err)
	}
}

// --- parseThreadResponse ----------------------------------------------------

func TestParseThreadResponse_SenderExtraction(t *testing.T) {
	tr := gmailThreadResponse{
		ID: "thread-abc",
		Messages: []gmailMsgMeta{
			{
				ID:           "m1",
				InternalDate: "1751616000000", // 2025-07-04T08:00:00Z in ms
				Payload: gmailMsgPayload{Headers: []gmailHeader{
					{Name: "From", Value: "Agent <zd.agents@gmail.com>"},
				}},
			},
			{
				ID:           "m2",
				InternalDate: "1751616060000",
				Payload: gmailMsgPayload{Headers: []gmailHeader{
					{Name: "From", Value: "Daniel Castro <danielcastrolocal@gmail.com>"},
				}},
			},
			{
				ID:           "m3",
				InternalDate: "bad-date", // malformed — should be skipped
				Payload: gmailMsgPayload{Headers: []gmailHeader{
					{Name: "From", Value: "x@example.com"},
				}},
			},
		},
	}

	msgs, err := parseThreadResponse(tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2 (malformed-date entry should be skipped)", len(msgs))
	}
	if msgs[0].Sender != systemEmailSender {
		t.Errorf("msgs[0].Sender = %q, want %q", msgs[0].Sender, systemEmailSender)
	}
	if msgs[1].Sender != "danielcastrolocal@gmail.com" {
		t.Errorf("msgs[1].Sender = %q, want %q", msgs[1].Sender, "danielcastrolocal@gmail.com")
	}
}
