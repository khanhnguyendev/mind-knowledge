package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONIsCompactAndNewlineTerminated(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, map[string]string{"id": "a3f2k1"}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	got := buf.String()
	if got != "{\"id\":\"a3f2k1\"}\n" {
		t.Errorf("JSON output = %q, want compact object plus newline", got)
	}
}

func TestErrorOutJSONShape(t *testing.T) {
	var buf bytes.Buffer
	ErrorOut(&buf, true, 2, "unknown status \"todo\"")

	var got struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal %q: %v", buf.String(), err)
	}
	if got.Error.Code != 2 {
		t.Errorf("code = %d, want 2", got.Error.Code)
	}
	if got.Error.Message != `unknown status "todo"` {
		t.Errorf("message = %q, want the raw message", got.Error.Message)
	}
}

func TestErrorOutPlainShape(t *testing.T) {
	var buf bytes.Buffer
	ErrorOut(&buf, false, 1, "no such project")
	if got := buf.String(); got != "error: no such project\n" {
		t.Errorf("plain error = %q", got)
	}
}

func TestTableAlignsColumns(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"ID", "TITLE"}, [][]string{
		{"a3f2k1", "short"},
		{"b4g3l2", "a much longer title"},
	})
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header plus 2 rows, got %d lines: %q", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "ID") || !strings.Contains(lines[0], "TITLE") {
		t.Errorf("header line = %q", lines[0])
	}
	// Every row must start the second column at the same offset.
	off1 := strings.Index(lines[1], "short")
	off2 := strings.Index(lines[2], "a much longer title")
	if off1 != off2 {
		t.Errorf("second column misaligned: %d vs %d", off1, off2)
	}
}
