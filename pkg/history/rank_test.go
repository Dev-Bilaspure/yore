package history

import (
	"testing"
	"time"
)

func TestFrecencyDecay(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	half := 24 * time.Hour

	tests := []struct {
		name string
		e    Entry
		want float64
	}{
		{
			name: "no timestamp falls back to count",
			e:    Entry{Count: 5},
			want: 5,
		},
		{
			name: "used now keeps full weight",
			e:    Entry{Count: 4, LastUsed: now},
			want: 4,
		},
		{
			name: "one half-life ago halves the weight",
			e:    Entry{Count: 4, LastUsed: now.Add(-half)},
			want: 2,
		},
		{
			name: "two half-lives ago quarters the weight",
			e:    Entry{Count: 4, LastUsed: now.Add(-2 * half)},
			want: 1,
		},
		{
			name: "future timestamp clamped to now",
			e:    Entry{Count: 3, LastUsed: now.Add(half)},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := frecency(tt.e, half, now)
			if !approxEqual(got, tt.want, 1e-9) {
				t.Errorf("frecency = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRankFrecencyOrder(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	half := 24 * time.Hour

	// "rare-but-recent" used once just now; "old-favourite" used 100 times but
	// four half-lives ago (weight 100/16 = 6.25). Frecency should still favour
	// the frequent one here, but a recent burst should beat a stale single use.
	entries := []Entry{
		{Command: "old-favourite", Count: 100, LastUsed: now.Add(-4 * half)},
		{Command: "rare-but-recent", Count: 1, LastUsed: now},
		{Command: "fresh-and-frequent", Count: 10, LastUsed: now},
	}
	Rank(entries, SortFrecency, half, now)

	wantOrder := []string{"fresh-and-frequent", "old-favourite", "rare-but-recent"}
	assertOrder(t, entries, wantOrder)
}

func TestRankModes(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	half := 24 * time.Hour
	base := func() []Entry {
		return []Entry{
			{Command: "a", Count: 1, LastUsed: now.Add(-3 * time.Hour)},
			{Command: "b", Count: 9, LastUsed: now.Add(-100 * time.Hour)},
			{Command: "c", Count: 3, LastUsed: now.Add(-1 * time.Hour)},
		}
	}

	t.Run("recent", func(t *testing.T) {
		e := base()
		Rank(e, SortRecent, half, now)
		assertOrder(t, e, []string{"c", "a", "b"})
	})
	t.Run("frequent", func(t *testing.T) {
		e := base()
		Rank(e, SortFrequent, half, now)
		assertOrder(t, e, []string{"b", "c", "a"})
	})
	t.Run("raw preserves input order", func(t *testing.T) {
		e := base()
		Rank(e, SortRaw, half, now)
		assertOrder(t, e, []string{"a", "b", "c"})
	})
}

func TestRankFrecencyTieBreak(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	// Identical score and timestamp: break ties alphabetically for determinism.
	entries := []Entry{
		{Command: "zebra", Count: 1, LastUsed: now},
		{Command: "apple", Count: 1, LastUsed: now},
	}
	Rank(entries, SortFrecency, time.Hour, now)
	assertOrder(t, entries, []string{"apple", "zebra"})
}

func approxEqual(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func assertOrder(t *testing.T, entries []Entry, want []string) {
	t.Helper()
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i := range want {
		if entries[i].Command != want[i] {
			got := make([]string, len(entries))
			for j, e := range entries {
				got[j] = e.Command
			}
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}
