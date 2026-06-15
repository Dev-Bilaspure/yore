// Command yore prints a clean, project-aware, frecency-ranked view of the shell
// commands you actually reuse — and lets you save the keepers as reusable,
// team-shareable recipes.
//
// See https://github.com/Dev-Bilaspure/yore for documentation.
package main

import (
	"os"

	"github.com/Dev-Bilaspure/yore/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
