// Package pattern turns a stream of recorded commands into reusable-recipe
// suggestions, deterministically and without any machine learning.
//
// The approach is structural template mining (the Drain/IPLoM family applied to
// shell commands): commands are tokenized and grouped by a structural
// signature, then the slots that vary across a group's occurrences become
// fill-in-the-blank parameters. Crucially, what counts as a "value" is learned
// from observed variation — yore never has to understand a token's meaning — so
// the whole thing is a pure function of the input, reproducible and offline.
package pattern

import (
	"hash/fnv"
	"strconv"
	"strings"
	"time"
)

// Observation is one recorded command run, the only input the engine needs.
type Observation struct {
	Command string
	Dir     string
	Time    time.Time
}

// Param is an inferred fill-in-the-blank, with the distinct values observed for
// it (the evidence that makes the inference legible).
type Param struct {
	Name   string
	Values []string
}

// Candidate is a suggested recipe mined from repeated usage.
type Candidate struct {
	Name     string    // proposed recipe name
	Template string    // command with {blanks}, ready to display/save
	Params   []Param   // the blanks, in template order
	Count    int       // occurrences (within the recency window)
	Dirs     []string  // distinct directories it ran in (capped)
	LastUsed time.Time // most recent occurrence
	Key      string    // stable structural id, for dedup and dismissal
}

// Options tunes detection.
type Options struct {
	MinCount     int           // minimum occurrences to suggest (default 5)
	RecentWithin time.Duration // ignore commands older than this (default 30d; 0 = no limit)
	Now          time.Time     // reference time (default time.Now)
	MaxValues    int           // cap on evidence values shown per param (default 5)
	Limit        int           // cap on candidates returned (0 = all)
}

const (
	defaultMinCount  = 5
	defaultRecent    = 30 * 24 * time.Hour
	defaultMaxValues = 5
	minCommandLen    = 8 // below this, a command isn't worth a recipe
)

// trivialProg lists programs that are never worth a recipe even when frequent.
var trivialProg = map[string]bool{
	"cd": true, "ls": true, "ll": true, "la": true, "cls": true, "clear": true,
	"pwd": true, "exit": true, "z": true, "cat": true, "which": true, "echo": true,
}

// Detect mines suggestions from observations. It is deterministic: the same
// input always yields the same candidates.
func Detect(obs []Observation, opts Options) []Candidate {
	opts = withDefaults(opts)

	// A group of occurrences that share a structural signature.
	type group struct {
		sig     string
		members []parsed
		obs     []Observation
	}
	groups := map[string]*group{}
	var order []string // preserve first-seen group order for determinism

	for _, o := range obs {
		cmd := strings.TrimSpace(o.Command)
		if !opts.Now.IsZero() && opts.RecentWithin > 0 && o.Time.Before(opts.Now.Add(-opts.RecentWithin)) {
			continue
		}
		toks := tokenize(cmd)
		if !worthwhile(cmd, toks) {
			continue
		}
		p := parse(toks)
		sig := p.signature()
		g := groups[sig]
		if g == nil {
			g = &group{sig: sig}
			groups[sig] = g
			order = append(order, sig)
		}
		g.members = append(g.members, p)
		g.obs = append(g.obs, o)
	}

	var out []Candidate
	for _, sig := range order {
		g := groups[sig]
		if len(g.members) < opts.MinCount {
			continue
		}
		if c, ok := buildCandidate(g.members, g.obs, sig, opts); ok {
			out = append(out, c)
		}
	}

	rankCandidates(out)
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out
}

func withDefaults(o Options) Options {
	if o.MinCount <= 0 {
		o.MinCount = defaultMinCount
	}
	if o.RecentWithin == 0 {
		o.RecentWithin = defaultRecent
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	if o.MaxValues <= 0 {
		o.MaxValues = defaultMaxValues
	}
	return o
}

// worthwhile filters out commands not worth turning into a recipe.
func worthwhile(cmd string, toks []token) bool {
	if len(toks) < 2 || len(cmd) < minCommandLen {
		return false
	}
	return !trivialProg[toks[0].text]
}

// buildCandidate turns a group into a Candidate: it decides which slots are
// blanks (by observed variation), names them, and renders the template from the
// most recent member.
func buildCandidate(members []parsed, obs []Observation, sig string, opts Options) (Candidate, bool) {
	// Blank flags: a flag whose value varies across the group.
	flagValues := map[string][]string{} // flag name -> distinct values (ordered)
	flagSeen := map[string]map[string]bool{}
	flagBoolean := map[string]bool{} // appeared at least once without a value
	for _, m := range members {
		for _, f := range m.flags {
			if !f.hasValue {
				flagBoolean[f.name] = true
				continue
			}
			if flagSeen[f.name] == nil {
				flagSeen[f.name] = map[string]bool{}
			}
			if !flagSeen[f.name][f.value] {
				flagSeen[f.name][f.value] = true
				flagValues[f.name] = append(flagValues[f.name], f.value)
			}
		}
	}
	blankFlags := map[string]bool{}
	for name, vals := range flagValues {
		if len(vals) >= 2 && !flagBoolean[name] {
			blankFlags[name] = true
		}
	}

	// Blank positionals: value-shaped slots that vary (indices align across the
	// group, since the signature fixes the positional shape).
	posCount := len(members[0].posn)
	posValues := make([]map[string]bool, posCount)
	posOrdered := make([][]string, posCount)
	for _, m := range members {
		for i, pt := range m.posn {
			if i >= posCount || isWord(pt.text) {
				continue // word positionals are fixed (part of the signature)
			}
			if posValues[i] == nil {
				posValues[i] = map[string]bool{}
			}
			if !posValues[i][pt.text] {
				posValues[i][pt.text] = true
				posOrdered[i] = append(posOrdered[i], pt.text)
			}
		}
	}
	blankPos := map[int]bool{}
	for i := range posOrdered {
		if len(posOrdered[i]) >= 2 {
			blankPos[i] = true
		}
	}

	// Render from the most recent member so the template reflects current usage.
	repIdx := mostRecent(obs)
	rep := members[repIdx]

	tmpl, params := render(rep, blankFlags, blankPos, flagValues, posOrdered, opts.MaxValues)

	// Skip zero-blank candidates that are trivial to type (keeps out noise like
	// "git status"); parameterized ones are always worth offering.
	if len(params) == 0 && len(rep.flags) == 0 && len(rep.tokens) < 3 {
		return Candidate{}, false
	}

	return Candidate{
		Name:     proposeName(rep),
		Template: tmpl,
		Params:   params,
		Count:    len(members),
		Dirs:     distinctDirs(obs),
		LastUsed: obs[repIdx].Time,
		Key:      hashKey(sig),
	}, true
}

// render produces the display template (blanks substituted) and the ordered
// param list with evidence. Names come from flags where possible, else argN.
func render(rep parsed, blankFlags map[string]bool, blankPos map[int]bool,
	flagValues map[string][]string, posValues [][]string, maxValues int) (string, []Param) {

	out := make([]string, len(rep.tokens))
	for i, t := range rep.tokens {
		out[i] = shellQuote(t.text, t.quoted)
	}

	var params []Param
	used := map[string]bool{}
	argN := 0

	for _, f := range rep.flags {
		if !blankFlags[f.name] {
			continue
		}
		name := dedupeName(paramFromFlag(f.name), used)
		if f.valIdx >= 0 { // "--flag value"
			out[f.valIdx] = "{" + name + "}"
		} else { // inline "--flag=value"
			out[f.idx] = f.name + "={" + name + "}"
		}
		params = append(params, Param{Name: name, Values: limitVals(flagValues[f.name], maxValues)})
	}

	posIdx := 0
	for _, pt := range rep.posn {
		if blankPos[posIdx] {
			argN++
			name := dedupeName("arg"+strconv.Itoa(argN), used)
			out[pt.idx] = "{" + name + "}"
			params = append(params, Param{Name: name, Values: limitVals(posValues[posIdx], maxValues)})
		}
		posIdx++
	}

	return strings.Join(out, " "), params
}

// proposeName suggests a recipe name from the program and any subcommand.
func proposeName(p parsed) string {
	base := p.prog
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	for _, ext := range []string{".sh", ".py", ".rb", ".bash", ".zsh"} {
		base = strings.TrimSuffix(base, ext)
	}
	if multiTool[base] && len(p.posn) > 0 && isWord(p.posn[0].text) {
		return base + "-" + p.posn[0].text
	}
	return base
}

var multiTool = map[string]bool{
	"git": true, "kubectl": true, "docker": true, "npm": true, "pnpm": true,
	"yarn": true, "go": true, "cargo": true, "make": true, "terraform": true,
	"helm": true, "brew": true, "gh": true, "aws": true, "gcloud": true,
}

// helpers ------------------------------------------------------------------

func mostRecent(obs []Observation) int {
	best := 0
	for i := 1; i < len(obs); i++ {
		if obs[i].Time.After(obs[best].Time) {
			best = i
		}
	}
	return best
}

func distinctDirs(obs []Observation) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, o := range obs {
		if o.Dir == "" || seen[o.Dir] {
			continue
		}
		seen[o.Dir] = true
		dirs = append(dirs, o.Dir)
		if len(dirs) >= 3 {
			break
		}
	}
	return dirs
}

func paramFromFlag(flag string) string {
	name := strings.TrimLeft(flag, "-")
	if name == "" {
		name = "arg"
	}
	return sanitizeName(name)
}

// sanitizeName makes a string a valid {placeholder} identifier: letters, digits
// and underscores, starting with a letter or underscore.
func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "arg"
	}
	if c := out[0]; c >= '0' && c <= '9' {
		out = "_" + out
	}
	return out
}

func dedupeName(name string, used map[string]bool) string {
	if !used[name] {
		used[name] = true
		return name
	}
	for i := 2; ; i++ {
		cand := name + strconv.Itoa(i)
		if !used[cand] {
			used[cand] = true
			return cand
		}
	}
}

// shellQuote re-quotes a literal token that needs it (empty or contains
// whitespace). Tokens that were quoted in the source but are simple are left
// bare to keep templates readable.
func shellQuote(s string, _ bool) string {
	if s == "" || strings.ContainsAny(s, " \t\n") {
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
	return s
}

func limitVals(vals []string, n int) []string {
	if len(vals) > n {
		return append([]string(nil), vals[:n]...)
	}
	return vals
}

// KeyOf returns the structural key for a single command — the same identity
// Detect assigns to its group — so callers can match a command against existing
// recipes or dismissed suggestions.
func KeyOf(command string) string {
	return hashKey(parse(tokenize(command)).signature())
}

func hashKey(sig string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(sig))
	return strconv.FormatUint(h.Sum64(), 16)
}

// rankCandidates orders by a simple frecency-like score: more occurrences and
// more recent first; ties broken by template for determinism.
func rankCandidates(c []Candidate) {
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && less(c[j], c[j-1]); j-- {
			c[j-1], c[j] = c[j], c[j-1]
		}
	}
}

func less(a, b Candidate) bool {
	if a.Count != b.Count {
		return a.Count > b.Count
	}
	if !a.LastUsed.Equal(b.LastUsed) {
		return a.LastUsed.After(b.LastUsed)
	}
	return a.Template < b.Template
}
