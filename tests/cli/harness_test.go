// Package cli_test exercises the mk binary end to end. These tests are the
// contract that the /mk-* skills depend on, so their assertions describe
// observable output rather than internal behaviour.
package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binPath string

// TestMain builds the mk binary fresh for this test run and drives it via
// exec.Command in every test below. Go's test cache keys a package's
// cached result on that package's own source files (and whatever files it
// reads during the run) — it has no way to know this package's real
// subject is a binary built at runtime from internal/..., cmd/mk. So a
// change to production code under those directories does NOT invalidate a
// cached PASS here: `go test ./tests/cli/` can report success without
// having run against the new code at all.
//
// `make test` accounts for this by always passing -count=1 for this
// package (see the Makefile). If you run this package directly — e.g.
// `go test ./tests/cli/...` — pass -count=1 yourself, or you may be
// looking at a stale result.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mk-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binPath = filepath.Join(dir, "mk")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/mk")
	build.Dir = "../.."
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("building mk: " + err.Error())
	}

	os.Exit(m.Run())
}

type result struct {
	stdout string
	stderr string
	code   int
}

// mk runs the binary against a database private to this test.
func mk(t *testing.T, db string, args ...string) result {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "MK_DB="+db)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running mk %v: %v", args, err)
	}

	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// newDB returns a database path unique to this test.
func newDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "mk.db")
}

// decode parses r.stdout as JSON into v, failing the test on malformed
// output.
func decode(t *testing.T, r result, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(r.stdout), v); err != nil {
		t.Fatalf("stdout is not valid JSON (%v):\n%s", err, r.stdout)
	}
}

// requireCode asserts the process exit code.
func requireCode(t *testing.T, r result, want int) {
	t.Helper()
	if r.code != want {
		t.Fatalf("exit code = %d, want %d\nstdout: %s\nstderr: %s",
			r.code, want, r.stdout, r.stderr)
	}
}
