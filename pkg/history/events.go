package history

import (
	"strings"
	"time"

	"github.com/indihood/yore/pkg/store"
)

// EventQuery configures how recorded events are filtered, aggregated and ranked
// into a list of [Entry] by [RankEvents].
type EventQuery struct {
	// Sort selects the output ordering (default frecency).
	Sort SortMode
	// HalfLife is the frecency decay half-life; <= 0 uses DefaultHalfLife.
	HalfLife time.Duration
	// Dir, when non-empty, keeps only events that ran in this directory or a
	// subdirectory of it. This is how project-scoped recall is implemented.
	Dir string
	// MinCount drops commands seen fewer than MinCount times.
	MinCount int
	// SuccessOnly drops commands that have never exited successfully.
	SuccessOnly bool
	// Redact drops commands that look like they contain secrets.
	Redact bool
	// Limit caps the number of returned entries (0 = no limit).
	Limit int
	// Now overrides the reference time for frecency decay (tests); zero = now.
	Now time.Time
}

// RankEvents aggregates recorded events into deduplicated, ranked entries.
//
// It groups identical commands, summing occurrences and tracking the most
// recent timestamp, the directory of the most recent run, and the success and
// failure counts. The result is filtered and ranked according to q.
func RankEvents(events []store.Event, q EventQuery) []Entry {
	if q.HalfLife <= 0 {
		q.HalfLife = DefaultHalfLife
	}
	now := q.Now
	if now.IsZero() {
		now = time.Now()
	}

	index := make(map[string]int, len(events))
	entries := make([]Entry, 0, len(events))

	for _, ev := range events {
		cmd := strings.TrimSpace(ev.Command)
		if !keepCommand(cmd, q.Redact) {
			continue
		}
		if q.Dir != "" && !dirMatches(ev.Dir, q.Dir) {
			continue
		}

		i, ok := index[cmd]
		if !ok {
			index[cmd] = len(entries)
			entries = append(entries, Entry{Command: cmd, Shell: shellFromName(ev.Shell)})
			i = len(entries) - 1
		}
		e := &entries[i]
		e.Count++
		switch {
		case ev.Exit == 0:
			e.Successes++
		case ev.Exit != store.ExitUnknown:
			e.Failures++
		}
		if ev.Time.After(e.LastUsed) {
			e.LastUsed = ev.Time
			e.Shell = shellFromName(ev.Shell)
			if ev.Dir != "" {
				e.Dir = ev.Dir
			}
		}
	}

	out := entries[:0]
	for _, e := range entries {
		if q.MinCount > 1 && e.Count < q.MinCount {
			continue
		}
		if q.SuccessOnly && !e.EverSucceeded() {
			continue
		}
		out = append(out, e)
	}

	Rank(out, q.Sort, q.HalfLife, now)
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out
}

// dirMatches reports whether eventDir is root or a subdirectory of root.
func dirMatches(eventDir, root string) bool {
	if eventDir == "" {
		return false
	}
	if eventDir == root {
		return true
	}
	return strings.HasPrefix(eventDir, ensureTrailingSlash(root))
}

func ensureTrailingSlash(p string) string {
	if strings.HasSuffix(p, "/") {
		return p
	}
	return p + "/"
}

// shellFromName maps a recorded shell name to a Shell value.
func shellFromName(name string) Shell {
	switch name {
	case "zsh":
		return ShellZsh
	case "bash":
		return ShellBash
	default:
		return ShellUnknown
	}
}
