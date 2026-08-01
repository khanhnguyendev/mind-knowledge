package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSourceAddWithAssetDoesNotReadStdin pins the hang an agent harness
// trips on. A harness routinely hands its child an inherited pipe on
// stdin that nobody writes to and nobody closes. --asset means the content
// is on disk and no body is wanted, but the asset-only path used to fall
// through to io.ReadAll(os.Stdin) anyway — and os.ModeCharDevice does not
// catch an inherited pipe, so mk blocked forever with no output.
func TestSourceAddWithAssetDoesNotReadStdin(t *testing.T) {
	db := newDB(t)

	asset := filepath.Join(t.TempDir(), "diagram.png")
	if err := os.WriteFile(asset, []byte("not really a png"), 0o644); err != nil {
		t.Fatalf("writing the asset: %v", err)
	}

	cmd := exec.Command(binPath, "source", "add", "--title", "A diagram",
		"--kind", "asset", "--asset", asset)
	cmd.Env = append(os.Environ(), "MK_DB="+db)

	// An open pipe held by this process: exactly the inherited-stdin case.
	// It is never written to and never closed until the test is done.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	defer stdin.Close()

	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting mk: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("mk source add --asset failed: %v\n%s", err, out.String())
		}
		if strings.TrimSpace(out.String()) == "" {
			t.Errorf("source add printed nothing; want the new source id")
		}
	case <-time.After(15 * time.Second):
		cmd.Process.Kill()
		t.Fatal("mk source add --asset blocked on an inherited stdin pipe")
	}
}

func TestSourceAddStillReadsPipedStdin(t *testing.T) {
	db := newDB(t)

	cmd := exec.Command(binPath, "source", "add", "--title", "Piped")
	cmd.Env = append(os.Environ(), "MK_DB="+db)
	cmd.Stdin = strings.NewReader("body from stdin")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("mk source add with piped stdin: %v", err)
	}

	view := mk(t, db, "source", "view", strings.TrimSpace(string(out)))
	requireCode(t, view, 0)
	if !strings.Contains(view.stdout, "body from stdin") {
		t.Errorf("stored body = %q, want the piped text", view.stdout)
	}
}

// TestPlainFlagIsGone: --plain was declared, registered, and documented,
// but never read by anything. Rather than invent a meaning for it, it is
// removed — plain text is already what every command emits without --json.
func TestPlainFlagIsGone(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--plain", "project", "ls")
	requireCode(t, r, 2)

	help := mk(t, db, "--help")
	requireCode(t, help, 0)
	if strings.Contains(help.stdout, "--plain") {
		t.Errorf("--help still advertises --plain:\n%s", help.stdout)
	}
}

// --- log kinds are one namespace, not two ---

func TestLogKindIsCaseInsensitiveEndToEnd(t *testing.T) {
	db := newDB(t)

	requireCode(t, mk(t, db, "log", "add", "--kind", "Done", "--summary", "first"), 0)
	requireCode(t, mk(t, db, "log", "add", "--kind", "  done  ", "--summary", "second"), 0)

	r := mk(t, db, "--json", "log", "ls", "--kind", "done")
	requireCode(t, r, 0)

	var entries []struct {
		Kind    string `json:"kind"`
		Summary string `json:"summary"`
	}
	decode(t, r, &entries)

	if len(entries) != 2 {
		t.Fatalf("log ls --kind done returned %d entries, want both: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if e.Kind != "done" {
			t.Errorf("stored kind = %q, want the normalized %q", e.Kind, "done")
		}
	}
}

func TestLogAddKindHelpDoesNotPromiseAnEnum(t *testing.T) {
	r := mk(t, newDB(t), "log", "add", "--help")
	requireCode(t, r, 0)

	// The listed kinds are a suggested vocabulary; nothing rejects other
	// values, and the help must not imply otherwise.
	if !strings.Contains(r.stdout, "free text") {
		t.Errorf("--kind help does not say the value is free text:\n%s", r.stdout)
	}
}

// --- negative row counts are bad input, not "unlimited" ---

func TestNegativeLimitIsRejected(t *testing.T) {
	db := newDB(t)
	seedProject(t, db)

	requireCode(t, mk(t, db, "project", "ls", "--limit", "-1"), 2)
	requireCode(t, mk(t, db, "story", "ls", "--limit", "-5"), 2)
}

func TestNegativeTailIsRejected(t *testing.T) {
	db := newDB(t)
	requireCode(t, mk(t, db, "log", "add", "--kind", "note", "--summary", "s"), 0)

	requireCode(t, mk(t, db, "log", "ls", "--tail", "-1"), 2)
}

func TestZeroLimitStillMeansUnlimited(t *testing.T) {
	db := newDB(t)
	seedProject(t, db)

	requireCode(t, mk(t, db, "project", "ls", "--limit", "0"), 0)
}

// --- plain `story view` is a parseable surface ---

// TestStoryViewSectionOrderIsStable guards against ranging over a map,
// which randomizes section order between runs and produces skills that
// pass locally and fail in CI.
func TestStoryViewSectionOrderIsStable(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	requireCode(t, mk(t, db, "story", "edit", sid,
		"-d", "the description",
		"--acceptance", "the acceptance",
		"--plan", "the plan",
		"--notes", "the notes"), 0)

	want := []string{"## description", "## acceptance", "## plan", "## notes"}

	first := mk(t, db, "story", "view", sid)
	requireCode(t, first, 0)

	at := -1
	for _, section := range want {
		i := strings.Index(first.stdout, section)
		if i < 0 {
			t.Fatalf("story view is missing %q:\n%s", section, first.stdout)
		}
		if i < at {
			t.Fatalf("sections are out of order (%q came too early):\n%s", section, first.stdout)
		}
		at = i
	}

	// Run it repeatedly: a map range only *sometimes* produces the order
	// a single run happens to observe.
	for i := 0; i < 20; i++ {
		again := mk(t, db, "story", "view", sid)
		requireCode(t, again, 0)
		if again.stdout != first.stdout {
			t.Fatalf("story view output changed between runs:\n%s\n---\n%s",
				first.stdout, again.stdout)
		}
	}
}

// TestConcurrentFirstRunAllSucceed is the cross-process form of
// store.TestConcurrentOpenOfAFreshDatabaseAllSucceed: several mk processes
// racing to create one brand-new database, which is exactly what a
// parallel-subagent harness does on its first call.
func TestConcurrentFirstRunAllSucceed(t *testing.T) {
	db := filepath.Join(t.TempDir(), "fresh.db")

	const invocations = 8
	type outcome struct {
		out  string
		code int
	}
	results := make([]outcome, invocations)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < invocations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			cmd := exec.Command(binPath, "--json", "project", "ls")
			cmd.Env = append(os.Environ(), "MK_DB="+db)
			out, err := cmd.CombinedOutput()

			code := 0
			if exitErr, ok := err.(*exec.ExitError); ok {
				code = exitErr.ExitCode()
			}
			results[i] = outcome{out: string(out), code: code}
		}(i)
	}
	close(start)
	wg.Wait()

	for i, r := range results {
		if r.code != 0 {
			t.Errorf("concurrent first-run invocation %d exited %d: %s", i, r.code, r.out)
			continue
		}
		if strings.TrimSpace(r.out) != "[]" {
			t.Errorf("concurrent first-run invocation %d printed %q, want []", i, r.out)
		}
	}
}
