package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndLoad(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "history.jsonl"))
	if !s.IsEmpty() {
		t.Fatal("new store should be empty")
	}

	want := []Event{
		{Time: time.Unix(1000, 0), Command: "git status", Dir: "/a", Exit: 0, Shell: "zsh"},
		{Time: time.Unix(1001, 0), Command: "make test", Dir: "/a", Exit: 1, Duration: 2 * time.Second, Shell: "zsh"},
		{Time: time.Unix(1002, 0), Command: "ls", Dir: "/b", Exit: ExitUnknown},
	}
	for _, e := range want {
		if err := s.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if s.IsEmpty() {
		t.Fatal("store should not be empty after appends")
	}

	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("loaded %d events, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Command != want[i].Command || got[i].Dir != want[i].Dir ||
			got[i].Exit != want[i].Exit || !got[i].Time.Equal(want[i].Time) ||
			got[i].Duration != want[i].Duration {
			t.Errorf("event[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "nope.jsonl"))
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load of missing file should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no events, got %d", len(got))
	}
}

func TestLoadSkipsCorruptLines(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "history.jsonl"))
	_ = s.Append(Event{Time: time.Unix(1, 0), Command: "good one", Exit: 0})
	// Manually append a corrupt line.
	if err := appendRaw(s.Path(), "{not valid json\n"); err != nil {
		t.Fatal(err)
	}
	_ = s.Append(Event{Time: time.Unix(2, 0), Command: "good two", Exit: 0})

	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 valid events (corrupt skipped), got %d", len(got))
	}
}

func TestSucceeded(t *testing.T) {
	cases := []struct {
		exit int
		want bool
	}{{0, true}, {ExitUnknown, true}, {1, false}, {127, false}}
	for _, c := range cases {
		if got := (Event{Exit: c.exit}).Succeeded(); got != c.want {
			t.Errorf("Succeeded(exit=%d) = %v, want %v", c.exit, got, c.want)
		}
	}
}

func TestDefaultHonoursXDG(t *testing.T) {
	base := t.TempDir() // an absolute, OS-native path
	t.Setenv("XDG_DATA_HOME", base)
	s, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	// Build the expectation with filepath.Join so separators match on Windows.
	if want := filepath.Join(base, "yore", "history.jsonl"); s.Path() != want {
		t.Errorf("Default path = %q, want %q", s.Path(), want)
	}
}

// appendRaw is a test helper that appends a raw string to a file.
func appendRaw(path, s string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(s)
	return err
}
