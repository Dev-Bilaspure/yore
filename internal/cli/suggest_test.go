package cli

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Dev-Bilaspure/yore/pkg/pattern"
	"github.com/Dev-Bilaspure/yore/pkg/recipe"
	"github.com/Dev-Bilaspure/yore/pkg/store"
)

func cand() pattern.Candidate {
	return pattern.Candidate{
		Name:     "deploy",
		Template: "./deploy.sh --env {env}",
		Params:   []pattern.Param{{Name: "env", Values: []string{"staging", "prod"}}},
		Count:    7,
		Key:      "abc123",
	}
}

func ask(t *testing.T, input string) (decision, string) {
	t.Helper()
	return askCandidate(cand(), bufio.NewReader(strings.NewReader(input)), io.Discard, plainStyles())
}

func TestAskCandidateDecisions(t *testing.T) {
	cases := map[string]decision{
		"y\n": decSavePersonal,
		"p\n": decSaveProject,
		"n\n": decDismiss,
		"s\n": decSkip,
		"\n":  decSkip, // bare Enter is a safe skip, never a save
		"q\n": decQuit,
		"":    decQuit, // EOF ends the session
	}
	for in, want := range cases {
		if got, _ := ask(t, in); got != want {
			t.Errorf("input %q → %v, want %v", in, got, want)
		}
	}
}

func TestAskCandidateRename(t *testing.T) {
	d, name := ask(t, "r\nshipit\ny\n")
	if d != decSavePersonal || name != "shipit" {
		t.Errorf("rename flow → (%v, %q), want (save, shipit)", d, name)
	}
}

func TestSaveCandidateWritesRecipeWithDefaults(t *testing.T) {
	withTempData(t)
	if !saveCandidate(io.Discard, plainStyles(), cand(), "deploy", false) {
		t.Fatal("saveCandidate returned false")
	}
	recipes := loadRecipes()
	r, ok := recipe.Find(recipes, "deploy")
	if !ok {
		t.Fatalf("recipe not saved: %+v", recipes)
	}
	if r.Command != "./deploy.sh --env {env}" {
		t.Errorf("command = %q", r.Command)
	}
	if len(r.Params) != 1 || r.Params[0].Name != "env" || r.Params[0].Default != "staging" {
		t.Errorf("params = %+v, want env defaulting to staging", r.Params)
	}
}

func TestSuggestionCandidatesFiltersDismissedAndRecipes(t *testing.T) {
	withTempData(t)
	st, _ := store.Default()
	now := time.Now()
	for i, env := range []string{"staging", "prod", "staging", "prod", "dev"} {
		_ = st.Append(store.Event{
			Time:    now.Add(time.Duration(i) * time.Second),
			Command: "./deploy.sh --env " + env,
			Dir:     "/proj", Exit: 0, Shell: "zsh",
		})
	}

	got := suggestionCandidates(3, false)
	if len(got) != 1 || got[0].Template != "./deploy.sh --env {env}" {
		t.Fatalf("expected one deploy candidate, got %+v", got)
	}
	key := got[0].Key

	// Dismiss it → gone.
	s := loadSuggestState()
	s.Dismissed[key] = true
	saveSuggestState(s)
	if got := suggestionCandidates(3, false); len(got) != 0 {
		t.Errorf("dismissed candidate should not reappear, got %+v", got)
	}

	// Un-dismiss, then make it a recipe → also gone (already captured).
	s = loadSuggestState()
	delete(s.Dismissed, key)
	saveSuggestState(s)
	if !saveCandidate(io.Discard, plainStyles(), got1(t), "deploy", false) {
		t.Fatal("save failed")
	}
	if got := suggestionCandidates(3, false); len(got) != 0 {
		t.Errorf("already-a-recipe candidate should not reappear, got %+v", got)
	}
}

// got1 re-derives the single candidate for the recipe-save step.
func got1(t *testing.T) pattern.Candidate {
	t.Helper()
	c := suggestionCandidates(3, false)
	if len(c) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(c))
	}
	return c[0]
}

func TestSuggestStateRoundTrip(t *testing.T) {
	withTempData(t)
	s := loadSuggestState()
	if s.Muted || len(s.Dismissed) != 0 || len(s.Skipped) != 0 {
		t.Fatal("fresh state should be empty")
	}
	s.Dismissed["k"] = true
	s.Skipped["sk"] = 6
	s.Muted = true
	saveSuggestState(s)
	got := loadSuggestState()
	if !got.Muted || !got.Dismissed["k"] || got.Skipped["sk"] != 6 {
		t.Errorf("state did not round-trip: %+v", got)
	}
}

// Skipping snoozes a pattern: it stays hidden until the command's usage roughly
// doubles, at which point it has re-earned a suggestion and reappears.
func TestSkipSnoozeHidesUntilUsageDoubles(t *testing.T) {
	withTempData(t)
	st, _ := store.Default()
	now := time.Now()
	add := func(envs []string, base int) {
		for i, env := range envs {
			_ = st.Append(store.Event{
				Time:    now.Add(time.Duration(base+i) * time.Second),
				Command: "./deploy.sh --env " + env,
				Dir:     "/proj", Exit: 0, Shell: "zsh",
			})
		}
	}
	add([]string{"staging", "prod", "staging", "prod", "dev", "qa"}, 0) // 6 runs

	got := suggestionCandidates(3, false)
	if len(got) != 1 {
		t.Fatalf("expected one candidate, got %+v", got)
	}
	key, count := got[0].Key, got[0].Count // count == 6

	// Skip it → snoozed at the current count → hidden.
	s := loadSuggestState()
	s.Skipped[key] = count
	saveSuggestState(s)
	if c := suggestionCandidates(3, false); len(c) != 0 {
		t.Fatalf("snoozed candidate should be hidden, got %+v", c)
	}

	// Use it enough more that usage doubles → it resurfaces.
	add([]string{"a", "b", "c", "d", "e", "f"}, 100)
	got2 := suggestionCandidates(3, false)
	if len(got2) != 1 || got2[0].Count < count*2 {
		t.Fatalf("doubled-usage candidate should resurface, got %+v", got2)
	}
}
