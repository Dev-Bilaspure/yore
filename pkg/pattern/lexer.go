package pattern

import (
	"regexp"
	"strings"
)

// token is one lexed shell token with its quoting noted, so rendering can
// re-quote literals that need it.
type token struct {
	text   string
	quoted bool // the source token was quoted (or contained whitespace)
}

// tokenize splits a command line into tokens, honouring quotes and backslash
// escapes. It's a lexer, not a shell parser: operators like | come through as
// ordinary tokens (they just stay literal in any template).
func tokenize(s string) []token {
	var toks []token
	var cur strings.Builder
	started, quoted := false, false
	var inSingle, inDouble bool

	flush := func() {
		if started {
			toks = append(toks, token{text: cur.String(), quoted: quoted})
			cur.Reset()
			started, quoted = false, false
		}
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			started = true
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			started = true
			switch {
			case c == '"':
				inDouble = false
			case c == '\\' && i+1 < len(s):
				i++
				cur.WriteByte(s[i])
			default:
				cur.WriteByte(c)
			}
		case c == '\'':
			started, quoted, inSingle = true, true, true
		case c == '"':
			started, quoted, inDouble = true, true, true
		case c == '\\' && i+1 < len(s):
			i++
			started = true
			cur.WriteByte(s[i])
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
		default:
			started = true
			cur.WriteByte(c)
		}
	}
	flush()
	return toks
}

// flagTok is a parsed flag, optionally with the value that followed it.
type flagTok struct {
	name      string // e.g. "--env" or "-n"
	value     string // the value, when hasValue
	hasValue  bool
	valQuoted bool
	idx       int // index of this flag in the token slice
	valIdx    int // index of the value token (-1 if inline/none)
}

// posTok is a positional (non-flag, non-flag-value) token.
type posTok struct {
	text   string
	quoted bool
	idx    int
}

// parsed is a command decomposed into program, flags and positionals.
type parsed struct {
	prog   string
	tokens []token
	flags  []flagTok
	posn   []posTok
}

// isFlag reports whether a token looks like a flag ("-x", "--long", "--k=v").
func isFlag(t string) bool {
	return len(t) >= 2 && t[0] == '-' && t != "--"
}

// parse decomposes tokens into program, flags and positionals. A bare token
// after a value-less flag is taken as that flag's value; misattribution is
// harmless, since a constant value renders literally either way.
func parse(toks []token) parsed {
	p := parsed{tokens: toks}
	if len(toks) == 0 {
		return p
	}
	p.prog = toks[0].text
	for i := 1; i < len(toks); i++ {
		t := toks[i]
		if isFlag(t.text) {
			f := flagTok{idx: i, valIdx: -1}
			if eq := strings.IndexByte(t.text, '='); eq >= 0 {
				f.name, f.value, f.hasValue = t.text[:eq], t.text[eq+1:], true
			} else {
				f.name = t.text
				if i+1 < len(toks) && !isFlag(toks[i+1].text) {
					f.value, f.hasValue, f.valQuoted = toks[i+1].text, true, toks[i+1].quoted
					f.valIdx = i + 1
					i++
				}
			}
			p.flags = append(p.flags, f)
			continue
		}
		p.posn = append(p.posn, posTok{text: t.text, quoted: t.quoted, idx: i})
	}
	return p
}

// wordRE matches a "fixed word" positional — a subcommand-like token. Anything
// else (paths, numbers, versions, values with punctuation/digits) is treated as
// a value-shaped wildcard, which is what lets value positionals group together.
var wordRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

func isWord(s string) bool { return wordRE.MatchString(s) }

// signature is the structural identity of a command: program + sorted flag
// names + positional shape (literal word, or "*" for a value-shaped slot). Same
// signature == same command with possibly different values == one group. It's
// also the dedup/dismissal key: stable no matter how many values are observed.
func (p parsed) signature() string {
	names := make([]string, 0, len(p.flags))
	seen := map[string]bool{}
	for _, f := range p.flags {
		if !seen[f.name] {
			seen[f.name] = true
			names = append(names, f.name)
		}
	}
	sortStrings(names)

	shape := make([]string, 0, len(p.posn))
	for _, pt := range p.posn {
		if isWord(pt.text) {
			shape = append(shape, pt.text) // subcommands keep groups apart
		} else {
			shape = append(shape, "*") // value-shaped positionals group together
		}
	}

	var b strings.Builder
	b.WriteString(p.prog)
	b.WriteString("\x00F:")
	b.WriteString(strings.Join(names, ","))
	b.WriteString("\x00P:")
	b.WriteString(strings.Join(shape, ","))
	return b.String()
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
