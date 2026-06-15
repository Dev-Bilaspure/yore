package history

import (
	"testing"
	"time"
)

func TestBashParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Raw
	}{
		{
			name: "plain one command per line",
			in:   "git status\nls -la\n",
			want: []Raw{
				{Command: "git status", Shell: ShellBash},
				{Command: "ls -la", Shell: ShellBash},
			},
		},
		{
			name: "timestamped entries",
			in:   "#1700000000\ngit status\n#1700000100\ndocker ps\n",
			want: []Raw{
				{Command: "git status", Timestamp: time.Unix(1700000000, 0), Shell: ShellBash},
				{Command: "docker ps", Timestamp: time.Unix(1700000100, 0), Shell: ShellBash},
			},
		},
		{
			name: "multi-line command between timestamps",
			in:   "#1700000000\necho one\ntwo\n#1700000100\nls\n",
			want: []Raw{
				{Command: "echo one\ntwo", Timestamp: time.Unix(1700000000, 0), Shell: ShellBash},
				{Command: "ls", Timestamp: time.Unix(1700000100, 0), Shell: ShellBash},
			},
		},
		{
			name: "a command line that itself starts with # but is not a timestamp",
			in:   "#1700000000\n# not a timestamp because not all digits\n",
			want: []Raw{
				{Command: "# not a timestamp because not all digits", Timestamp: time.Unix(1700000000, 0), Shell: ShellBash},
			},
		},
		{
			name: "blank lines in plain history are skipped",
			in:   "git status\n\nls\n",
			want: []Raw{
				{Command: "git status", Shell: ShellBash},
				{Command: "ls", Shell: ShellBash},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bashSource{}.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			assertRaws(t, got, tt.want)
		})
	}
}
