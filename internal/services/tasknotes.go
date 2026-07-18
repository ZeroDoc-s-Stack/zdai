// Pre-filters which agent-harness items are eligible for this dispatch cycle
// by reading the vault's TaskNotes/Tasks/ directory directly. No MCP server
// dependency — pure Go file parsing of YAML frontmatter.
package services

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// taskFrontmatter holds the subset of fields we need from a task note.
type taskFrontmatter struct {
	Status      string   `yaml:"status"`
	DateCreated string   `yaml:"dateCreated"`
	Tags        []string `yaml:"tags"`
}

// taskEntry is a task file with its parsed metadata.
type taskEntry struct {
	path string
	fm   taskFrontmatter
}

var (
	agentStateHeadingRe = regexp.MustCompile(`(?m)^## Agent State\s*$`)
	agentStatusRe       = regexp.MustCompile(`(?m)^-\s*(?:stage|status):\s*(\S+)\s*$`)
)

// parseFrontmatter extracts the YAML frontmatter from a note file.
func parseFrontmatter(data []byte) (taskFrontmatter, bool) {
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return taskFrontmatter{}, false
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return taskFrontmatter{}, false
	}
	var fm taskFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return taskFrontmatter{}, false
	}
	return fm, true
}

// hasTag reports whether the frontmatter tags list contains tag.
func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// scanTasks walks TaskNotes/Tasks/ and returns all task entries with the
// given tag whose status is in the candidate set (open/approved/ready/needs-rework).
func scanTasks(vaultDir, tag string) ([]taskEntry, error) {
	tasksDir := filepath.Join(vaultDir, "TaskNotes", "Tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, err
	}

	eligible := map[string]bool{
		"open": true, "approved": true, "ready": true, "needs-rework": true,
	}

	var out []taskEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tasksDir, e.Name()))
		if err != nil {
			continue
		}
		fm, ok := parseFrontmatter(data)
		if !ok {
			continue
		}
		if !eligible[fm.Status] {
			continue
		}
		if !hasTag(fm.Tags, tag) {
			continue
		}
		// Skip template/example notes with unexpanded placeholders.
		if strings.Contains(string(data), "{{") {
			continue
		}
		// Vault-relative path used as the identifier throughout zdai.
		rel := filepath.Join("TaskNotes", "Tasks", e.Name())
		out = append(out, taskEntry{path: rel, fm: fm})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].fm.DateCreated < out[j].fm.DateCreated
	})
	return out, nil
}

// readAgentStatus reads a ticket's `## Agent State` status line from disk.
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

// readAgentKind reads the agent dispatch hint from a ticket's frontmatter tags.
// "agent:<name>" (direct persona) takes precedence over "agent-kind:<kind>".
func readAgentKind(vaultDir, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(vaultDir, path))
	if err != nil {
		return "", err
	}
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return "", nil
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", nil
	}
	frontmatter := rest[:end]

	var directAgent, kindAgent string
	for _, line := range strings.Split(frontmatter, "\n") {
		trimmed := strings.TrimSpace(line)
		tag, ok := strings.CutPrefix(trimmed, "- ")
		if !ok {
			continue
		}
		tag = strings.TrimSpace(tag)
		if v, ok2 := strings.CutPrefix(tag, "agent:"); ok2 && directAgent == "" {
			directAgent = strings.TrimSpace(v)
		}
		if v, ok2 := strings.CutPrefix(tag, "agent-kind:"); ok2 && kindAgent == "" {
			kindAgent = strings.TrimSpace(v)
		}
	}
	if directAgent != "" {
		return directAgent, nil
	}
	return kindAgent, nil
}

// ticketEligible mirrors harness-coordinator eligibility rules.
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

// eligibleWork scans the vault directly for eligible agent-request tasks and
// agent-queue tickets. No MCP server required.
func eligibleWork(vaultDir string) (requests, tickets []string, err error) {
	requestCandidates, err := scanTasks(vaultDir, "agent-request")
	if err != nil {
		return nil, nil, err
	}
	for _, t := range requestCandidates {
		requests = append(requests, t.path)
		if len(requests) == eligibilityCap {
			break
		}
	}

	ticketCandidates, err := scanTasks(vaultDir, "agent-queue")
	if err != nil {
		return nil, nil, err
	}
	for _, t := range ticketCandidates {
		agentStatus, err := readAgentStatus(vaultDir, t.path)
		if err != nil {
			continue
		}
		if !ticketEligible(agentStatus, t.fm.Status) {
			continue
		}
		tickets = append(tickets, t.path)
		if len(tickets) == eligibilityCap {
			break
		}
	}
	return requests, tickets, nil
}
