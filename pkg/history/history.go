// Package history reads, cleans, deduplicates and ranks shell command history.
//
// It is the reusable core behind the yore command-line tool. The typical entry
// point is [Collect], which discovers the appropriate history files, parses
// them, strips noise, deduplicates identical commands and ranks the result by
// frecency (a blend of frequency and recency).
//
// The package has no third-party dependencies and is safe for concurrent use
// in the sense that each call to Collect operates on its own state.
package history

import (
	"fmt"
	"io"
	"os"
	"time"
)

// DefaultHalfLife is the default frecency decay half-life: a command used
// exactly one half-life ago contributes half of its frequency weight to its
// score. Thirty days is a sensible default for interactive shell use.
const DefaultHalfLife = 30 * 24 * time.Hour

// Shell identifies the shell a command originated from.
type Shell int

const (
	// ShellUnknown is the zero value, used when the origin shell is unknown.
	ShellUnknown Shell = iota
	// ShellZsh is the Z shell.
	ShellZsh
	// ShellBash is the Bourne-again shell.
	ShellBash
)

// String returns the lowercase name of the shell ("zsh", "bash" or "unknown").
func (s Shell) String() string {
	switch s {
	case ShellZsh:
		return "zsh"
	case ShellBash:
		return "bash"
	default:
		return "unknown"
	}
}

// Entry is a single deduplicated command together with the usage statistics
// derived from history.
type Entry struct {
	// Command is the cleaned, directly usable command text.
	Command string
	// Count is the number of times the command appeared across all parsed
	// history (after deduplication).
	Count int
	// LastUsed is the most recent time the command was seen. It is the zero
	// time when the source provided no timestamps (e.g. plain bash history),
	// in which case ranking falls back to pure frequency.
	LastUsed time.Time
	// Shell is the shell the command was most recently seen in.
	Shell Shell
	// Score is the frecency score assigned during ranking. It is only
	// meaningful after [Collect] (or [Rank]) has run.
	Score float64
	// Dir is the directory the command was most recently run in, when known
	// (populated from recorded events, empty for plain history).
	Dir string
	// Successes and Failures count occurrences with a known exit status of
	// zero and non-zero respectively. Occurrences with an unknown exit status
	// are not counted in either. Populated only from recorded events.
	Successes int
	Failures  int
}

// EverSucceeded reports whether the command has at least one known successful
// run, or no exit information at all (treated as success).
func (e Entry) EverSucceeded() bool {
	return e.Successes > 0 || (e.Successes == 0 && e.Failures == 0)
}

// Raw is a single, unprocessed command occurrence produced by a [Source].
type Raw struct {
	// Command is the raw command text, possibly spanning multiple lines.
	Command string
	// Timestamp is when the command ran, or the zero time if the source did
	// not record one.
	Timestamp time.Time
	// Shell records which shell produced this occurrence.
	Shell Shell
}

// SortMode selects how [Collect] orders its output.
type SortMode int

const (
	// SortFrecency orders by frecency score, descending. This is the default.
	SortFrecency SortMode = iota
	// SortRecent orders by most recently used, descending.
	SortRecent
	// SortFrequent orders by raw usage count, descending.
	SortFrequent
	// SortRaw preserves the order of first appearance in the underlying
	// history (after deduplication), performing no ranking.
	SortRaw
)

// ParseSortMode converts a mode name to a [SortMode]. It accepts "frecency",
// "recent", "frequent" and "raw" (case-insensitive).
func ParseSortMode(s string) (SortMode, error) {
	switch s {
	case "frecency", "":
		return SortFrecency, nil
	case "recent":
		return SortRecent, nil
	case "frequent":
		return SortFrequent, nil
	case "raw":
		return SortRaw, nil
	default:
		return SortFrecency, fmt.Errorf("unknown sort mode %q (want frecency, recent, frequent or raw)", s)
	}
}

// Options configures a call to [Collect].
//
// The zero value is not the recommended default; use [DefaultOptions] and
// adjust from there so that Dedup and HalfLife carry sensible values.
type Options struct {
	// Shells restricts collection to the given shells. When empty, Collect
	// auto-detects the current shell from $SHELL and falls back to every shell
	// whose history file exists.
	Shells []Shell
	// Files lists explicit history files to read. When non-empty it overrides
	// shell discovery entirely; each file is parsed using ShellFor to pick a
	// parser, falling back to the zsh parser.
	Files []string
	// Sort selects the output ordering.
	Sort SortMode
	// HalfLife is the frecency decay half-life. Values <= 0 fall back to
	// DefaultHalfLife.
	HalfLife time.Duration
	// MinCount drops commands seen fewer than MinCount times. Zero or one keep
	// everything.
	MinCount int
	// Dedup controls deduplication. It is true in DefaultOptions; when false,
	// every occurrence is emitted in source order (implies SortRaw semantics
	// for grouping).
	Dedup bool
	// Limit caps the number of returned entries. Zero means no limit.
	Limit int
	// Redact enables best-effort redaction of commands that look like they
	// contain secrets (see filter.go). Such commands are dropped entirely so
	// they never reach stdout, a pager or fzf.
	Redact bool
	// Now overrides the reference time used for frecency decay. It is intended
	// for tests; when zero, time.Now is used.
	Now time.Time
	// FS overrides filesystem access. It is intended for tests; when nil, the
	// real filesystem is used.
	FS FS
	// Stdin supplies the reader used when a file path is "-". When nil, "-" is
	// rejected with an error. The CLI sets this to os.Stdin.
	Stdin io.Reader
}

// DefaultOptions returns the recommended defaults: deduplicate, rank by
// frecency with a 30-day half-life, auto-detect shells and emit everything.
func DefaultOptions() Options {
	return Options{
		Sort:     SortFrecency,
		HalfLife: DefaultHalfLife,
		Dedup:    true,
	}
}

// FS is the minimal filesystem interface the package needs. It exists so that
// tests can supply in-memory history without touching the real filesystem.
type FS interface {
	Open(name string) (io.ReadCloser, error)
	Stat(name string) error // returns nil if the file exists and is readable
}

// osFS is the production [FS] backed by the operating system.
type osFS struct{}

func (osFS) Open(name string) (io.ReadCloser, error) { return os.Open(name) }

func (osFS) Stat(name string) error {
	_, err := os.Stat(name)
	return err
}

// Collect runs the full pipeline: discover sources, parse, filter noise,
// deduplicate, rank and limit. It is the primary entry point of the package.
func Collect(opts Options) ([]Entry, error) {
	if opts.HalfLife <= 0 {
		opts.HalfLife = DefaultHalfLife
	}
	fsys := opts.FS
	if fsys == nil {
		fsys = osFS{}
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	raws, err := readAll(fsys, opts)
	if err != nil {
		return nil, err
	}

	raws = filterRaw(raws, opts.Redact)

	var entries []Entry
	if opts.Dedup {
		entries = dedup(raws)
	} else {
		entries = passthrough(raws)
	}

	if opts.MinCount > 1 {
		entries = applyMinCount(entries, opts.MinCount)
	}

	Rank(entries, opts.Sort, opts.HalfLife, now)

	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[:opts.Limit]
	}
	return entries, nil
}

// applyMinCount returns only the entries whose Count is at least minCount.
func applyMinCount(entries []Entry, minCount int) []Entry {
	out := entries[:0]
	for _, e := range entries {
		if e.Count >= minCount {
			out = append(out, e)
		}
	}
	return out
}
