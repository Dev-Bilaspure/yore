package history

// dedup groups identical commands into a single [Entry], summing occurrence
// counts and keeping the most recent timestamp (and the shell that produced
// it). The first-seen order is preserved so that [SortRaw] is meaningful.
func dedup(raws []Raw) []Entry {
	index := make(map[string]int, len(raws))
	entries := make([]Entry, 0, len(raws))

	for _, r := range raws {
		if i, ok := index[r.Command]; ok {
			e := &entries[i]
			e.Count++
			if r.Timestamp.After(e.LastUsed) {
				e.LastUsed = r.Timestamp
				e.Shell = r.Shell
			}
			continue
		}
		index[r.Command] = len(entries)
		entries = append(entries, Entry{
			Command:  r.Command,
			Count:    1,
			LastUsed: r.Timestamp,
			Shell:    r.Shell,
		})
	}
	return entries
}

// passthrough converts raw occurrences to entries without deduplication, one
// entry per occurrence, preserving order.
func passthrough(raws []Raw) []Entry {
	entries := make([]Entry, 0, len(raws))
	for _, r := range raws {
		entries = append(entries, Entry{
			Command:  r.Command,
			Count:    1,
			LastUsed: r.Timestamp,
			Shell:    r.Shell,
		})
	}
	return entries
}
