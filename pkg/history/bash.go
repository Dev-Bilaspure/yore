package history

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// bashSource parses Bourne-again shell history files.
//
// Plain bash history is one command per line with no metadata. When
// HISTTIMEFORMAT is set, bash precedes each entry with a comment line of the
// form "#<epoch>". In the timestamped form, any lines between two timestamp
// markers belong to a single (multi-line) command, so we join them.
type bashSource struct{}

func (bashSource) Shell() Shell { return ShellBash }

func (bashSource) DefaultPath() (string, bool) {
	if h := os.Getenv("HISTFILE"); h != "" {
		return expandHome(h), true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".bash_history"), true
}

func (bashSource) Parse(data []byte) ([]Raw, error) {
	lines := splitLines(data)
	var out []Raw

	var pending []string    // accumulated command lines for the current entry
	var pendingTS time.Time // timestamp for the current entry, if any
	flush := func() {
		if len(pending) == 0 {
			return
		}
		out = append(out, Raw{
			Command:   strings.Join(pending, "\n"),
			Timestamp: pendingTS,
			Shell:     ShellBash,
		})
		pending = nil
		pendingTS = time.Time{}
	}

	for _, line := range lines {
		if ts, ok := parseBashTimestamp(line); ok {
			// A new timestamped entry begins: close out the previous one.
			flush()
			pendingTS = ts
			continue
		}
		if pendingTS.IsZero() && len(pending) == 0 && strings.TrimSpace(line) == "" {
			continue
		}
		if pendingTS.IsZero() {
			// Plain (untimestamped) history: one command per line.
			pending = append(pending, line)
			flush()
			continue
		}
		// Timestamped history: lines accumulate until the next marker.
		pending = append(pending, line)
	}
	flush()
	return out, nil
}

// parseBashTimestamp recognises a "#<epoch>" HISTTIMEFORMAT marker line.
func parseBashTimestamp(line string) (time.Time, bool) {
	if len(line) < 2 || line[0] != '#' {
		return time.Time{}, false
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(line[1:]), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(epoch, 0), true
}
