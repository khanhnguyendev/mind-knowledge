package cli_test

import (
	"strings"
	"testing"
)

func TestVersionFlagPrintsAVersion(t *testing.T) {
	r := mk(t, newDB(t), "--version")
	requireCode(t, r, 0)

	if strings.TrimSpace(r.stdout) == "" {
		t.Error("--version printed nothing")
	}
	if !strings.Contains(r.stdout, "mk") {
		t.Errorf("--version = %q, want the binary name", r.stdout)
	}
}

func TestVersionDoesNotTouchTheDatabase(t *testing.T) {
	// Point at a path that could not be opened, to prove --version is
	// answered before any store work happens.
	r := mk(t, "/nonexistent-dir-99/mk.db", "--version")
	requireCode(t, r, 0)
}
