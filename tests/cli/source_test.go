package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceAddFromBodyFlag(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "source", "add",
		"--title", "An article", "--kind", "article",
		"--uri", "https://example.com/a", "--body", "hello")
	requireCode(t, r, 0)

	var src struct {
		Title       string `json:"title"`
		Kind        string `json:"kind"`
		URI         string `json:"uri"`
		Body        string `json:"body"`
		ContentHash string `json:"content_hash"`
	}
	decode(t, r, &src)

	if src.Title != "An article" || src.Kind != "article" {
		t.Errorf("source = %+v", src)
	}
	if src.URI != "https://example.com/a" {
		t.Errorf("uri = %q", src.URI)
	}
	if src.Body != "hello" {
		t.Errorf("body = %q", src.Body)
	}
	if len(src.ContentHash) != 64 {
		t.Errorf("content_hash = %q, want 64 hex characters", src.ContentHash)
	}
}

func TestSourceAddFromFile(t *testing.T) {
	db := newDB(t)

	path := filepath.Join(t.TempDir(), "clip.md")
	if err := os.WriteFile(path, []byte("# Clipped\n\nBody text."), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := mk(t, db, "--json", "source", "add", "--title", "Clipped", "--file", path)
	requireCode(t, r, 0)

	var src struct {
		Body string `json:"body"`
	}
	decode(t, r, &src)
	if !strings.Contains(src.Body, "Body text.") {
		t.Errorf("body = %q, want file contents", src.Body)
	}
}

// TestSourceAddBodyWinsOverFile confirms --body takes precedence when both
// --body and --file are given, per the documented precedence order.
func TestSourceAddBodyWinsOverFile(t *testing.T) {
	db := newDB(t)

	path := filepath.Join(t.TempDir(), "clip.md")
	if err := os.WriteFile(path, []byte("from file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := mk(t, db, "--json", "source", "add",
		"--title", "Both", "--body", "from flag", "--file", path)
	requireCode(t, r, 0)

	var src struct {
		Body string `json:"body"`
	}
	decode(t, r, &src)
	if src.Body != "from flag" {
		t.Errorf("body = %q, want the --body value to win over --file", src.Body)
	}
}

// TestSourceAddFileWinsOverStdin confirms --file takes precedence over
// piped stdin when both are present.
func TestSourceAddFileWinsOverStdin(t *testing.T) {
	db := newDB(t)

	path := filepath.Join(t.TempDir(), "clip.md")
	if err := os.WriteFile(path, []byte("from file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := exec.Command(binPath, "--json", "source", "add", "--title", "Both", "--file", path)
	cmd.Env = append(os.Environ(), "MK_DB="+db)
	cmd.Stdin = strings.NewReader("from stdin")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running mk: %v", err)
	}
	if !strings.Contains(string(out), "from file") || strings.Contains(string(out), "from stdin") {
		t.Errorf("output = %s, want --file to win over piped stdin", out)
	}
}

func TestSourceAddFromStdin(t *testing.T) {
	db := newDB(t)

	cmd := exec.Command(binPath, "--json", "source", "add", "--title", "Piped")
	cmd.Env = append(os.Environ(), "MK_DB="+db)
	cmd.Stdin = strings.NewReader("piped body")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running mk: %v", err)
	}
	if !strings.Contains(string(out), "piped body") {
		t.Errorf("output = %s, want the piped body stored", out)
	}
}

func TestSourceAddMissingBodyExitsTwo(t *testing.T) {
	db := newDB(t)

	// No --body, no --file, and stdin is empty.
	r := mk(t, db, "source", "add", "--title", "Nothing")
	requireCode(t, r, 2)
}

func TestSourceAddUnknownKindExitsTwo(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "source", "add", "--title", "X", "--kind", "video", "--body", "y")
	requireCode(t, r, 2)
}

func TestSourceAddWarnsOnDuplicateHash(t *testing.T) {
	db := newDB(t)

	first := mk(t, db, "--json", "source", "add", "--title", "One", "--body", "identical")
	requireCode(t, first, 0)

	// A duplicate is reported, not silently stored again.
	second := mk(t, db, "source", "add", "--title", "Two", "--body", "identical")
	requireCode(t, second, 2)
	if !strings.Contains(second.stdout, "already") {
		t.Errorf("duplicate message unhelpful: %q", second.stdout)
	}
}

func TestSourceAddForceAllowsDuplicate(t *testing.T) {
	db := newDB(t)

	mk(t, db, "source", "add", "--title", "One", "--body", "identical")
	r := mk(t, db, "source", "add", "--title", "Two", "--body", "identical", "--force")
	requireCode(t, r, 0)
}

// TestSourceListEmptyIsNotNull asserts on raw stdout text: json.Unmarshal
// treats "null" and "[]" identically, so a test that decodes into a slice
// cannot tell an empty list apart from a bug that emits null.
// TestSourceAddTwoDifferentAssetsNotFlaggedAsDuplicate guards the
// body != "" half of the duplicate-check condition in cmd/source.go. Every
// asset-only source has an empty body, so they all hash to sha256(""); the
// only thing stopping the second one from being rejected as a bogus
// duplicate is that the pre-check is skipped whenever body is empty.
func TestSourceAddTwoDifferentAssetsNotFlaggedAsDuplicate(t *testing.T) {
	db := newDB(t)

	pathA := filepath.Join(t.TempDir(), "a.png")
	pathB := filepath.Join(t.TempDir(), "b.png")

	first := mk(t, db, "source", "add", "--title", "A", "--asset", pathA)
	requireCode(t, first, 0)
	second := mk(t, db, "source", "add", "--title", "B", "--asset", pathB)
	requireCode(t, second, 0)

	idA := strings.TrimSpace(first.stdout)
	idB := strings.TrimSpace(second.stdout)
	if idA == "" || idB == "" || idA == idB {
		t.Errorf("ids = %q, %q, want two distinct non-empty ids", idA, idB)
	}
}

func TestSourceListEmptyIsNotNull(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "source", "ls")
	requireCode(t, r, 0)
	if got := strings.TrimSpace(r.stdout); got != "[]" {
		t.Errorf("stdout = %q, want the literal text []", got)
	}
}

func TestSourceListAndView(t *testing.T) {
	db := newDB(t)

	add := mk(t, db, "source", "add", "--title", "One", "--body", "a")
	id := strings.TrimSpace(add.stdout)

	list := mk(t, db, "--json", "source", "ls")
	requireCode(t, list, 0)

	var sources []struct {
		ID string `json:"id"`
	}
	decode(t, list, &sources)
	if len(sources) != 1 || sources[0].ID != id {
		t.Errorf("sources = %+v", sources)
	}

	view := mk(t, db, "source", "view", id)
	requireCode(t, view, 0)
	if !strings.Contains(view.stdout, "One") {
		t.Errorf("view = %q", view.stdout)
	}
}
