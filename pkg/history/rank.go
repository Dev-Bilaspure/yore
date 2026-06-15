package history

import (
	"math"
	"sort"
	"time"
)

// Rank assigns a frecency score to every entry and sorts the slice in place
// according to mode. halfLife controls the recency decay and now is the
// reference time. Frecency is always computed (so [Entry.Score] is populated)
// regardless of the chosen sort mode.
func Rank(entries []Entry, mode SortMode, halfLife time.Duration, now time.Time) {
	for i := range entries {
		entries[i].Score = frecency(entries[i], halfLife, now)
	}

	switch mode {
	case SortRaw:
		// Leave first-seen order untouched.
		return
	case SortRecent:
		sort.SliceStable(entries, func(i, j int) bool {
			if !entries[i].LastUsed.Equal(entries[j].LastUsed) {
				return entries[i].LastUsed.After(entries[j].LastUsed)
			}
			return entries[i].Command < entries[j].Command
		})
	case SortFrequent:
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Count != entries[j].Count {
				return entries[i].Count > entries[j].Count
			}
			return entries[i].Command < entries[j].Command
		})
	default: // SortFrecency
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Score != entries[j].Score {
				return entries[i].Score > entries[j].Score
			}
			if !entries[i].LastUsed.Equal(entries[j].LastUsed) {
				return entries[i].LastUsed.After(entries[j].LastUsed)
			}
			return entries[i].Command < entries[j].Command
		})
	}
}

// frecency blends frequency and recency using exponential decay:
//
//	score = count * 2^(-age / halfLife)
//
// A command keeps at least half its frequency weight while it is younger than
// one half-life, and decays smoothly thereafter. When an entry has no
// timestamp (LastUsed is zero), recency is unknown and the score falls back to
// the raw count, so timestamp-less history still ranks by pure frequency.
func frecency(e Entry, halfLife time.Duration, now time.Time) float64 {
	count := float64(e.Count)
	if e.LastUsed.IsZero() || halfLife <= 0 {
		return count
	}
	age := now.Sub(e.LastUsed)
	if age < 0 {
		age = 0 // a clock skew into the future should not boost a command
	}
	decay := math.Exp2(-float64(age) / float64(halfLife))
	return count * decay
}
