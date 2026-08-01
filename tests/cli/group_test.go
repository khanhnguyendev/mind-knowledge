package cli_test

import "testing"

// TestGroupUnknownSubcommand covers every group command (project, epic,
// story, source, wiki, link, tag, log). Each group must behave like Root:
// an unrecognized subcommand is an invalid-input error (exit 2, and under
// --json a parseable {"error":{"code":2,...}} envelope on stdout), while
// the bare group with no arguments at all still prints help and exits 0.
//
// Before the fix, every group left cobra's Args nil and had no RunE of its
// own, so cobra printed help and exited 0 for *any* input to the group,
// including a typo'd subcommand — a silent no-op that looks like success
// to a caller branching on exit code.
func TestGroupUnknownSubcommand(t *testing.T) {
	groups := []string{
		"project",
		"epic",
		"story",
		"source",
		"wiki",
		"link",
		"tag",
		"log",
	}

	for _, group := range groups {
		t.Run(group, func(t *testing.T) {
			t.Run("unknown subcommand exits 2", func(t *testing.T) {
				r := mk(t, newDB(t), group, "nonesuch")
				requireCode(t, r, 2)
			})

			t.Run("bare group prints help and exits 0", func(t *testing.T) {
				r := mk(t, newDB(t), group)
				requireCode(t, r, 0)
			})

			t.Run("--json unknown subcommand yields error envelope", func(t *testing.T) {
				r := mk(t, newDB(t), "--json", group, "nonesuch")
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
			})
		})
	}
}
