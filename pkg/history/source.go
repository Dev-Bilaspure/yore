package history

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Source knows how to locate and parse one shell's history format. Adding
// support for a new shell (e.g. fish) is a matter of implementing this
// interface and registering it in sources.
type Source interface {
	// Shell reports which shell this source handles.
	Shell() Shell
	// DefaultPath resolves the history file for this shell, honouring the
	// relevant environment variables. The bool is false when no path can be
	// determined.
	DefaultPath() (string, bool)
	// Parse reads raw command occurrences from history file contents.
	Parse(data []byte) ([]Raw, error)
}

// sources is the registry of known shells, in a stable order.
var sources = []Source{
	zshSource{},
	bashSource{},
}

// sourceFor returns the [Source] for a given shell, or nil if unsupported.
func sourceFor(s Shell) Source {
	for _, src := range sources {
		if src.Shell() == s {
			return src
		}
	}
	return nil
}

// ShellFor guesses the shell responsible for a history file from its path.
// It is used when explicit files are supplied via [Options.Files]. Unknown
// paths default to [ShellZsh], whose parser also handles plain one-command
// -per-line files.
func ShellFor(path string) Shell {
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.Contains(base, "bash"):
		return ShellBash
	case strings.Contains(base, "zsh"):
		return ShellZsh
	default:
		return ShellZsh
	}
}

// detectShell inspects $SHELL to identify the user's interactive shell.
func detectShell() (Shell, bool) {
	sh := strings.ToLower(filepath.Base(os.Getenv("SHELL")))
	switch {
	case strings.Contains(sh, "zsh"):
		return ShellZsh, true
	case strings.Contains(sh, "bash"):
		return ShellBash, true
	default:
		return ShellUnknown, false
	}
}

// resolveShells determines which shells to read for the given options:
//   - explicit opts.Shells if provided;
//   - otherwise the detected $SHELL, if its history file exists;
//   - otherwise every registered shell whose history file exists.
func resolveShells(fsys FS, opts Options) []Shell {
	if len(opts.Shells) > 0 {
		return opts.Shells
	}
	if s, ok := detectShell(); ok {
		if src := sourceFor(s); src != nil {
			if p, ok := src.DefaultPath(); ok && fsys.Stat(p) == nil {
				return []Shell{s}
			}
		}
	}
	var found []Shell
	for _, src := range sources {
		if p, ok := src.DefaultPath(); ok && fsys.Stat(p) == nil {
			found = append(found, src.Shell())
		}
	}
	return found
}

// readAll discovers the relevant history files (or uses opts.Files), parses
// each and returns the concatenated raw occurrences.
func readAll(fsys FS, opts Options) ([]Raw, error) {
	type job struct {
		path     string
		shell    Shell
		explicit bool // user named this file; a read error is fatal
	}
	var jobs []job

	if len(opts.Files) > 0 {
		for _, f := range opts.Files {
			jobs = append(jobs, job{path: f, shell: ShellFor(f), explicit: true})
		}
	} else {
		shells := resolveShells(fsys, opts)
		if len(shells) == 0 {
			return nil, fmt.Errorf("no shell history found: set $SHELL, or pass an explicit file")
		}
		for _, s := range shells {
			src := sourceFor(s)
			if src == nil {
				return nil, fmt.Errorf("unsupported shell: %s", s)
			}
			p, ok := src.DefaultPath()
			if !ok {
				return nil, fmt.Errorf("could not locate %s history file (set $HISTFILE)", s)
			}
			jobs = append(jobs, job{path: p, shell: s})
		}
	}

	var raws []Raw
	for _, j := range jobs {
		data, err := readSource(fsys, opts, j.path)
		if err != nil {
			// An explicitly named file (or stdin) is the user's intent, so a
			// failure is fatal. An auto-discovered file that vanished or is
			// unreadable is skipped best-effort so one bad source does not
			// sink the whole run.
			if j.explicit {
				return nil, fmt.Errorf("reading %s: %w", j.path, err)
			}
			continue
		}
		src := sourceFor(j.shell)
		if src == nil {
			src = zshSource{}
		}
		parsed, err := src.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", j.path, err)
		}
		raws = append(raws, parsed...)
	}
	return raws, nil
}

// readSource reads an entire history source: stdin when path is "-", otherwise
// a file via the [FS] abstraction.
func readSource(fsys FS, opts Options, path string) ([]byte, error) {
	if path == "-" {
		if opts.Stdin == nil {
			return nil, fmt.Errorf("cannot read from stdin: no input stream available")
		}
		return io.ReadAll(opts.Stdin)
	}
	rc, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }() // read-only; close error is not actionable
	return io.ReadAll(rc)
}

// expandHome expands a leading ~ in a path to the user's home directory.
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}
