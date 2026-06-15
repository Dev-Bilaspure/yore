package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/Dev-Bilaspure/yore/pkg/history"
	"github.com/Dev-Bilaspure/yore/pkg/store"
)

// runImport handles `yore import`: seed the store from existing shell history
// files so past commands become permanent, searchable memory. Imported events
// carry no directory or exit information. Re-running is safe — events already
// present (matched by timestamp and command) are skipped.
func runImport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("yore import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		shellMode string
		files     stringSlice
	)
	fs.StringVar(&shellMode, "shell", "auto", "which history to import: auto|zsh|bash|all")
	fs.Var(&files, "file", "history file to import (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var shells []history.Shell
	if len(files) == 0 {
		s, err := parseShellMode(shellMode)
		if err != nil {
			fmt.Fprintf(stderr, "yore: %v\n", err)
			return 2
		}
		shells = s
	}

	incoming, err := histfileEvents(shells, files)
	if err != nil {
		fmt.Fprintf(stderr, "yore: import: %v\n", err)
		return 1
	}

	st, err := store.Default()
	if err != nil {
		fmt.Fprintf(stderr, "yore: %v\n", err)
		return 1
	}
	existing, err := st.Load()
	if err != nil {
		fmt.Fprintf(stderr, "yore: %v\n", err)
		return 1
	}

	seen := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		seen[eventKey(e)] = struct{}{}
	}

	imported := 0
	for _, e := range incoming {
		if _, ok := seen[eventKey(e)]; ok {
			continue
		}
		seen[eventKey(e)] = struct{}{}
		if err := st.Append(e); err != nil {
			fmt.Fprintf(stderr, "yore: import: %v\n", err)
			return 1
		}
		imported++
	}

	fmt.Fprintf(stdout, "yore: imported %d commands into %s\n", imported, st.Path())
	return 0
}

func eventKey(e store.Event) string {
	return strconv.FormatInt(e.Time.Unix(), 10) + "\x00" + e.Command
}
