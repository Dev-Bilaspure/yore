package history

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// zshSource parses Z shell history files.
//
// zsh writes two formats. With EXTENDED_HISTORY, each entry is:
//
//	: <start-epoch>:<elapsed-seconds>;<command>
//
// Without it, each entry is simply the command on its own line. Multi-line
// commands are stored with embedded newlines escaped by a trailing backslash,
// so a logical entry continues while its line ends in an odd number of
// backslashes.
type zshSource struct{}

func (zshSource) Shell() Shell { return ShellZsh }

func (zshSource) DefaultPath() (string, bool) {
	if h := os.Getenv("HISTFILE"); h != "" {
		return expandHome(h), true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".zsh_history"), true
}

func (zshSource) Parse(data []byte) ([]Raw, error) {
	lines := splitLines(unmetafy(data))
	var out []Raw

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}

		ts, cmd, ok := parseZshExtended(line)
		if !ok {
			// Plain format: the whole line is the command.
			cmd = line
		}

		// Join continuation lines (command ends in an odd number of '\').
		for endsWithOddBackslash(cmd) && i+1 < len(lines) {
			i++
			cmd = strings.TrimSuffix(cmd, "\\") + "\n" + lines[i]
		}

		out = append(out, Raw{
			Command:   cmd,
			Timestamp: ts,
			Shell:     ShellZsh,
		})
	}
	return out, nil
}

// parseZshExtended parses a single EXTENDED_HISTORY header line. It returns the
// timestamp, the command portion, and whether the line matched the format.
func parseZshExtended(line string) (time.Time, string, bool) {
	if !strings.HasPrefix(line, ": ") {
		return time.Time{}, "", false
	}
	semi := strings.IndexByte(line, ';')
	if semi < 0 {
		return time.Time{}, "", false
	}
	meta := line[2:semi] // "<epoch>:<elapsed>"
	colon := strings.IndexByte(meta, ':')
	if colon < 0 {
		return time.Time{}, "", false
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(meta[:colon]), 10, 64)
	if err != nil {
		return time.Time{}, "", false
	}
	return time.Unix(epoch, 0), line[semi+1:], true
}

// endsWithOddBackslash reports whether s ends with an odd number of backslash
// characters, which in zsh history marks an escaped (continued) newline.
func endsWithOddBackslash(s string) bool {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}

// splitLines splits file contents into lines, tolerating both "\n" and
// "\r\n" line endings and a missing trailing newline.
func splitLines(data []byte) []string {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// zshMeta is the byte zsh uses to escape "special" bytes when writing history,
// so the file stays 8-bit clean. The escaped byte follows, XOR-ed with 0x20.
const zshMeta = 0x83

// unmetafy reverses zsh's history metafication: each zshMeta byte is dropped
// and the following byte is XOR-ed with 0x20, restoring the original byte (for
// example a UTF-8 sequence in a command with non-ASCII characters). Input that
// contains no meta bytes is returned unchanged, so this is safe to run on plain
// or already-decoded history. A trailing lone meta byte is left as-is.
func unmetafy(data []byte) []byte {
	if bytes.IndexByte(data, zshMeta) < 0 {
		return data
	}
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == zshMeta && i+1 < len(data) {
			out = append(out, data[i+1]^0x20)
			i++
			continue
		}
		out = append(out, data[i])
	}
	return out
}
