package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Dev-Bilaspure/yore/pkg/history"
	"github.com/Dev-Bilaspure/yore/pkg/store"
)

// runRecord handles `yore record ...`, the target of the shell hook. It appends
// one event to the store and is designed to be fast and quiet: it is on the
// interactive hot path, so it never prints to stdout and treats a missing or
// trivial command as a no-op success.
func runRecord(args []string, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("yore record", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		command    string
		dir        string
		shell      string
		exit       int
		durationMS int
	)
	fs.StringVar(&command, "command", "", "the command that ran")
	fs.StringVar(&dir, "dir", "", "working directory")
	fs.StringVar(&shell, "shell", "", "shell name")
	fs.IntVar(&exit, "exit", store.ExitUnknown, "exit status")
	fs.IntVar(&durationMS, "duration-ms", 0, "duration in milliseconds")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return 0 // nothing to record
	}
	// Never persist commands that look like they contain secrets: keeping them
	// out of the on-disk store is more important than recording every command.
	if history.LooksLikeSecret(command) {
		return 0
	}

	st, err := store.Default()
	if err != nil {
		return 0 // never disrupt the shell over a recording failure
	}
	ev := store.Event{
		Time:     time.Now(),
		Command:  command,
		Dir:      dir,
		Exit:     exit,
		Duration: time.Duration(durationMS) * time.Millisecond,
		Shell:    shell,
	}
	if err := st.Append(ev); err != nil {
		fmt.Fprintf(stderr, "yore: record: %v\n", err)
		return 1
	}
	return 0
}
