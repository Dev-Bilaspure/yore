package history

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// memFS is an in-memory FS for tests.
type memFS map[string]string

func (m memFS) Open(name string) (io.ReadCloser, error) {
	data, ok := m[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(data)), nil
}

func (m memFS) Stat(name string) error {
	if _, ok := m[name]; ok {
		return nil
	}
	return os.ErrNotExist
}

func TestCollectDedupAndRank(t *testing.T) {
	const file = "/fake/.zsh_history"
	fs := memFS{file: strings.Join([]string{
		": 1700000000:0;git status",
		": 1700000100:0;ls",
		": 1700000200:0;git status",
		": 1700000300:0;git status",
		": 1700000400:0;q", // dropped: single char
		"",                 // dropped: blank
	}, "\n") + "\n"}

	opts := DefaultOptions()
	opts.Files = []string{file}
	opts.FS = fs
	opts.Now = time.Unix(1700000400, 0)

	entries, err := Collect(opts)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// "git status" (3x, recent) should outrank "ls" (1x); "q" is filtered out.
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %#v", len(entries), entries)
	}
	if entries[0].Command != "git status" || entries[0].Count != 3 {
		t.Errorf("entry[0] = %+v, want git status x3", entries[0])
	}
	if entries[1].Command != "ls" {
		t.Errorf("entry[1] = %q, want ls", entries[1].Command)
	}
}

func TestCollectLimitAndMinCount(t *testing.T) {
	const file = "/fake/.zsh_history"
	fs := memFS{file: strings.Join([]string{
		": 1:0;alpha",
		": 2:0;alpha",
		": 3:0;beta",
		": 4:0;gamma",
		": 5:0;gamma",
		": 6:0;gamma",
	}, "\n") + "\n"}

	t.Run("limit", func(t *testing.T) {
		opts := DefaultOptions()
		opts.Files = []string{file}
		opts.FS = fs
		opts.Limit = 1
		opts.Now = time.Unix(100, 0)
		got, err := Collect(opts)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Command != "gamma" {
			t.Errorf("limit: got %#v, want [gamma]", got)
		}
	})

	t.Run("min-count", func(t *testing.T) {
		opts := DefaultOptions()
		opts.Files = []string{file}
		opts.FS = fs
		opts.MinCount = 2
		opts.Now = time.Unix(100, 0)
		got, err := Collect(opts)
		if err != nil {
			t.Fatal(err)
		}
		// beta (count 1) excluded; gamma (3) and alpha (2) remain.
		if len(got) != 2 {
			t.Fatalf("min-count: got %d entries, want 2: %#v", len(got), got)
		}
		for _, e := range got {
			if e.Command == "beta" {
				t.Errorf("beta should be excluded by min-count")
			}
		}
	})
}

func TestCollectNoDedup(t *testing.T) {
	const file = "/fake/.zsh_history"
	fs := memFS{file: ": 1:0;same\n: 2:0;same\n: 3:0;same\n"}
	opts := DefaultOptions()
	opts.Files = []string{file}
	opts.FS = fs
	opts.Dedup = false
	opts.Sort = SortRaw
	opts.Now = time.Unix(100, 0)
	got, err := Collect(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("no-dedup: got %d entries, want 3", len(got))
	}
}

func TestCollectFromTestdataFixtures(t *testing.T) {
	// Exercises the real os filesystem against checked-in fixtures.
	opts := DefaultOptions()
	opts.Files = []string{"testdata/zsh_history", "testdata/bash_history"}
	opts.Now = time.Unix(1700000600, 0)
	got, err := Collect(opts)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected some entries from fixtures")
	}
	// "git status" appears in both fixtures and most often, so it should rank
	// first under frecency.
	if got[0].Command != "git status" {
		t.Errorf("top command = %q, want git status", got[0].Command)
	}
	// The single-char "q" entry in the zsh fixture must be filtered out.
	for _, e := range got {
		if e.Command == "q" {
			t.Errorf("single-char command q should be filtered out")
		}
	}
}

func TestParseSortMode(t *testing.T) {
	cases := map[string]SortMode{
		"":         SortFrecency,
		"frecency": SortFrecency,
		"recent":   SortRecent,
		"frequent": SortFrequent,
		"raw":      SortRaw,
	}
	for in, want := range cases {
		got, err := ParseSortMode(in)
		if err != nil {
			t.Errorf("ParseSortMode(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("ParseSortMode(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := ParseSortMode("bogus"); err == nil {
		t.Error("expected error for bogus sort mode")
	}
}
