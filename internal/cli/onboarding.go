package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dev-Bilaspure/yore/pkg/store"
)

// Onboarding guides a first-time user from install to their first "this is
// useful" moment, entirely from the CLI. It is deliberately quiet: guidance is
// written only to stderr, only on an interactive terminal, never to stdout (so
// pipes and scripts are unaffected), and each stage's nudge is shown until the
// user passes it and then never again.
//
// The journey has three milestones:
//
//  1. value     — the first `yore` shows the commands you already reuse
//  2. activate  — enable the shell hook (the one required step); this is the
//                 north-star: until it happens, every interactive run nudges it
//  3. aha       — once recording, try the Ctrl-G picker
//
// Stage is derived from real state, never scripted: whether the shell hook is
// loaded (a sentinel the init script exports), and small persisted flags.

// onboardState is the small amount of progress we persist between runs.
type onboardState struct {
	Welcomed  bool `json:"welcomed"`  // the first-run welcome has been shown
	Activated bool `json:"activated"` // we've confirmed the hook is enabled at least once
	PickUsed  bool `json:"pick_used"` // the interactive picker has been used
}

// welcomePreview caps how many commands the very first interactive run prints,
// so the first impression is a digestible preview rather than a wall of history.
const welcomePreview = 20

func onboardStatePath() (string, bool) {
	st, err := store.Default()
	if err != nil {
		return "", false
	}
	return filepath.Join(filepath.Dir(st.Path()), "onboarding.json"), true
}

func loadOnboardState() onboardState {
	p, ok := onboardStatePath()
	if !ok {
		return onboardState{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return onboardState{}
	}
	var s onboardState
	_ = json.Unmarshal(data, &s)
	return s
}

func saveOnboardState(s onboardState) {
	p, ok := onboardStatePath()
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

// markPickUsed records that the interactive picker has been used, so the
// "press Ctrl-G" nudge stops appearing.
func markPickUsed() {
	s := loadOnboardState()
	if !s.PickUsed {
		s.PickUsed = true
		saveOnboardState(s)
	}
}

// hookActive reports whether yore's shell integration is loaded in the current
// shell. The init script exports YORE_SESSION, so this is a reliable signal
// rather than a guess.
func hookActive() bool { return os.Getenv("YORE_SESSION") != "" }

// isTerminal reports whether w is a real terminal (character device). Buffers,
// pipes and files are not, so tests and `yore | …` never see guidance.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// guidanceEnabled reports whether interactive guidance should be shown: stdout
// is a terminal and the user hasn't opted out.
func guidanceEnabled(stdout io.Writer) bool {
	return isTerminal(stdout) && os.Getenv("YORE_NO_HINTS") == ""
}

// enableLine returns the shell-specific one-liner that turns yore on, plus the
// rc file it belongs in, based on $SHELL.
func enableLine() (cmd, rcFile string) {
	switch sh := filepath.Base(os.Getenv("SHELL")); {
	case strings.Contains(sh, "fish"):
		return "yore init fish | source", "~/.config/fish/config.fish"
	case strings.Contains(sh, "bash"):
		return `eval "$(yore init bash)"`, "~/.bashrc"
	default:
		return `eval "$(yore init zsh)"`, "~/.zshrc"
	}
}

// styles holds text decorators so rendering stays pure and testable: production
// passes ANSI decorators, tests pass identity functions. Colors use the basic
// ANSI palette so they adapt to the user's terminal theme rather than fighting
// it.
type styles struct {
	title func(string) string // headings / brand
	key   func(string) string // commands and keys the user should type
	dim   func(string) string // secondary text
	ok    func(string) string // success
}

func plainStyles() styles {
	id := func(s string) string { return s }
	return styles{title: id, key: id, dim: id, ok: id}
}

func ansiStyles() styles {
	wrap := func(code string) func(string) string {
		return func(s string) string { return "\033[" + code + "m" + s + "\033[0m" }
	}
	return styles{
		title: wrap("1"),    // bold
		key:   wrap("1;36"), // bold cyan — the things to type
		dim:   wrap("2"),    // dim — secondary text
		ok:    wrap("1;32"), // bold green — success
	}
}

// onboardInput is everything renderOnboarding needs, with no I/O of its own.
type onboardInput struct {
	hookActive bool
	hasEntries bool
	enableCmd  string
	enableRC   string
	state      onboardState
}

// renderOnboarding is the pure heart of the journey: given the current input it
// returns the guidance to print (empty if none) and the state to persist. It
// has no side effects, which is what makes the journey testable.
func renderOnboarding(in onboardInput, s styles) (string, onboardState) {
	st := in.state
	var b strings.Builder

	if in.hookActive {
		switch {
		case !st.Activated:
			// First run after the hook is enabled: confirm it worked (closes the
			// "did that work?" loop) and point at the payoff.
			st.Activated = true
			fmt.Fprintf(&b, "\n%s yore is recording — it now learns each command with its project and exit status.\n", s.ok("✓"))
			fmt.Fprintf(&b, "  Press %s to fuzzy-pick a command onto your prompt.\n", s.key("Ctrl-G"))
		case !st.PickUsed:
			// Recording, but they haven't tried the picker yet — the last nudge.
			fmt.Fprintf(&b, "\n%s press %s to fuzzy-pick a command onto your prompt.\n", s.dim("tip:"), s.key("Ctrl-G"))
		}
		return b.String(), st
	}

	// Hook not active — getting it enabled is the whole job here.
	if !st.Welcomed {
		st.Welcomed = true
		fmt.Fprintf(&b, "\n%s\n", s.title("Welcome to yore."))
		if in.hasEntries {
			fmt.Fprintln(&b, s.dim("The commands above are the ones you reuse most, from your shell history."))
		} else {
			fmt.Fprintln(&b, s.dim("Enable yore and the commands you reuse most will show up here."))
		}
		fmt.Fprintln(&b, "\nEnable yore to unlock the rest:")
		fmt.Fprintf(&b, "  %s live recording of every command, with its project and exit status\n", s.dim("·"))
		fmt.Fprintf(&b, "  %s %s — the commands you actually run in the current project\n", s.dim("·"), s.key("yore here"))
		fmt.Fprintf(&b, "  %s %s — fuzzy-pick any command straight onto your prompt\n", s.dim("·"), s.key("Ctrl-G"))
		fmt.Fprintf(&b, "\nAdd this line, then restart your shell:\n  %s   %s\n", s.key(in.enableCmd), s.dim("# "+in.enableRC))
		return b.String(), st
	}

	// Seen the welcome, still not enabled: a slim, persistent nudge toward the
	// one step that matters. It stays until the hook is on, then is gone for good.
	fmt.Fprintf(&b, "\n%s enable recording for %s and %s:\n  %s   %s\n",
		s.dim("yore isn't recording yet —"), s.key("yore here"), s.key("Ctrl-G"),
		s.key(in.enableCmd), s.dim("# "+in.enableRC))
	return b.String(), st
}

// onboardingFooter renders stage-appropriate guidance to stderr and persists any
// state change. It is a no-op unless stdout is an interactive terminal and the
// user hasn't opted out, so pipes, scripts and tests never see it.
func onboardingFooter(stdout, stderr io.Writer, hasEntries bool) {
	if !guidanceEnabled(stdout) {
		return
	}
	cmd, rc := enableLine()
	in := onboardInput{
		hookActive: hookActive(),
		hasEntries: hasEntries,
		enableCmd:  cmd,
		enableRC:   rc,
		state:      loadOnboardState(),
	}
	st := plainStyles()
	if isTerminal(stderr) && os.Getenv("NO_COLOR") == "" {
		st = ansiStyles()
	}
	text, next := renderOnboarding(in, st)
	if text != "" {
		fmt.Fprint(stderr, text)
	}
	if next != in.state {
		saveOnboardState(next)
	}
}

// onFirstInteractiveRun reports whether this is the very first interactive,
// not-yet-enabled run — used to cap the preview so the first impression is
// digestible rather than a full history dump.
func onFirstInteractiveRun(stdout io.Writer) bool {
	return guidanceEnabled(stdout) && !hookActive() && !loadOnboardState().Welcomed
}
