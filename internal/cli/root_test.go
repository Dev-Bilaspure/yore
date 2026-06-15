package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// epipeWriter simulates a downstream pipe that has been closed (e.g. `| head`).
type epipeWriter struct{}

func (epipeWriter) Write([]byte) (int, error) { return 0, syscall.EPIPE }

func run(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func TestRunDefaultOutput(t *testing.T) {
	stdout, stderr, code := run(t, "--file", "testdata/history")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	lines := splitNonEmpty(stdout)
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3: %q", len(lines), stdout)
	}
	// git status (2x) ranks first under frecency.
	if lines[0] != "git status" {
		t.Errorf("first line = %q, want git status", lines[0])
	}
}

func TestRunTopLimit(t *testing.T) {
	stdout, _, code := run(t, "--file", "testdata/history", "-n", "1")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got := splitNonEmpty(stdout); len(got) != 1 || got[0] != "git status" {
		t.Errorf("top 1 = %q, want [git status]", got)
	}
}

func TestRunCountPrefix(t *testing.T) {
	stdout, _, _ := run(t, "--file", "testdata/history", "--count", "-n", "1")
	if !strings.HasPrefix(stdout, "2\tgit status") {
		t.Errorf("output = %q, want it to start with \"2\\tgit status\"", stdout)
	}
}

func TestRunNullSeparated(t *testing.T) {
	stdout, _, _ := run(t, "--file", "testdata/history", "-0")
	if strings.Contains(stdout, "\n") {
		t.Errorf("NUL output should contain no newlines, got %q", stdout)
	}
	if !strings.Contains(stdout, "\x00") {
		t.Errorf("expected NUL separators in %q", stdout)
	}
}

func TestRunRedact(t *testing.T) {
	dir := t.TempDir()
	hist := filepath.Join(dir, "zsh_history")
	contents := "git status\nexport AWS_SECRET_ACCESS_KEY=wJalrXabc123\ndocker ps\n"
	if err := os.WriteFile(hist, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	withRedact, _, _ := run(t, "--file", hist, "--redact")
	if strings.Contains(withRedact, "AWS_SECRET_ACCESS_KEY") {
		t.Errorf("--redact should have hidden the secret, got %q", withRedact)
	}
	if !strings.Contains(withRedact, "git status") {
		t.Errorf("--redact should keep ordinary commands, got %q", withRedact)
	}

	without, _, _ := run(t, "--file", hist)
	if !strings.Contains(without, "AWS_SECRET_ACCESS_KEY") {
		t.Errorf("without --redact the command should be present, got %q", without)
	}
}

func TestRunBrokenPipeIsClean(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"--file", "testdata/history"}, epipeWriter{}, &stderr)
	if code != 0 {
		t.Errorf("broken pipe should exit 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Errorf("broken pipe should be silent, stderr = %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	stdout, _, code := run(t, "--version")
	if code != 0 || !strings.HasPrefix(stdout, "yore ") {
		t.Errorf("version output = %q, code = %d", stdout, code)
	}
}

func TestRunBadFlag(t *testing.T) {
	_, _, code := run(t, "--sort", "nonsense", "--file", "testdata/history")
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for bad sort mode", code)
	}
}

func TestRunUnexpectedArg(t *testing.T) {
	_, stderr, code := run(t, "leftover")
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("stderr = %q, want it to mention unexpected argument", stderr)
	}
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
