package history

import (
	"testing"
	"time"

	"github.com/Dev-Bilaspure/yore/pkg/store"
)

func sampleEvents() []store.Event {
	now := time.Unix(1_000_000, 0)
	return []store.Event{
		{Time: now.Add(-48 * time.Hour), Command: "git status", Dir: "/proj", Exit: 0, Shell: "zsh"},
		{Time: now.Add(-2 * time.Hour), Command: "git status", Dir: "/proj/sub", Exit: 0, Shell: "zsh"},
		{Time: now.Add(-1 * time.Hour), Command: "make test", Dir: "/proj", Exit: 1, Shell: "zsh"},
		{Time: now.Add(-30 * time.Minute), Command: "make test", Dir: "/proj", Exit: 0, Shell: "zsh"},
		{Time: now.Add(-10 * time.Minute), Command: "ls", Dir: "/other", Exit: 0, Shell: "zsh"},
		{Time: now.Add(-5 * time.Minute), Command: "q", Dir: "/proj", Exit: 0}, // filtered (1 char)
	}
}

func TestRankEventsAggregates(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	got := RankEvents(sampleEvents(), EventQuery{Now: now})

	byCmd := map[string]Entry{}
	for _, e := range got {
		byCmd[e.Command] = e
	}
	gs, ok := byCmd["git status"]
	if !ok {
		t.Fatal("git status missing")
	}
	if gs.Count != 2 || gs.Successes != 2 || gs.Failures != 0 {
		t.Errorf("git status = %+v, want count 2 / 2 success", gs)
	}
	if gs.Dir != "/proj/sub" {
		t.Errorf("git status dir = %q, want most-recent /proj/sub", gs.Dir)
	}
	mt := byCmd["make test"]
	if mt.Successes != 1 || mt.Failures != 1 {
		t.Errorf("make test success/failure = %d/%d, want 1/1", mt.Successes, mt.Failures)
	}
	if _, ok := byCmd["q"]; ok {
		t.Error("single-char command q should be filtered")
	}
}

func TestRankEventsDirScope(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	got := RankEvents(sampleEvents(), EventQuery{Dir: "/proj", Now: now})
	for _, e := range got {
		if e.Command == "ls" {
			t.Error("ls ran in /other and must be excluded from /proj scope")
		}
	}
	// git status (incl. /proj/sub) and make test should remain.
	if len(got) != 2 {
		t.Fatalf("dir-scoped results = %d, want 2: %+v", len(got), got)
	}
}

func TestRankEventsSuccessOnly(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	events := []store.Event{
		{Time: now.Add(-time.Hour), Command: "always fails", Dir: "/p", Exit: 1},
		{Time: now.Add(-time.Hour), Command: "always fails", Dir: "/p", Exit: 2},
		{Time: now.Add(-time.Minute), Command: "works", Dir: "/p", Exit: 0},
	}
	got := RankEvents(events, EventQuery{SuccessOnly: true, Now: now})
	if len(got) != 1 || got[0].Command != "works" {
		t.Errorf("SuccessOnly = %+v, want only [works]", got)
	}
}

func TestRankEventsFrecencyOrder(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	got := RankEvents(sampleEvents(), EventQuery{Now: now})
	// "make test" (most recent, 2x) should outrank stale "git status".
	if got[0].Command != "make test" {
		t.Errorf("top = %q, want make test (most recent)", got[0].Command)
	}
}
