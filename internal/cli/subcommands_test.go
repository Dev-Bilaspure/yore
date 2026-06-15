package cli

import (
	"strings"
	"testing"
)

// withTempData points the store at a throwaway XDG data dir and HISTFILE at an
// empty file so recall sees only what the test records.
func withTempData(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	empty := t.TempDir() + "/empty_history"
	t.Setenv("HISTFILE", empty)
	t.Setenv("SHELL", "/bin/zsh")
}

func TestRecordThenRecall(t *testing.T) {
	withTempData(t)

	rec := func(dir, cmd string, exit int) {
		_, _, code := run(t, "record", "--shell", "zsh", "--dir", dir, "--exit", itoa(exit), "--command", cmd)
		if code != 0 {
			t.Fatalf("record %q exit %d", cmd, code)
		}
	}
	rec("/proj", "make test", 0)
	rec("/proj", "make test", 0)
	rec("/proj", "broken cmd", 1)
	rec("/other", "elsewhere cmd", 0)

	// Global recall sees everything.
	stdout, _, code := run(t, "--count")
	if code != 0 {
		t.Fatalf("recall exit %d", code)
	}
	if !strings.Contains(stdout, "make test") || !strings.Contains(stdout, "elsewhere cmd") {
		t.Errorf("global recall missing commands:\n%s", stdout)
	}
	if !strings.Contains(stdout, "2\tmake test") {
		t.Errorf("expected make test count 2, got:\n%s", stdout)
	}
}

func TestRecordEmptyCommandIsNoop(t *testing.T) {
	withTempData(t)
	_, _, code := run(t, "record", "--command", "   ")
	if code != 0 {
		t.Errorf("empty record should be a no-op success, got %d", code)
	}
}

func TestRecordSkipsSecrets(t *testing.T) {
	withTempData(t)
	run(t, "record", "--command", "git status", "--exit", "0")
	run(t, "record", "--command", "mysql -u root --password hunter2", "--exit", "0")
	run(t, "record", "--command", "export AWS_SECRET_ACCESS_KEY=wJalrXabc123", "--exit", "0")

	out, _, _ := run(t)
	if strings.Contains(out, "hunter2") || strings.Contains(out, "AWS_SECRET_ACCESS_KEY") {
		t.Errorf("secrets must not be recorded or recalled, got:\n%s", out)
	}
	if !strings.Contains(out, "git status") {
		t.Errorf("ordinary command should still be recorded, got:\n%s", out)
	}
}

func TestInitSubcommand(t *testing.T) {
	for _, sh := range []string{"zsh", "bash", "fish"} {
		stdout, _, code := run(t, "init", sh)
		if code != 0 || !strings.Contains(stdout, "yore record") {
			t.Errorf("init %s: code=%d, output should install the record hook", sh, code)
		}
	}
	if _, _, code := run(t, "init", "powershell"); code != 2 {
		t.Errorf("init with unknown shell should exit 2")
	}
}

func TestImportSeedsStore(t *testing.T) {
	withTempData(t)
	// Point import at a known fixture rather than the user's real history.
	stdout, stderr, code := run(t, "import", "--file", "../../pkg/history/testdata/zsh_history")
	if code != 0 {
		t.Fatalf("import exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "imported") {
		t.Errorf("import should report a count, got %q", stdout)
	}
	// A second import is idempotent: nothing new.
	stdout2, _, _ := run(t, "import", "--file", "../../pkg/history/testdata/zsh_history")
	if !strings.Contains(stdout2, "imported 0 commands") {
		t.Errorf("re-import should add 0, got %q", stdout2)
	}
}

// itoa avoids pulling strconv into the test for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
