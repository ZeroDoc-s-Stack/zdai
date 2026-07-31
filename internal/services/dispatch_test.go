package services

import (
	"os"
	"path/filepath"
	"testing"
)

// helpers shared by readAgentKind tests

func writeTempTicket(t *testing.T, content string) (vaultDir, relPath string) {
	t.Helper()
	dir := t.TempDir()
	relPath = "ticket.md"
	if err := os.WriteFile(filepath.Join(dir, relPath), []byte(content), 0o644); err != nil {
		t.Fatalf("write temp ticket: %v", err)
	}
	return dir, relPath
}

func writeTicketWithFrontmatter(t *testing.T, frontmatter, body string) (vaultDir, relPath string) {
	t.Helper()
	content := "---\n" + frontmatter + "\n---\n\n" + body
	return writeTempTicket(t, content)
}

// --- readAgentKind -----------------------------------------------------------

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
