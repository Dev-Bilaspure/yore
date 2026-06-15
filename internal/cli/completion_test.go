package cli

import (
	"strings"
	"testing"
)

func TestCompletionScripts(t *testing.T) {
	cases := map[string]string{
		"bash": "complete -F _yore yore",
		"zsh":  "#compdef yore",
		"fish": "complete -c yore",
	}
	for shell, marker := range cases {
		t.Run(shell, func(t *testing.T) {
			stdout, stderr, code := run(t, "completion", shell)
			if code != 0 {
				t.Fatalf("exit = %d, stderr = %q", code, stderr)
			}
			if !strings.Contains(stdout, marker) {
				t.Errorf("%s completion missing %q; got:\n%s", shell, marker, stdout)
			}
			// Every script should reference the value choices for --sort.
			if !strings.Contains(stdout, "frecency") {
				t.Errorf("%s completion does not complete --sort values", shell)
			}
		})
	}
}

func TestCompletionErrors(t *testing.T) {
	if _, _, code := run(t, "completion"); code != 2 {
		t.Errorf("no shell arg: exit = %d, want 2", code)
	}
	if _, _, code := run(t, "completion", "powershell"); code != 2 {
		t.Errorf("unknown shell: exit = %d, want 2", code)
	}
	if stdout, _, code := run(t, "completion", "--help"); code != 0 || !strings.Contains(stdout, "Usage") {
		t.Errorf("completion --help: exit = %d, stdout = %q", code, stdout)
	}
}
