// Package cli implements the yore command-line interface: a thin layer over the
// history package that parses flags, runs the collection pipeline and writes
// clean, directly usable commands to stdout.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Dev-Bilaspure/yore/pkg/history"
)

const usageText = `yore - your shell history as project-aware, frecency-ranked memory.

Usage:
  yore [flags]            recall commands, best-first (deduplicated, ranked)
  yore here [flags]       recall commands used in the current project
  yore <subcommand> ...

yore ranks the commands you actually reuse by frecency (frequency + recency).
Install the recording hook (yore init) to also capture each command's directory
and exit status, unlocking project-aware recall ('yore here') and what-worked
filtering. Until then, yore reads your existing shell history.

Examples:
  yore                      # all commands, best-first
  yore -n 20                # top 20 by frecency
  yore here                 # commands that matter in THIS project
  yore here --ok            # ...only ones that exited successfully
  yore | fzf                # interactive fuzzy pick
  yore --count              # prefix each command with its usage count
  eval "$(yore init zsh)"   # start recording (add to ~/.zshrc)

Flags:
  -n, --top N            limit output to the top N commands (0 = all)
      --here             scope to the current project (same as 'yore here')
      --ok               only commands that have exited successfully
      --sort MODE        ordering: frecency|recent|frequent|raw (default frecency)
      --half-life DUR    frecency decay half-life, e.g. 720h (default 720h = 30d)
      --min-count N      only commands used at least N times
      --count            prefix each line with its usage count (TAB-separated)
      --redact           drop commands that look like they contain secrets
      --file PATH        read a specific history file instead of recorded memory
                         (repeatable; use - for stdin)
      --shell MODE       with --file: auto|zsh|bash|all (default auto)
      --no-dedup         with --file: do not deduplicate
  -0, --null             separate entries with NUL instead of newline
      --version          print version and exit
  -h, --help             show this help and exit

Subcommands:
  pick                   fuzzy-pick a command or recipe (bound to Ctrl-G by init)
  here                   recall commands used in the current project
  save <name> -- <cmd>   save a command as a reusable recipe
  run <name>             run a saved recipe (prompts for {parameters})
  recipes                list saved recipes
  init <shell>           print shell integration to record commands (eval it)
  import                 seed memory from existing shell history files
  record ...             append one event (used internally by the hook)
  completion <shell>     output a bash, zsh or fish completion script

Recipes are stored in ~/.config/yore/recipes (personal) and ./.yorefile
(per-project; commit it to share your team's runbook).
`

// stringSlice is a flag.Value that accumulates repeated string flags.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// Run parses args (excluding the program name) and executes the CLI, writing
// results to stdout and diagnostics to stderr. It returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	// Subcommands are dispatched before flag parsing so they can take their own
	// positional arguments.
	if len(args) > 0 {
		switch args[0] {
		case "completion":
			return runCompletion(args[1:], stdout, stderr)
		case "init":
			return runInit(args[1:], stdout, stderr)
		case "record":
			return runRecord(args[1:], stdout, stderr)
		case "import":
			return runImport(args[1:], stdout, stderr)
		case "save":
			return runSave(args[1:], stdout, stderr)
		case "run":
			return runRun(args[1:], stdout, stderr)
		case "recipes":
			return runRecipes(args[1:], stdout, stderr)
		case "pick":
			return runPick(args[1:], stdout, stderr)
		}
	}
	// `yore here` is an alias for `yore --here`.
	forceHere := false
	if len(args) > 0 && args[0] == "here" {
		forceHere = true
		args = args[1:]
	}

	fs := flag.NewFlagSet("yore", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usageText) }

	var (
		top         int
		shellMode   string
		files       stringSlice
		sortMode    string
		halfLife    = history.DefaultHalfLife
		minCount    int
		noDedup     bool
		redact      bool
		hereFlag    bool
		okOnly      bool
		showCount   bool
		nullSep     bool
		showVersion bool
	)

	fs.IntVar(&top, "top", 0, "")
	fs.IntVar(&top, "n", 0, "")
	fs.StringVar(&shellMode, "shell", "auto", "")
	fs.Var(&files, "file", "")
	fs.StringVar(&sortMode, "sort", "frecency", "")
	fs.DurationVar(&halfLife, "half-life", history.DefaultHalfLife, "")
	fs.IntVar(&minCount, "min-count", 0, "")
	fs.BoolVar(&noDedup, "no-dedup", false, "")
	fs.BoolVar(&redact, "redact", false, "")
	fs.BoolVar(&hereFlag, "here", false, "")
	fs.BoolVar(&okOnly, "ok", false, "")
	fs.BoolVar(&showCount, "count", false, "")
	fs.BoolVar(&nullSep, "null", false, "")
	fs.BoolVar(&nullSep, "0", false, "")
	fs.BoolVar(&showVersion, "version", false, "")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if showVersion {
		fmt.Fprintln(stdout, versionString())
		return 0
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "yore: unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return 2
	}

	sortM, err := history.ParseSortMode(sortMode)
	if err != nil {
		fmt.Fprintf(stderr, "yore: %v\n", err)
		return 2
	}

	var entries []history.Entry
	if len(files) > 0 {
		// Explicit file/stdin source: read it directly (the original behaviour).
		opts, err := buildOptions(top, shellMode, files, sortM, halfLife, minCount, noDedup, redact)
		if err != nil {
			fmt.Fprintf(stderr, "yore: %v\n", err)
			return 2
		}
		entries, err = history.Collect(opts)
		if err != nil {
			fmt.Fprintf(stderr, "yore: %v\n", err)
			return 1
		}
	} else {
		// Default: recall from recorded memory (plus live history).
		events, err := loadRecallEvents()
		if err != nil {
			fmt.Fprintf(stderr, "yore: %v\n", err)
			return 1
		}
		q := history.EventQuery{
			Sort:        sortM,
			HalfLife:    halfLife,
			MinCount:    minCount,
			SuccessOnly: okOnly,
			Redact:      redact,
			Limit:       top,
		}
		if hereFlag || forceHere {
			q.Dir = projectRoot(currentDir())
		}
		entries = history.RankEvents(events, q)
		if (hereFlag || forceHere) && len(entries) == 0 && hookActive() {
			fmt.Fprintln(stderr, "yore: no commands recorded in this project yet — run some and they'll appear here.")
		}
		// First interactive run (before recording is enabled): show a digestible
		// preview rather than dumping the entire history.
		if !hereFlag && !forceHere && top == 0 && onFirstInteractiveRun(stdout) && len(entries) > welcomePreview {
			entries = entries[:welcomePreview]
		}
	}

	if err := writeEntries(stdout, entries, showCount, nullSep); err != nil {
		// A closed downstream pipe (e.g. `yore | head`, or fzf exiting on
		// selection) is normal for a filter and must not be reported as an error.
		if isBrokenPipe(err) {
			return 0
		}
		fmt.Fprintf(stderr, "yore: %v\n", err)
		return 1
	}
	// Onboarding guidance (stderr, interactive-only) for the recall path.
	if len(files) == 0 {
		onboardingFooter(stdout, stderr, len(entries) > 0)
	}
	return 0
}

// isBrokenPipe reports whether err is an EPIPE-style "downstream closed the
// pipe" error, which a Unix filter should treat as a clean exit.
func isBrokenPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}

// buildOptions translates the --file-mode flags into history.Options.
func buildOptions(top int, shellMode string, files []string, sort history.SortMode, halfLife time.Duration, minCount int, noDedup, redact bool) (history.Options, error) {
	opts := history.DefaultOptions()
	opts.Limit = top
	opts.Files = files
	opts.HalfLife = halfLife
	opts.MinCount = minCount
	opts.Dedup = !noDedup
	opts.Redact = redact
	opts.Stdin = os.Stdin
	opts.Sort = sort

	if len(files) == 0 {
		shells, err := parseShellMode(shellMode)
		if err != nil {
			return opts, err
		}
		opts.Shells = shells
	}
	return opts, nil
}

// parseShellMode maps the --shell flag to a set of shells. "auto" returns nil,
// which lets the history package auto-detect.
func parseShellMode(mode string) ([]history.Shell, error) {
	switch strings.ToLower(mode) {
	case "auto", "":
		return nil, nil
	case "all":
		return []history.Shell{history.ShellZsh, history.ShellBash}, nil
	case "zsh":
		return []history.Shell{history.ShellZsh}, nil
	case "bash":
		return []history.Shell{history.ShellBash}, nil
	default:
		return nil, fmt.Errorf("unknown --shell %q (want auto, all, zsh or bash)", mode)
	}
}

// writeEntries prints entries to w. With showCount, each line is prefixed by
// the usage count and a TAB. With nullSep, entries are NUL-separated (and
// embedded newlines in multi-line commands are preserved verbatim); otherwise
// entries are newline-separated.
func writeEntries(w io.Writer, entries []history.Entry, showCount, nullSep bool) error {
	bw := bufio.NewWriter(w)
	sep := byte('\n')
	if nullSep {
		sep = 0
	}
	for _, e := range entries {
		if showCount {
			if _, err := bw.WriteString(strconv.Itoa(e.Count) + "\t"); err != nil {
				return err
			}
		}
		if _, err := bw.WriteString(e.Command); err != nil {
			return err
		}
		if err := bw.WriteByte(sep); err != nil {
			return err
		}
	}
	return bw.Flush()
}
