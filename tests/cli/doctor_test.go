package cli_test

import (
	"strings"
	"testing"
)

func TestDoctorReportsPlanlessDoneStory(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)
	requireCode(t, mk(t, db, "story", "mv", sid, "--to", "done"), 0)

	r := mk(t, db, "--json", "doctor", "--scope", "stories")
	requireCode(t, r, 0)

	var findings []struct {
		Check string `json:"check"`
		ID    string `json:"id"`
	}
	decode(t, r, &findings)

	found := false
	for _, f := range findings {
		if f.Check == "story.planless" && f.ID == sid {
			found = true
		}
	}
	if !found {
		t.Errorf("findings = %+v, want story.planless for %s", findings, sid)
	}
}

func TestDoctorReportsUnprocessedSource(t *testing.T) {
	db := newDB(t)

	sid := strings.TrimSpace(
		mk(t, db, "source", "add", "--title", "An article", "--body", "text").stdout)

	r := mk(t, db, "--json", "doctor", "--scope", "wiki")
	requireCode(t, r, 0)

	var findings []struct {
		Check string `json:"check"`
		ID    string `json:"id"`
	}
	decode(t, r, &findings)

	found := false
	for _, f := range findings {
		if f.Check == "wiki.unprocessed" && f.ID == sid {
			found = true
		}
	}
	if !found {
		t.Errorf("findings = %+v, want wiki.unprocessed for %s", findings, sid)
	}
}

func TestDoctorCleanDatabaseIsEmptyArray(t *testing.T) {
	r := mk(t, newDB(t), "--json", "doctor")
	requireCode(t, r, 0)

	if strings.TrimSpace(r.stdout) != "[]" {
		t.Errorf("stdout = %q, want []", r.stdout)
	}
}

func TestDoctorUnknownScopeExitsTwo(t *testing.T) {
	requireCode(t, mk(t, newDB(t), "doctor", "--scope", "nonsense"), 2)
}

func TestDoctorPlainOutputGroupsByCheck(t *testing.T) {
	db := newDB(t)

	mk(t, db, "source", "add", "--title", "An article", "--body", "text")

	r := mk(t, db, "doctor", "--scope", "wiki")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "wiki.unprocessed") {
		t.Errorf("plain output = %q, want the check name", r.stdout)
	}
}

func TestDoctorExitsZeroEvenWithFindings(t *testing.T) {
	db := newDB(t)

	mk(t, db, "source", "add", "--title", "An article", "--body", "text")

	// Findings are information, not failure. A skill decides what to do.
	requireCode(t, mk(t, db, "doctor"), 0)
}

// TestDoctorRejectsStrayArgument guards against the same silent-success
// shape as the group-command unknown-subcommand bug: doctor takes no
// positional arguments, and leaving cobra's Args unset let it silently
// accept and ignore one instead of erroring.
func TestDoctorRejectsStrayArgument(t *testing.T) {
	r := mk(t, newDB(t), "doctor", "bogus")
	requireCode(t, r, 2)
}
