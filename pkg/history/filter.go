package history

import (
	"regexp"
	"strings"
)

// secretPattern matches commands that very likely embed a credential. When
// redaction is enabled, matching commands are dropped entirely so a secret can
// never be surfaced to stdout, a pager or fzf. The patterns are intentionally
// conservative: a false negative merely shows a command the user already typed,
// whereas an over-broad pattern would silently hide useful history.
var secretPattern = regexp.MustCompile(
	`(?i)(` +
		`--?(?:password|passwd|pwd|token|secret|api[-_]?key|access[-_]?key|auth)[ =]\S` + // flags
		`|(?:password|passwd|token|secret|api[_-]?key|access[_-]?key|aws_secret_access_key)=\S` + // KEY=value
		`|\bAKIA[0-9A-Z]{16}\b` + // AWS access key id
		`|\bghp_[0-9A-Za-z]{20,}\b` + // GitHub personal access token
		`|\bxox[baprs]-[0-9A-Za-z-]{10,}\b` + // Slack token
		`)`,
)

// filterRaw cleans a stream of raw occurrences: it trims surrounding
// whitespace, drops blank and trivially short commands, and (when redact is
// set) drops commands that appear to contain secrets. Order is preserved.
func filterRaw(raws []Raw, redact bool) []Raw {
	out := raws[:0]
	for _, r := range raws {
		r.Command = strings.TrimSpace(r.Command)
		if !keepCommand(r.Command, redact) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// keepCommand reports whether a cleaned command should be retained.
func keepCommand(cmd string, redact bool) bool {
	// Drop empties and single-character noise (e.g. stray "q", "c").
	if len([]rune(cmd)) < 2 {
		return false
	}
	if redact && secretPattern.MatchString(cmd) {
		return false
	}
	return true
}

// LooksLikeSecret reports whether a command appears to contain a credential
// (a password/token/key flag, a KEY=secret assignment, or a recognisable token
// such as an AWS key id). It is deliberately conservative. Callers use it to
// keep secrets out of recorded history and output.
func LooksLikeSecret(command string) bool {
	return secretPattern.MatchString(command)
}
