package cli

import (
	"strings"
	"testing"
)

func TestGatherCandidates(t *testing.T) {
	withTempData(t)
	if _, _, c := run(t, "save", "greet", "--", "echo", "hi"); c != 0 {
		t.Fatalf("save exit %d", c)
	}
	if _, _, c := run(t, "record", "--command", "make test", "--exit", "0"); c != 0 {
		t.Fatalf("record exit %d", c)
	}

	cands := gatherCandidates(false)
	if len(cands) < 2 {
		t.Fatalf("expected at least recipe + command, got %d", len(cands))
	}
	// Recipes come first and are marked.
	if cands[0].recipe == nil || !strings.HasPrefix(cands[0].display, recipeSigil) {
		t.Errorf("first candidate should be the recipe, got %+v", cands[0])
	}
	// The recorded command should be present as a plain row.
	found := false
	for _, c := range cands {
		if c.recipe == nil && c.command == "make test" {
			found = true
		}
	}
	if !found {
		t.Error("recorded command 'make test' missing from candidates")
	}
}

func TestMaybeHintFzfShowsOnce(t *testing.T) {
	withTempData(t)
	var first, second strings.Builder
	maybeHintFzf(&first)
	maybeHintFzf(&second)
	if !strings.Contains(first.String(), "fzf") {
		t.Errorf("first call should print the fzf hint, got %q", first.String())
	}
	if second.String() != "" {
		t.Errorf("second call should be silent (already shown), got %q", second.String())
	}
}

func TestSelectByNumber(t *testing.T) {
	cands := []candidate{
		{display: "a", command: "a"},
		{display: "b", command: "b"},
		{display: "c", command: "c"},
	}
	var devnull strings.Builder

	got, ok := selectByNumber(cands, strings.NewReader("2\n"), &devnull)
	if !ok || got.command != "b" {
		t.Errorf("select 2 = %+v, ok=%v; want b", got, ok)
	}
	if _, ok := selectByNumber(cands, strings.NewReader("nope\n"), &devnull); ok {
		t.Error("non-numeric input should cancel")
	}
	if _, ok := selectByNumber(cands, strings.NewReader("99\n"), &devnull); ok {
		t.Error("out-of-range input should cancel")
	}
}
