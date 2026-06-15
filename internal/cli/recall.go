package cli

import (
	"strconv"

	"github.com/Dev-Bilaspure/yore/pkg/history"
	"github.com/Dev-Bilaspure/yore/pkg/store"
)

// histfileEvents reads the given shell history files (or auto-detected ones)
// as individual occurrences and converts them to events. These events carry no
// directory or exit information, since plain history files record neither.
func histfileEvents(shells []history.Shell, files []string) ([]store.Event, error) {
	opts := history.DefaultOptions()
	opts.Dedup = false // keep one entry per occurrence
	opts.Sort = history.SortRaw
	opts.Shells = shells
	opts.Files = files

	entries, err := history.Collect(opts)
	if err != nil {
		return nil, err
	}
	events := make([]store.Event, 0, len(entries))
	for _, e := range entries {
		events = append(events, store.Event{
			Time:    e.LastUsed,
			Command: e.Command,
			Exit:    store.ExitUnknown,
			Shell:   e.Shell.String(),
		})
	}
	return events, nil
}

// recallKey identifies an execution by its second-resolution timestamp and
// command. It is used only to suppress a live-history entry that duplicates a
// recorded one (the hook records a command and the shell also writes it to the
// history file); recorded events are never deduplicated against each other.
func recallKey(e store.Event) string {
	return strconv.FormatInt(e.Time.Unix(), 10) + "\x00" + e.Command
}

// loadRecallEvents returns the events used for global recall: every recorded
// event, supplemented with live history-file entries that are not already
// recorded. This makes yore useful before the recording hook is installed,
// without double-counting once it is.
func loadRecallEvents() ([]store.Event, error) {
	st, err := store.Default()
	if err != nil {
		return nil, err
	}
	recorded, err := st.Load()
	if err != nil {
		return nil, err
	}

	live, err := histfileEvents(nil, nil)
	if err != nil {
		live = nil // a missing history file is fine
	}

	seen := make(map[string]struct{}, len(recorded))
	for _, e := range recorded {
		seen[recallKey(e)] = struct{}{}
	}
	out := recorded
	for _, e := range live {
		if _, ok := seen[recallKey(e)]; ok {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}
