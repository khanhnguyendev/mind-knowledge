// Command mk is mind-knowledge's command-line interface.
package main

import (
	"os"

	"github.com/khanhnguyendev/mind-knowledge/internal/cli"
)

func main() {
	os.Exit(cli.ExitCode(cli.Execute()))
}
