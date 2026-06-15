package history

import (
	"strings"
	"testing"
	"time"
)

func TestCollectFromStdin(t *testing.T) {
	opts := DefaultOptions()
	opts.Files = []string{"-"}
	opts.Stdin = strings.NewReader(": 1:0;git status\n: 2:0;git status\n: 3:0;ls\n")
	opts.Now = time.Unix(100, 0)
	got, err := Collect(opts)
	if err != nil {
		t.Fatalf("Collect from stdin: %v", err)
	}
	if len(got) != 2 || got[0].Command != "git status" {
		t.Errorf("stdin collect = %#v, want git status first", got)
	}
}

func TestStdinUnavailableErrors(t *testing.T) {
	opts := DefaultOptions()
	opts.Files = []string{"-"} // Stdin left nil
	if _, err := Collect(opts); err == nil {
		t.Fatal("expected error when reading - with no Stdin set")
	}
}

func TestExplicitMissingFileErrors(t *testing.T) {
	opts := DefaultOptions()
	opts.Files = []string{"/no/such/file"}
	opts.FS = memFS{}
	if _, err := Collect(opts); err == nil {
		t.Fatal("expected error for an explicitly named missing file")
	}
}

func TestDiscoveredMissingFileSkipped(t *testing.T) {
	// $SHELL is zsh and its file exists, but bash is also requested via an
	// explicit shell set; the missing bash file must be skipped, not fatal.
	zp, _ := zshSource{}.DefaultPath()
	fs := memFS{zp: ": 1:0;git status\n"} // only zsh file exists
	opts := DefaultOptions()
	opts.Shells = []Shell{ShellZsh, ShellBash} // like --shell all
	opts.FS = fs
	opts.Now = time.Unix(100, 0)
	got, err := Collect(opts)
	if err != nil {
		t.Fatalf("Collect should skip the missing bash file, got error: %v", err)
	}
	if len(got) != 1 || got[0].Command != "git status" {
		t.Errorf("got %#v, want just git status from the zsh file", got)
	}
}

func TestShellFor(t *testing.T) {
	cases := map[string]Shell{
		"/home/u/.bash_history": ShellBash,
		"/home/u/.zsh_history":  ShellZsh,
		"BASH_HISTORY":          ShellBash,
		"/var/log/something":    ShellZsh, // unknown -> zsh parser (also handles plain)
	}
	for path, want := range cases {
		if got := ShellFor(path); got != want {
			t.Errorf("ShellFor(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestDetectShell(t *testing.T) {
	cases := []struct {
		shell  string
		want   Shell
		wantOK bool
	}{
		{"/bin/zsh", ShellZsh, true},
		{"/usr/local/bin/bash", ShellBash, true},
		{"/usr/bin/fish", ShellUnknown, false},
		{"", ShellUnknown, false},
	}
	for _, tt := range cases {
		t.Setenv("SHELL", tt.shell)
		got, ok := detectShell()
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("detectShell() with SHELL=%q = (%v,%v), want (%v,%v)", tt.shell, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestResolveShellsExplicit(t *testing.T) {
	opts := Options{Shells: []Shell{ShellBash}}
	got := resolveShells(memFS{}, opts)
	if len(got) != 1 || got[0] != ShellBash {
		t.Errorf("resolveShells with explicit shells = %v, want [bash]", got)
	}
}

func TestResolveShellsFallbackToExistingFiles(t *testing.T) {
	// $SHELL points at fish (unsupported), so resolution must fall back to
	// whichever registered shells have an existing history file.
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("HISTFILE", "") // force DefaultPath to use ~/.zsh_history etc.

	// Build a memFS where only the zsh default path exists.
	zp, _ := zshSource{}.DefaultPath()
	fs := memFS{zp: ": 1:0;git status\n"}

	got := resolveShells(fs, Options{})
	if len(got) != 1 || got[0] != ShellZsh {
		t.Errorf("fallback resolveShells = %v, want [zsh]", got)
	}
}

func TestReadAllNoHistoryError(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("HISTFILE", "/nonexistent/path/history")
	_, err := readAll(memFS{}, Options{})
	if err == nil {
		t.Fatal("expected an error when no history can be found")
	}
}
