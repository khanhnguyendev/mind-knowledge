// Package render turns store results into the two output shapes mk
// supports: machine-readable JSON and aligned plain text.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// JSON writes v as a single compact JSON line.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

type errEnvelope struct {
	Error errBody `json:"error"`
}

type errBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorOut writes an error in whichever shape the caller asked for. Per
// the spec, JSON errors go to stdout so a skill parsing stdout sees them.
func ErrorOut(w io.Writer, jsonMode bool, code int, msg string) {
	if jsonMode {
		JSON(w, errEnvelope{Error: errBody{Code: code, Message: msg}})
		return
	}
	fmt.Fprintf(w, "error: %s\n", msg)
}

// Table writes headers and rows as space-aligned columns. Cells are not
// truncated; callers shorten values before calling.
func Table(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	writeRow := func(cells []string) {
		var b strings.Builder
		for i, cell := range cells {
			if i > 0 {
				b.WriteString("  ")
			}
			if i == len(cells)-1 {
				b.WriteString(cell)
			} else {
				b.WriteString(cell + strings.Repeat(" ", widths[i]-len(cell)))
			}
		}
		fmt.Fprintln(w, b.String())
	}

	writeRow(headers)
	for _, row := range rows {
		writeRow(row)
	}
}
