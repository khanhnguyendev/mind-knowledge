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
