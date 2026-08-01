package cli_test

import (
	"strings"
	"testing"
)

func TestNoArgsPrintsHelp(t *testing.T) {
	r := mk(t, newDB(t))
	requireCode(t, r, 0)
	if !strings.Contains(r.stdout, "mind-knowledge") {
		t.Errorf("help output missing description:\n%s", r.stdout)
	}
}

func TestUnknownCommandExitsTwo(t *testing.T) {
	r := mk(t, newDB(t), "nonesuch")
	requireCode(t, r, 2)
}

func TestUnknownCommandJSONErrorEnvelope(t *testing.T) {
	r := mk(t, newDB(t), "--json", "nonesuch")
	requireCode(t, r, 2)

	var env struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decode(t, r, &env)
	if env.Error.Code != 2 {
		t.Errorf("error.code = %d, want 2", env.Error.Code)
	}
	if env.Error.Message == "" {
		t.Error("error.message is empty")
	}
}

// TestProjectShorthandWorksOnCommandsWithLocalProjectFlag guards against a
// name collision: a subcommand that registers its own "--project" flag
// (same long name as the persistent --project/-p) causes cobra to skip
// merging the persistent flag entirely for that subcommand — including its
// -p shorthand. epic create, epic edit, wiki add, wiki edit, and log add
// all used to fall into this trap; they must read the persistent flag
// instead of declaring their own.
//
// epic edit and wiki edit no longer *act* on -p — reassignment moved to
// --set-project — but they must still accept it, since -p is a global
// flag and a skill threading it through every call must not get an error.
func TestProjectShorthandWorksOnCommandsWithLocalProjectFlag(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	createRes := mk(t, db, "epic", "create", "-p", pid, "--title", "Auth overhaul")
	requireCode(t, createRes, 0)
	eid := strings.TrimSpace(createRes.stdout)

	requireCode(t, mk(t, db, "epic", "edit", eid, "-p", pid), 0)

	requireCode(t, mk(t, db, "wiki", "add",
		"-p", pid, "--title", "Scoped Page", "--body", "b"), 0)

	requireCode(t, mk(t, db, "wiki", "edit", "scoped-page", "-p", pid), 0)

	requireCode(t, mk(t, db, "log", "add",
		"-p", pid, "--kind", "ingest", "--summary", "shorthand works"), 0)
}
