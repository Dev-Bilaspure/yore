package history

import (
	"testing"
	"time"
)

func TestZshParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Raw
	}{
		{
			name: "extended format with timestamp",
			in:   ": 1700000000:0;git status\n",
			want: []Raw{{Command: "git status", Timestamp: time.Unix(1700000000, 0), Shell: ShellZsh}},
		},
		{
			name: "plain format without timestamp",
			in:   "git status\nls -la\n",
			want: []Raw{
				{Command: "git status", Shell: ShellZsh},
				{Command: "ls -la", Shell: ShellZsh},
			},
		},
		{
			name: "elapsed seconds are ignored",
			in:   ": 1700000000:42;make build\n",
			want: []Raw{{Command: "make build", Timestamp: time.Unix(1700000000, 0), Shell: ShellZsh}},
		},
		{
			name: "multi-line command joined on trailing backslash",
			in:   ": 1700000000:0;for i in 1 2; do\\\necho $i\\\ndone\n",
			want: []Raw{{
				Command:   "for i in 1 2; do\necho $i\ndone",
				Timestamp: time.Unix(1700000000, 0),
				Shell:     ShellZsh,
			}},
		},
		{
			name: "command containing a semicolon",
			in:   ": 1700000000:0;cd /tmp; ls\n",
			want: []Raw{{Command: "cd /tmp; ls", Timestamp: time.Unix(1700000000, 0), Shell: ShellZsh}},
		},
		{
			name: "blank lines are skipped",
			in:   "git status\n\n\nls\n",
			want: []Raw{
				{Command: "git status", Shell: ShellZsh},
				{Command: "ls", Shell: ShellZsh},
			},
		},
		{
			name: "missing trailing newline",
			in:   "git status",
			want: []Raw{{Command: "git status", Shell: ShellZsh}},
		},
		{
			name: "crlf line endings",
			in:   "git status\r\nls\r\n",
			want: []Raw{
				{Command: "git status", Shell: ShellZsh},
				{Command: "ls", Shell: ShellZsh},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := zshSource{}.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			assertRaws(t, got, tt.want)
		})
	}
}

func TestZshUnmetafy(t *testing.T) {
	// zsh metafies the UTF-8 bytes of "é" (0xC3 0xA9): each escaped byte is
	// written as zshMeta followed by byte^0x20.
	meta := []byte{':', ' ', '1', ':', '0', ';', 'e', 'c', 'h', 'o', ' ',
		zshMeta, 0xC3 ^ 0x20, zshMeta, 0xA9 ^ 0x20, '\n'}
	got, err := zshSource{}.Parse(meta)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d raws, want 1", len(got))
	}
	if got[0].Command != "echo é" {
		t.Errorf("command = %q, want %q", got[0].Command, "echo é")
	}
}

func TestUnmetafyNoOp(t *testing.T) {
	in := []byte("plain ascii, no meta bytes")
	if got := unmetafy(in); string(got) != string(in) {
		t.Errorf("unmetafy mangled clean input: %q", got)
	}
}

func assertRaws(t *testing.T, got, want []Raw) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d raws, want %d: %#v", len(got), len(want), got)
	}
	for i := range got {
		if got[i].Command != want[i].Command {
			t.Errorf("[%d] command = %q, want %q", i, got[i].Command, want[i].Command)
		}
		if !got[i].Timestamp.Equal(want[i].Timestamp) {
			t.Errorf("[%d] timestamp = %v, want %v", i, got[i].Timestamp, want[i].Timestamp)
		}
		if got[i].Shell != want[i].Shell {
			t.Errorf("[%d] shell = %v, want %v", i, got[i].Shell, want[i].Shell)
		}
	}
}
