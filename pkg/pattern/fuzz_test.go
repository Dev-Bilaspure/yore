package pattern

import (
	"strings"
	"testing"
)

// The engine must never panic, whatever the recorded command text looks like.
// Run extended fuzzing with:  go test -run=x -fuzz=FuzzDetect ./pkg/pattern
func FuzzDetect(f *testing.F) {
	seeds := []string{
		"", "   ", "git", "ls -la",
		"git commit -m 'message here'",
		`./a.sh --env=prod -x "quoted value" pos`,
		"a | b && c > /dev/null",
		"--", "-", "---x",
		`broken "unterminated quote`,
		"kubectl -n ns logs -f deploy/api",
		strings.Repeat("tok ", 500),
		"echo 🚀 --flag=ünïcode /päth/file",
		"cmd --a 1 --a 2 --a 3 -- --b",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, cmd string) {
		obs := []Observation{
			{Command: cmd, Dir: "/a", Time: ref},
			{Command: cmd, Dir: "/b", Time: ref},
			{Command: cmd, Dir: "/a", Time: ref},
		}
		_ = Detect(obs, Options{MinCount: 1, Now: ref})
		_ = KeyOf(cmd)
	})
}
