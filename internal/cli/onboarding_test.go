package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderOnboarding(t *testing.T) {
	base := onboardInput{enableCmd: `eval "$(yore init zsh)"`, enableRC: "~/.zshrc", hasEntries: true}
	plain := plainStyles()

	t.Run("first run shows welcome + enable CTA", func(t *testing.T) {
		in := base
		text, next := renderOnboarding(in, plain)
		for _, want := range []string{"Welcome to yore", "Ctrl-G", "yore here", `eval "$(yore init zsh)"`, "~/.zshrc"} {
			if !strings.Contains(text, want) {
				t.Errorf("welcome missing %q:\n%s", want, text)
			}
		}
		if !next.Welcomed {
			t.Error("welcome should mark state.Welcomed")
		}
	})

	t.Run("after welcome, slim persistent enable nudge", func(t *testing.T) {
		in := base
		in.state = onboardState{Welcomed: true}
		text, next := renderOnboarding(in, plain)
		if strings.Contains(text, "Welcome to yore") {
			t.Error("should not repeat the welcome")
		}
		if !strings.Contains(text, "isn't recording yet") || !strings.Contains(text, "init zsh") {
			t.Errorf("expected slim enable nudge, got:\n%s", text)
		}
		if next != in.state {
			t.Error("slim nudge should not change state")
		}
	})

	t.Run("empty history adapts the welcome copy", func(t *testing.T) {
		in := base
		in.hasEntries = false
		text, _ := renderOnboarding(in, plain)
		if !strings.Contains(text, "will show up here") {
			t.Errorf("empty-history welcome copy missing:\n%s", text)
		}
	})

	t.Run("hook active first time confirms + points to Ctrl-G", func(t *testing.T) {
		in := base
		in.hookActive = true
		text, next := renderOnboarding(in, plain)
		if !strings.Contains(text, "✓") || !strings.Contains(text, "recording") || !strings.Contains(text, "Ctrl-G") {
			t.Errorf("activation message wrong:\n%s", text)
		}
		if !next.Activated {
			t.Error("should mark state.Activated")
		}
	})

	t.Run("recording but not yet picked shows the Ctrl-G tip", func(t *testing.T) {
		in := base
		in.hookActive = true
		in.state = onboardState{Welcomed: true, Activated: true}
		text, _ := renderOnboarding(in, plain)
		if !strings.Contains(text, "Ctrl-G") || strings.Contains(text, "✓") {
			t.Errorf("expected a plain Ctrl-G tip, got:\n%s", text)
		}
	})

	t.Run("fully onboarded is silent", func(t *testing.T) {
		in := base
		in.hookActive = true
		in.state = onboardState{Welcomed: true, Activated: true, PickUsed: true}
		text, _ := renderOnboarding(in, plain)
		if text != "" {
			t.Errorf("onboarded user should see nothing, got:\n%s", text)
		}
	})
}

func TestEnableLinePerShell(t *testing.T) {
	cases := map[string][2]string{
		"/bin/zsh":               {`eval "$(yore init zsh)"`, "~/.zshrc"},
		"/usr/local/bin/bash":    {`eval "$(yore init bash)"`, "~/.bashrc"},
		"/opt/homebrew/bin/fish": {"yore init fish | source", "~/.config/fish/config.fish"},
		"":                       {`eval "$(yore init zsh)"`, "~/.zshrc"}, // default
	}
	for sh, want := range cases {
		t.Setenv("SHELL", sh)
		cmd, rc := enableLine()
		if cmd != want[0] || rc != want[1] {
			t.Errorf("SHELL=%q → (%q,%q), want (%q,%q)", sh, cmd, rc, want[0], want[1])
		}
	}
}

func TestGuidanceSilentWhenNotTerminal(t *testing.T) {
	// A buffer is not a terminal, so guidance must be suppressed entirely.
	var out bytes.Buffer
	if guidanceEnabled(&out) {
		t.Error("guidance should be disabled for a non-terminal writer")
	}
	var stderr bytes.Buffer
	onboardingFooter(&out, &stderr, true)
	if stderr.Len() != 0 {
		t.Errorf("footer must be silent on a non-terminal, got %q", stderr.String())
	}
}

func TestPickUsedRoundTrip(t *testing.T) {
	withTempData(t)
	if loadOnboardState().PickUsed {
		t.Fatal("fresh state should have PickUsed=false")
	}
	markPickUsed()
	if !loadOnboardState().PickUsed {
		t.Error("markPickUsed should persist PickUsed=true")
	}
}
