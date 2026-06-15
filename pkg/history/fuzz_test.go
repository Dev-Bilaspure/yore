package history

import "testing"

// The parsers must never panic, regardless of how malformed or adversarial the
// input is. Run extended fuzzing with, e.g.:
//
//	go test -run=x -fuzz=FuzzZshParse ./pkg/history
//
// Without -fuzz, the seed corpus below still runs as ordinary regression tests.

var parserSeeds = []string{
	"",
	"\n\n\n",
	": 1700000000:0;git status\n",
	": notanumber:0;broken header\n",
	":",
	": 1:0;trailing backslash\\",
	"plain command\n",
	"#1700000000\ngit status\n",
	"#notatimestamp\n",
	"\x83\x80\x81 metafied bytes",
	"multi\\\nline\\\ncommand",
	"\r\n\r\n",
}

func FuzzZshParse(f *testing.F) {
	for _, s := range parserSeeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		// Success criterion: no panic, no error for arbitrary bytes.
		if _, err := (zshSource{}).Parse(data); err != nil {
			t.Fatalf("zsh Parse returned error on arbitrary input: %v", err)
		}
	})
}

func FuzzBashParse(f *testing.F) {
	for _, s := range parserSeeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if _, err := (bashSource{}).Parse(data); err != nil {
			t.Fatalf("bash Parse returned error on arbitrary input: %v", err)
		}
	})
}
