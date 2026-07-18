package services

import (
	"os"
	"path/filepath"
	"testing"
)

// --- ticketEligible -------------------------------------------------------

func TestTicketEligible(t *testing.T) {
	cases := []struct {
		name        string
		agentStatus string
		taskStatus  string
		want        bool
	}{
		{"needs-rework taskStatus with queued agentStatus", "queued", "needs-rework", true},
		{"needs-rework taskStatus with blocked agentStatus", "blocked", "needs-rework", true},
		{"needs-rework taskStatus with empty agentStatus", "", "needs-rework", true},
		{"needs-rework taskStatus with done agentStatus", "done", "needs-rework", true},

		{"queued agentStatus with open taskStatus", "queued", "open", true},
		{"queued agentStatus with approved taskStatus", "queued", "approved", true},
		{"queued agentStatus with ready taskStatus", "queued", "ready", true},
		{"queued agentStatus with blocked taskStatus", "queued", "blocked", true},
		{"queued agentStatus with done taskStatus", "queued", "done", true},

		{"blocked agentStatus with approved taskStatus", "blocked", "approved", true},
		{"blocked agentStatus with ready taskStatus", "blocked", "ready", true},
		{"blocked agentStatus with open taskStatus", "blocked", "open", false},
		{"blocked agentStatus with done taskStatus", "blocked", "done", false},
		{"needs-handoff agentStatus with approved taskStatus", "needs-handoff", "approved", true},
		{"needs-handoff agentStatus with ready taskStatus", "needs-handoff", "ready", true},
		{"needs-handoff agentStatus with open taskStatus", "needs-handoff", "open", false},
		{"needs-approval agentStatus with approved taskStatus", "needs-approval", "approved", true},
		{"needs-approval agentStatus with ready taskStatus", "needs-approval", "ready", true},
		{"needs-approval agentStatus with open taskStatus", "needs-approval", "open", false},

		{"done agentStatus with approved taskStatus", "done", "approved", false},
		{"done agentStatus with open taskStatus", "done", "open", false},
		{"failed agentStatus with ready taskStatus", "failed", "ready", false},
		{"in-progress agentStatus with open taskStatus", "in-progress", "open", false},
		{"empty agentStatus with open taskStatus", "", "open", false},
		{"empty agentStatus with approved taskStatus", "", "approved", false},
		{"unknown agentStatus with approved taskStatus", "some-unknown-value", "approved", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ticketEligible(c.agentStatus, c.taskStatus)
			if got != c.want {
				t.Errorf("ticketEligible(%q, %q) = %v, want %v", c.agentStatus, c.taskStatus, got, c.want)
			}
		})
	}
}

// --- helpers ---------------------------------------------------------------

func writeTempTicket(t *testing.T, content string) (vaultDir, relPath string) {
	t.Helper()
	dir := t.TempDir()
	relPath = "ticket.md"
	if err := os.WriteFile(filepath.Join(dir, relPath), []byte(content), 0o644); err != nil {
		t.Fatalf("write temp ticket: %v", err)
	}
	return dir, relPath
}

// --- readAgentStatus --------------------------------------------------------

func TestReadAgentStatus_StageField(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Goal\n\nDo the thing.\n\n## Agent State\n\n- stage: queued\n- iterations: 0\n\n## Agent Log\n")
	got, err := readAgentStatus(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "queued" {
		t.Errorf("got %q, want %q", got, "queued")
	}
}

func TestReadAgentStatus_StatusField(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Goal\n\nDo the thing.\n\n## Agent State\n\n- status: needs-approval\n- iterations: 2\n\n## Agent Log\n")
	got, err := readAgentStatus(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "needs-approval" {
		t.Errorf("got %q, want %q", got, "needs-approval")
	}
}

func TestReadAgentStatus_MissingSection(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Goal\n\nDo the thing.\n\n## Acceptance Criteria\n\n- [ ] done\n")
	got, err := readAgentStatus(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestReadAgentStatus_MalformedNoFieldLine(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Agent State\n\n- iterations: 0\n- artifact_path: foo\n\n## Agent Log\n")
	got, err := readAgentStatus(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestReadAgentStatus_IgnoresInlineMentionOutsideHeading(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Goal\n\nSee ## Agent State informally.\n\n## Agent State\n\n- stage: ready\n\n## Agent Log\n")
	got, err := readAgentStatus(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ready" {
		t.Errorf("got %q, want %q", got, "ready")
	}
}

func TestReadAgentStatus_StopsAtNextHeading(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Agent State\n\n- iterations: 1\n\n## Agent Log\n\n- stage: this-should-not-match\n")
	got, err := readAgentStatus(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestReadAgentStatus_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := readAgentStatus(dir, "does-not-exist.md")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadAgentStatus_PrefersFirstMatchWhenDuplicateFields(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Agent State\n\n- stage: queued\n- stage: blocked\n\n## Agent Log\n")
	got, err := readAgentStatus(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "queued" {
		t.Errorf("got %q, want %q (first match)", got, "queued")
	}
}

// --- readAgentKind ---------------------------------------------------------

func writeTicketWithFrontmatter(t *testing.T, frontmatter, body string) (vaultDir, relPath string) {
	t.Helper()
	content := "---\n" + frontmatter + "\n---\n\n" + body
	return writeTempTicket(t, content)
}

func TestReadAgentKind_AgentKindTag(t *testing.T) {
	fm := "title: Test\nstatus: open\ntags:\n  - agent-queue\n  - agent-kind:coding\n"
	dir, rel := writeTicketWithFrontmatter(t, fm, "## Goal\n\nDo it.\n")
	got, err := readAgentKind(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "coding" {
		t.Errorf("got %q, want %q", got, "coding")
	}
}

func TestReadAgentKind_DirectAgentTag(t *testing.T) {
	fm := "title: Test\nstatus: open\ntags:\n  - agent-queue\n  - agent:tess\n"
	dir, rel := writeTicketWithFrontmatter(t, fm, "## Goal\n\nDo it.\n")
	got, err := readAgentKind(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "tess" {
		t.Errorf("got %q, want %q", got, "tess")
	}
}

func TestReadAgentKind_DirectAgentTakesPrecedence(t *testing.T) {
	// "agent:" tag should win over "agent-kind:" when both are present.
	fm := "title: Test\nstatus: open\ntags:\n  - agent:tess\n  - agent-kind:coding\n"
	dir, rel := writeTicketWithFrontmatter(t, fm, "")
	got, err := readAgentKind(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "tess" {
		t.Errorf("got %q, want %q (agent: tag should win)", got, "tess")
	}
}

func TestReadAgentKind_NoMatchingTag(t *testing.T) {
	fm := "title: Test\nstatus: open\ntags:\n  - agent-queue\n  - task\n"
	dir, rel := writeTicketWithFrontmatter(t, fm, "")
	got, err := readAgentKind(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string for no matching tag", got)
	}
}

func TestReadAgentKind_NoFrontmatter(t *testing.T) {
	dir, rel := writeTempTicket(t, "## Goal\n\nNo frontmatter here.\n")
	got, err := readAgentKind(dir, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string for missing frontmatter", got)
	}
}
