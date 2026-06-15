package recipe

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Parse reads recipes from a .yorefile-format stream. Unknown keys are ignored;
// a block without a cmd is skipped. Parsing is lenient by design: a malformed
// recipe should never make the rest of the file unusable.
func Parse(r io.Reader) ([]Recipe, error) {
	var (
		recipes  []Recipe
		cur      *Recipe
		defaults map[string]string
	)
	flush := func() {
		if cur != nil && cur.Command != "" {
			cur.Params = resolveParams(cur.Command, defaults)
			recipes = append(recipes, *cur)
		}
		cur = nil
		defaults = nil
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			name := strings.TrimSpace(line[1 : len(line)-1])
			cur = &Recipe{Name: name}
			defaults = map[string]string{}
			continue
		}
		if cur == nil {
			continue // stray line before any [name] header
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "cmd", "command":
			cur.Command = val
		case "desc", "description":
			cur.Description = val
		case "tags":
			cur.Tags = splitTags(val)
		case "param":
			if n, d, ok := strings.Cut(val, "="); ok {
				defaults[strings.TrimSpace(n)] = strings.TrimSpace(d)
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return recipes, err
	}
	return recipes, nil
}

func splitTags(s string) []string {
	var tags []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// Format renders a single recipe back into .yorefile syntax (used by `yore
// save`). It emits declared param defaults so they round-trip.
func Format(r Recipe) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s]\n", r.Name)
	if r.Description != "" {
		fmt.Fprintf(&b, "desc: %s\n", r.Description)
	}
	fmt.Fprintf(&b, "cmd: %s\n", r.Command)
	// Emit params that carry a default, in a stable order.
	withDefaults := make([]Param, 0, len(r.Params))
	for _, p := range r.Params {
		if p.Default != "" {
			withDefaults = append(withDefaults, p)
		}
	}
	sort.Slice(withDefaults, func(i, j int) bool { return withDefaults[i].Name < withDefaults[j].Name })
	for _, p := range withDefaults {
		fmt.Fprintf(&b, "param: %s=%s\n", p.Name, p.Default)
	}
	if len(r.Tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(r.Tags, ", "))
	}
	return b.String()
}
