package contract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const contractPath = "../../skills/references/mk-contract.md"

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mk-contract-*")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "mk")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/mk")
	build.Dir = "../.."
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("building mk: " + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func contract(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("reading contract: %v", err)
	}
	return string(b)
}

// help returns the help output for a command path, e.g. help(t, "story", "mv").
func help(t *testing.T, args ...string) string {
	t.Helper()
	out, _ := exec.Command(binPath, append(args, "--help")...).CombinedOutput()
	return string(out)
}

func TestContractNamesOnlyRealCommands(t *testing.T) {
	groups := []string{"project", "epic", "story", "source", "wiki", "link", "tag", "log"}
	text := contract(t)

	for _, g := range groups {
		if !strings.Contains(text, g) {
			t.Errorf("contract never mentions the %q command group", g)
		}
	}

	// Every "mk <group> <sub>" the contract names must exist.
	for _, g := range groups {
		h := help(t, g)
		for _, sub := range []string{"add", "create", "ls", "view", "edit", "mv", "rm", "index"} {
			claimed := strings.Contains(text, "mk "+g+" "+sub)
			real := strings.Contains(h, "\n  "+sub+" ")
			if claimed && !real {
				t.Errorf("contract claims `mk %s %s` but the binary has no such subcommand", g, sub)
			}
		}
	}
}

func TestContractStatusVerbsMatchBinary(t *testing.T) {
	// epic and story change status with `mv --to`; project and wiki with
	// `edit --status`. A contract that says otherwise sends every skill
	// down the wrong path.
	cases := []struct {
		args []string
		flag string
	}{
		{[]string{"epic", "mv"}, "--to"},
		{[]string{"story", "mv"}, "--to"},
		{[]string{"project", "edit"}, "--status"},
		{[]string{"wiki", "edit"}, "--status"},
	}
	for _, c := range cases {
		if h := help(t, c.args...); !strings.Contains(h, c.flag) {
			t.Errorf("`mk %s` has no %s flag", strings.Join(c.args, " "), c.flag)
		}
	}
	if h := help(t, "story", "edit"); strings.Contains(h, "--status") {
		t.Error("`mk story edit --status` now exists; the contract says it does not")
	}
}

func TestContractEnumValuesExist(t *testing.T) {
	text := contract(t)
	enums := map[string][]string{
		"story status":  {"backlog", "ready", "in-progress", "review", "done", "dropped"},
		"wiki kind":     {"summary", "concept", "entity", "decision", "spec", "synthesis", "comparison"},
		"link relation": {"derived-from", "supersedes", "references", "implements"},
		"entity kind":   {"project", "epic", "story", "source", "wiki"},
		"sync state":    {"ok", "missing", "not-git", "remote-changed", "check-failed"},
	}
	for name, values := range enums {
		for _, v := range values {
			if !strings.Contains(text, v) {
				t.Errorf("contract omits %s value %q", name, v)
			}
		}
	}
}

func TestContractListsEveryDoctorCheck(t *testing.T) {
	text := contract(t)
	checks := []string{
		"wiki.orphans", "wiki.stale", "wiki.uncited", "wiki.unprocessed",
		"wiki.dangling", "wiki.missing", "story.planless", "story.stranded",
		"epic.empty", "project.missing", "project.unverifiable",
	}
	for _, c := range checks {
		if !strings.Contains(text, c) {
			t.Errorf("contract omits doctor check %q", c)
		}
	}
}

func TestContractExitCodesMatchBinary(t *testing.T) {
	db := filepath.Join(t.TempDir(), "mk.db")
	run := func(args ...string) int {
		cmd := exec.Command(binPath, args...)
		cmd.Env = append(os.Environ(), "MK_DB="+db)
		cmd.Run()
		return cmd.ProcessState.ExitCode()
	}
	if got := run("project", "view", "nope99"); got != 1 {
		t.Errorf("missing entity exit = %d, want 1", got)
	}
	if got := run("epic", "create", "--project", "nope99", "--title", "x"); got != 1 {
		t.Errorf("missing parent exit = %d, want 1", got)
	}
	if got := run("doctor", "--scope", "bogus"); got != 2 {
		t.Errorf("bad input exit = %d, want 2", got)
	}
	if got := run("story", "bogus"); got != 2 {
		t.Errorf("unknown subcommand exit = %d, want 2", got)
	}
}

// TestContractProjectScopeBehaviors pins the four -p/--project behaviours
// the contract's "-p behaviour by command" table claims: required-and-
// validated (epic create), accepted-but-unvalidated (story create),
// scoping (wiki ls), and outright rejection (search). This table is the
// section the brief calls out as the trickiest, and it is also the one a
// review already caught drifting once (story create was undocumented) —
// so it gets live assertions rather than resting on the contract's prose.
func TestContractProjectScopeBehaviors(t *testing.T) {
	db := filepath.Join(t.TempDir(), "mk.db")
	env := append(os.Environ(), "MK_DB="+db)

	run := func(args ...string) (string, int) {
		t.Helper()
		cmd := exec.Command(binPath, args...)
		cmd.Env = env
		out, _ := cmd.CombinedOutput()
		return string(out), cmd.ProcessState.ExitCode()
	}

	// epic create requires -p/--project: omitting it is bad input, exit 2.
	if out, code := run("epic", "create", "--title", "no project"); code != 2 {
		t.Errorf("epic create without -p: exit = %d, want 2 (a project should be required)\noutput: %s", code, out)
	}

	pOut, code := run("project", "add", "--name", "scope-a", "--path", "/tmp/mk-contract-scope-a")
	if code != 0 {
		t.Fatalf("project add failed (exit %d): %s", code, pOut)
	}
	proj := strings.TrimSpace(pOut)

	eOut, code := run("epic", "create", "--project", proj, "--title", "E")
	if code != 0 {
		t.Fatalf("epic create failed (exit %d): %s", code, eOut)
	}
	epic := strings.TrimSpace(eOut)

	// story create does NOT validate -p at all: a story's project comes
	// from its epic, so a nonexistent project passed via -p is a pure
	// no-op, not an error. If mk ever starts validating -p on story
	// create, this must start failing (exit would become nonzero), which
	// is exactly the drift this test exists to catch.
	if out, code := run("story", "create", "--epic", epic, "--title", "S", "-p", "totally-bogus-project-xyz"); code != 0 {
		t.Errorf("story create -p <nonexistent project>: exit = %d, want 0 (story create does not validate -p)\noutput: %s", code, out)
	}

	// wiki ls -p scopes its result to that project: a page in a second
	// project must not appear.
	p2Out, code := run("project", "add", "--name", "scope-b", "--path", "/tmp/mk-contract-scope-b")
	if code != 0 {
		t.Fatalf("project add (2) failed (exit %d): %s", code, p2Out)
	}
	proj2 := strings.TrimSpace(p2Out)

	if out, code := run("wiki", "add", "-p", proj, "--title", "Scope Page A"); code != 0 {
		t.Fatalf("wiki add (1) failed (exit %d): %s", code, out)
	}
	if out, code := run("wiki", "add", "-p", proj2, "--title", "Scope Page B"); code != 0 {
		t.Fatalf("wiki add (2) failed (exit %d): %s", code, out)
	}
	lsOut, code := run("wiki", "ls", "-p", proj, "--json")
	if code != 0 {
		t.Fatalf("wiki ls -p failed (exit %d): %s", code, lsOut)
	}
	if !strings.Contains(lsOut, "Scope Page A") {
		t.Errorf("wiki ls -p %s did not include that project's own page: %s", proj, lsOut)
	}
	if strings.Contains(lsOut, "Scope Page B") {
		t.Errorf("wiki ls -p %s leaked the other project's page: %s", proj, lsOut)
	}

	// search rejects -p outright.
	if out, code := run("search", "anything", "-p", proj); code != 2 {
		t.Errorf("search -p: exit = %d, want 2 (search rejects -p)\noutput: %s", code, out)
	}
}
