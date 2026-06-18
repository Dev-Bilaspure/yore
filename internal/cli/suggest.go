package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dev-Bilaspure/yore/pkg/pattern"
	"github.com/Dev-Bilaspure/yore/pkg/recipe"
	"github.com/Dev-Bilaspure/yore/pkg/store"
)

// nudgeInterval rate-limits the "worth keeping" nudge to roughly once a day.
const nudgeInterval = 20 * time.Hour

// suggestState persists the user's decisions so a dismissed pattern never
// reappears and the nudge doesn't nag.
type suggestState struct {
	Dismissed map[string]bool `json:"dismissed"` // "never" — permanent
	Skipped   map[string]int  `json:"skipped"`   // "skip"  — key -> usage count when skipped (snooze)
	Muted     bool            `json:"muted"`
	LastNudge time.Time       `json:"last_nudge"`
}

func suggestStatePath() (string, bool) {
	st, err := store.Default()
	if err != nil {
		return "", false
	}
	return filepath.Join(filepath.Dir(st.Path()), "suggestions.json"), true
}

func loadSuggestState() suggestState {
	s := suggestState{Dismissed: map[string]bool{}, Skipped: map[string]int{}}
	p, ok := suggestStatePath()
	if !ok {
		return s
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	if s.Dismissed == nil {
		s.Dismissed = map[string]bool{}
	}
	if s.Skipped == nil {
		s.Skipped = map[string]int{}
	}
	return s
}

func saveSuggestState(s suggestState) {
	p, ok := suggestStatePath()
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	if data, err := json.Marshal(s); err == nil {
		_ = os.WriteFile(p, data, 0o644)
	}
}

// suggestionCandidates mines recipe suggestions from recorded history, dropping
// anything already a recipe or previously dismissed.
func suggestionCandidates(minCount int, includeOld bool) []pattern.Candidate {
	st, err := store.Default()
	if err != nil {
		return nil
	}
	events, err := st.Load()
	if err != nil {
		return nil
	}
	obs := make([]pattern.Observation, 0, len(events))
	for _, e := range events {
		if !e.Succeeded() {
			continue
		}
		obs = append(obs, pattern.Observation{Command: e.Command, Dir: e.Dir, Time: e.Time})
	}

	opts := pattern.Options{MinCount: minCount}
	if includeOld {
		opts.RecentWithin = 100 * 365 * 24 * time.Hour
	}
	cands := pattern.Detect(obs, opts)

	state := loadSuggestState()
	recipeKeys := existingRecipeKeys()
	out := cands[:0]
	for _, c := range cands {
		if state.Dismissed[c.Key] || recipeKeys[c.Key] {
			continue
		}
		// Snoozed (skipped) patterns stay hidden until usage roughly doubles
		// since the skip, so they resurface only once they re-earn attention.
		if n, ok := state.Skipped[c.Key]; ok && c.Count < n*2 {
			continue
		}
		out = append(out, c)
	}
	return out
}

func existingRecipeKeys() map[string]bool {
	keys := map[string]bool{}
	for _, r := range loadRecipes() {
		keys[pattern.KeyOf(r.Command)] = true
	}
	return keys
}

const suggestUsage = `Usage: yore suggest [flags]

Review commands you run often and save the keepers as reusable recipes. yore
infers the fill-in-the-blanks from how you've actually used each command.

Flags:
  --min N   only suggest commands run at least N times (default 5)
  --all     consider your whole history, not just recent commands
`

// runSuggest is the interactive review.
func runSuggest(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("yore suggest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, suggestUsage) }
	var (
		minCount int
		all      bool
	)
	fs.IntVar(&minCount, "min", 0, "")
	fs.BoolVar(&all, "all", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cands := suggestionCandidates(minCount, all)
	if len(cands) == 0 {
		fmt.Fprintln(stdout, "Nothing to suggest yet — keep using your shell; yore will spot the commands you rely on.")
		return 0
	}

	s := stderrStyles(stderr)
	in := bufio.NewReader(ttyReader())
	state := loadSuggestState()
	saved, dismissed, skipped := 0, 0, 0

	// Each decision is written to disk the moment it's made, so quitting, a
	// Ctrl+C, or a closed terminal never discards choices already made.
	for _, c := range cands {
		printCandidate(stderr, s, c)
		switch d, name := askCandidate(c, in, stderr, s); d {
		case decSavePersonal:
			if saveCandidate(stderr, s, c, name, false) {
				saved++
			}
		case decSaveProject:
			if saveCandidate(stderr, s, c, name, true) {
				saved++
			}
		case decDismiss:
			state.Dismissed[c.Key] = true
			saveSuggestState(state)
			dismissed++
			fmt.Fprintln(stderr, s.dim("  · won't suggest this again"))
		case decSkip:
			state.Skipped[c.Key] = c.Count
			saveSuggestState(state)
			skipped++
			fmt.Fprintln(stderr, s.dim("  · skipped for now"))
		case decQuit:
			summary(stdout, saved, dismissed, skipped)
			return 0
		}
	}
	summary(stdout, saved, dismissed, skipped)
	return 0
}

type decision int

const (
	decSkip decision = iota
	decSavePersonal
	decSaveProject
	decDismiss
	decQuit
)

func printCandidate(w io.Writer, s styles, c pattern.Candidate) {
	where := ""
	if len(c.Dirs) > 0 {
		where = " in " + collapseHome(c.Dirs[0])
	}
	fmt.Fprintf(w, "\nYou've run this %s%s.\n\n", s.title(fmt.Sprintf("%d×", c.Count)), s.dim(where))
	fmt.Fprintf(w, "    %s\n", s.key(c.Template))
	for _, p := range c.Params {
		fmt.Fprintf(w, "    %s\n", s.dim(fmt.Sprintf("{%s} = %s", p.Name, strings.Join(p.Values, ", "))))
	}
}

func askCandidate(c pattern.Candidate, in *bufio.Reader, w io.Writer, s styles) (decision, string) {
	name := c.Name
	for {
		fmt.Fprintf(w, "\n  %s [%s] as %q   [%s] to project   [%s] rename\n  %s [%s] never suggest   [%s] skip for now   [%s] quit\n  %s ",
			s.dim("save:"), s.key("y"), name, s.key("p"), s.key("r"),
			s.dim("else:"), s.key("n"), s.key("s"), s.key("q"),
			s.dim("›"))
		line, err := in.ReadString('\n')
		choice := strings.ToLower(strings.TrimSpace(line))
		if err != nil && choice == "" { // EOF / no more input — end the session
			return decQuit, name
		}
		switch choice {
		case "y":
			return decSavePersonal, name
		case "p":
			return decSaveProject, name
		case "r":
			fmt.Fprintf(w, "  name [%s]: ", name)
			if n, _ := in.ReadString('\n'); strings.TrimSpace(n) != "" {
				name = strings.TrimSpace(n)
			}
		case "n":
			return decDismiss, name
		case "s", "": // bare Enter skips — never an accidental save
			return decSkip, name
		case "q":
			return decQuit, name
		}
	}
}

func saveCandidate(w io.Writer, s styles, c pattern.Candidate, name string, project bool) bool {
	r := recipe.Recipe{Name: name, Command: c.Template}
	for _, p := range c.Params {
		def := ""
		if len(p.Values) > 0 {
			def = p.Values[0]
		}
		r.Params = append(r.Params, recipe.Param{Name: p.Name, Default: def})
	}
	path, err := saveTarget(project)
	if err != nil {
		fmt.Fprintf(w, "  %s %v\n", s.dim("could not save:"), err)
		return false
	}
	if err := recipe.AppendToFile(path, r); err != nil {
		fmt.Fprintf(w, "  %s %v\n", s.dim("could not save:"), err)
		return false
	}
	fmt.Fprintf(w, "  %s saved %q → %s    run it:  %s\n",
		s.ok("✓"), name, collapseHome(path), s.key("yore run "+name))
	return true
}

func summary(w io.Writer, saved, dismissed, skipped int) {
	if saved == 0 && dismissed == 0 && skipped == 0 {
		return
	}
	fmt.Fprintf(w, "\nDone — %d saved, %d dismissed, %d skipped.\n", saved, dismissed, skipped)
}

// maybeSuggestNudge prints a one-line, rate-limited hint when there are commands
// worth saving. It only fires once recording is on and never on a stale window.
func maybeSuggestNudge(stdout, stderr io.Writer) {
	if !guidanceEnabled(stdout) || !hookActive() {
		return
	}
	state := loadSuggestState()
	if state.Muted || time.Since(state.LastNudge) < nudgeInterval {
		return
	}
	cands := suggestionCandidates(0, false)
	if len(cands) == 0 {
		return
	}
	state.LastNudge = time.Now()
	saveSuggestState(state)

	s := stderrStyles(stderr)
	noun, verb := "command", "looks"
	if len(cands) != 1 {
		noun, verb = "commands", "look"
	}
	fmt.Fprintf(stderr, "\nyore: %d %s %s worth keeping — run %s\n", len(cands), noun, verb, s.key("yore suggest"))
}

// collapseHome shortens an absolute path under $HOME to ~/… for display.
func collapseHome(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
