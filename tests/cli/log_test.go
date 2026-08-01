package cli_test

import (
	"strings"
	"testing"
)

func TestLogAddAndList(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	requireCode(t, mk(t, db, "log", "add",
		"--kind", "brainstorm", "--project", pid,
		"--summary", "broke auth work into 5 stories"), 0)

	r := mk(t, db, "--json", "log", "ls")
	requireCode(t, r, 0)

	var entries []struct {
		Kind    string `json:"kind"`
		Summary string `json:"summary"`
	}
	decode(t, r, &entries)
	if len(entries) != 1 || entries[0].Kind != "brainstorm" {
		t.Errorf("entries = %+v", entries)
	}
}

func TestLogListNewestFirst(t *testing.T) {
	db := newDB(t)

	mk(t, db, "log", "add", "--kind", "ingest", "--summary", "first")
	mk(t, db, "log", "add", "--kind", "query", "--summary", "second")

	r := mk(t, db, "--json", "log", "ls")
	var entries []struct {
		Summary string `json:"summary"`
	}
	decode(t, r, &entries)

	if len(entries) != 2 || entries[0].Summary != "second" {
		t.Errorf("entries = %+v, want newest first", entries)
	}
}

func TestLogTailLimits(t *testing.T) {
	db := newDB(t)

	for _, s := range []string{"a", "b", "c"} {
		mk(t, db, "log", "add", "--kind", "ingest", "--summary", s)
	}

	r := mk(t, db, "--json", "log", "ls", "--tail", "2")
	requireCode(t, r, 0)

	var entries []struct {
		Summary string `json:"summary"`
	}
	decode(t, r, &entries)
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestLogFilterByKind(t *testing.T) {
	db := newDB(t)

	mk(t, db, "log", "add", "--kind", "ingest", "--summary", "a")
	mk(t, db, "log", "add", "--kind", "query", "--summary", "b")

	r := mk(t, db, "--json", "log", "ls", "--kind", "query")
	var entries []struct {
		Summary string `json:"summary"`
	}
	decode(t, r, &entries)

	if len(entries) != 1 || entries[0].Summary != "b" {
		t.Errorf("entries = %+v, want just b", entries)
	}
}

func TestLogAddRequiresSummary(t *testing.T) {
	db := newDB(t)

	requireCode(t, mk(t, db, "log", "add", "--kind", "ingest"), 2)
}

func TestLogPlainOutputIsGreppable(t *testing.T) {
	db := newDB(t)

	mk(t, db, "log", "add", "--kind", "ingest", "--summary", "captured an article")

	r := mk(t, db, "log", "ls")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "ingest") ||
		!strings.Contains(r.stdout, "captured an article") {
		t.Errorf("plain log output = %q", r.stdout)
	}
}

func TestLogEmptyListSerializesAsEmptyArray(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "log", "ls")
	requireCode(t, r, 0)

	if strings.TrimSpace(r.stdout) != "[]" {
		t.Errorf("empty list stdout = %q, want []", r.stdout)
	}
}

func TestLogAddAndListProjectSymmetry(t *testing.T) {
	db := newDB(t)

	// add with unknown project should exit 1
	addRes := mk(t, db, "log", "add", "--kind", "ingest", "--project", "nope99", "--summary", "test")
	requireCode(t, addRes, 1)

	// ls with unknown project should also exit 1
	lsRes := mk(t, db, "log", "ls", "--project", "nope99")
	requireCode(t, lsRes, 1)
}

func TestLogAddAllowsOptionalProjectAndRef(t *testing.T) {
	db := newDB(t)

	// Add without project and ref
	addRes := mk(t, db, "--json", "log", "add", "--kind", "brainstorm", "--summary", "project-agnostic idea")
	requireCode(t, addRes, 0)

	var entry struct {
		ProjectID string `json:"project_id"`
		Ref       string `json:"ref"`
	}
	decode(t, addRes, &entry)

	if entry.ProjectID != "" || entry.Ref != "" {
		t.Errorf("optional fields not empty: project_id=%q, ref=%q", entry.ProjectID, entry.Ref)
	}
}
