package cli

import (
	_ "embed"
	"fmt"
	"io"
)

// Shell completion scripts, embedded at build time so the single binary can
// emit them via `yore completion <shell>`. Homebrew installs them with
// generate_completions_from_executable; users can also source them directly.
var (
	//go:embed completions/yore.bash
	bashCompletion string
	//go:embed completions/yore.zsh
	zshCompletion string
	//go:embed completions/yore.fish
	fishCompletion string
)

const completionUsage = `Usage: yore completion <bash|zsh|fish>

Output a shell completion script. Examples:
  yore completion bash > /usr/local/etc/bash_completion.d/yore
  yore completion zsh  > "${fpath[1]}/_yore"
  yore completion fish > ~/.config/fish/completions/yore.fish
`

// runCompletion handles `yore completion <shell>`. args are the tokens after
// the "completion" subcommand.
func runCompletion(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, completionUsage)
		return 2
	}
	var script string
	switch args[0] {
	case "bash":
		script = bashCompletion
	case "zsh":
		script = zshCompletion
	case "fish":
		script = fishCompletion
	case "-h", "--help":
		fmt.Fprint(stdout, completionUsage)
		return 0
	default:
		fmt.Fprintf(stderr, "yore: unknown shell %q (want bash, zsh or fish)\n", args[0])
		return 2
	}
	fmt.Fprint(stdout, script)
	return 0
}
