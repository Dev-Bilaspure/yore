package cli

import (
	_ "embed"
	"fmt"
	"io"
)

// Shell integration scripts, embedded so `yore init <shell>` can print them for
// the user to eval/source. They install the hooks that record each command.
var (
	//go:embed shellinit/zsh.sh
	zshInit string
	//go:embed shellinit/bash.sh
	bashInit string
	//go:embed shellinit/fish.fish
	fishInit string
)

const initUsage = `Usage: yore init <bash|zsh|fish>

Print the shell integration that records your commands. Add to your shell rc:
  zsh:   eval "$(yore init zsh)"          # in ~/.zshrc
  bash:  eval "$(yore init bash)"         # in ~/.bashrc
  fish:  yore init fish | source          # in ~/.config/fish/config.fish
`

// runInit handles `yore init <shell>`.
func runInit(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, initUsage)
		return 2
	}
	switch args[0] {
	case "zsh":
		fmt.Fprint(stdout, zshInit)
	case "bash":
		fmt.Fprint(stdout, bashInit)
	case "fish":
		fmt.Fprint(stdout, fishInit)
	case "-h", "--help":
		fmt.Fprint(stdout, initUsage)
	default:
		fmt.Fprintf(stderr, "yore: unknown shell %q (want bash, zsh or fish)\n", args[0])
		return 2
	}
	return 0
}
