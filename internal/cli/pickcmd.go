package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Dev-Bilaspure/yore/pkg/history"
	"github.com/Dev-Bilaspure/yore/pkg/recipe"
)

// recipeSigil marks recipe rows in the picker so they stand out and can be
// mapped back to a recipe after selection.
const recipeSigil = "★ "

// candidate is one selectable row in the picker.
type candidate struct {
	display string         // the line shown in the picker
	command string         // the command to emit (for plain history rows)
	recipe  *recipe.Recipe // non-nil for recipe rows
}

// pickLimit caps how many history commands are offered to the picker.
const pickLimit = 2000

// runPick handles `yore pick`: fuzzy-select a command or recipe and print the
// resulting command to stdout. It is the engine behind the Ctrl-G key-binding,
// but also works standalone. Parameter prompts and the picker UI use the
// terminal, so stdout carries only the final command.
func runPick(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("yore pick", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var here bool
	fs.BoolVar(&here, "here", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cands := gatherCandidates(here)
	if len(cands) == 0 {
		return 0
	}

	sel, ok := selectCandidate(cands, stderr)
	if !ok {
		return 0 // cancelled — nothing to insert
	}

	if sel.recipe != nil {
		values := promptParams(ttyReader(), stderr, sel.recipe.Params, false)
		fmt.Fprintln(stdout, sel.recipe.Render(values))
		return 0
	}
	fmt.Fprintln(stdout, sel.command)
	return 0
}

// gatherCandidates assembles the picker rows: curated recipes first (they are
// the highest-signal items), then frecency-ranked history.
func gatherCandidates(here bool) []candidate {
	var cands []candidate

	for _, r := range loadRecipes() {
		rr := r
		label := r.Description
		if label == "" {
			label = r.Command
		}
		cands = append(cands, candidate{
			display: recipeSigil + r.Name + "  —  " + label,
			recipe:  &rr,
		})
	}

	events, err := loadRecallEvents()
	if err == nil {
		q := history.EventQuery{Sort: history.SortFrecency, Limit: pickLimit}
		if here {
			q.Dir = projectRoot(currentDir())
		}
		for _, e := range history.RankEvents(events, q) {
			cands = append(cands, candidate{display: e.Command, command: e.Command})
		}
	}
	return cands
}

// selectCandidate runs the interactive selection, preferring fzf and falling
// back to a numbered prompt. The bool is false when the user cancels.
func selectCandidate(cands []candidate, stderr io.Writer) (candidate, bool) {
	if fzfPath, err := exec.LookPath("fzf"); err == nil {
		return selectWithFzf(fzfPath, cands)
	}
	return selectByNumber(cands, ttyReader(), stderr)
}

// selectWithFzf pipes the candidate displays through fzf and maps the chosen
// line back to its candidate.
func selectWithFzf(fzfPath string, cands []candidate) (candidate, bool) {
	byDisplay := make(map[string]candidate, len(cands))
	var input strings.Builder
	for _, c := range cands {
		byDisplay[c.display] = c
		input.WriteString(c.display)
		input.WriteByte('\n')
	}

	cmd := exec.Command(fzfPath, "--height=40%", "--reverse", "--no-multi",
		"--prompt=yore> ", "--no-sort")
	cmd.Stdin = strings.NewReader(input.String())
	cmd.Stderr = os.Stderr // fzf draws its UI on the controlling terminal
	out, err := cmd.Output()
	if err != nil {
		return candidate{}, false // user aborted (e.g. Esc) or fzf failed
	}
	line := strings.TrimRight(string(out), "\r\n")
	c, ok := byDisplay[line]
	return c, ok
}

// selectByNumber is the no-fzf fallback: it lists candidates and reads a number.
func selectByNumber(cands []candidate, in io.Reader, stderr io.Writer) (candidate, bool) {
	limit := len(cands)
	if limit > 30 {
		limit = 30 // keep the fallback list manageable
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(stderr, "%3d  %s\n", i+1, cands[i].display)
	}
	fmt.Fprint(stderr, "select> ")
	line, _ := bufio.NewReader(in).ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > limit {
		return candidate{}, false
	}
	return cands[n-1], true
}

// ttyReader returns a reader for the controlling terminal, used for prompts
// when stdin is otherwise occupied (e.g. driving fzf). It falls back to stdin.
func ttyReader() io.Reader {
	if f, err := os.Open("/dev/tty"); err == nil {
		return f
	}
	return os.Stdin
}
