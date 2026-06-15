// Package recipe is yore's curated command knowledge: named, parameterized,
// reusable commands that live in human-readable, git-committable files.
//
// A recipe captures a command worth keeping — the hard-won incantation you'd
// otherwise re-derive — with the variable parts written as {placeholders} so it
// can be reused. Recipes live in two places:
//
//   - a personal file at ~/.config/yore/recipes
//   - a project file named .yorefile at a repository's root (commit it to share
//     the project's runbook with your team)
//
// The file format is intentionally tiny and dependency-free:
//
//	# a comment
//	[deploy staging]
//	desc: build and ship to the staging cluster
//	cmd: ./deploy.sh --env {env} --tag {tag}
//	param: env=staging
//	tags: deploy, k8s
package recipe

import (
	"regexp"
	"strings"
)

// Param is a single substitutable placeholder in a recipe command.
type Param struct {
	// Name is the placeholder identifier (the foo in {foo}).
	Name string
	// Default is the value offered when the user is prompted ("" if none).
	Default string
}

// Recipe is a named, parameterized command.
type Recipe struct {
	Name        string
	Command     string
	Description string
	Params      []Param
	Tags        []string
	// Source notes where the recipe came from ("project" or "global"); set by
	// the loader, not the parser.
	Source string
}

// placeholderRE matches {name} placeholders with identifier-like names.
var placeholderRE = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// ParamsIn returns the placeholder names used in command, in first-appearance
// order, without duplicates.
func ParamsIn(command string) []string {
	var names []string
	seen := map[string]bool{}
	for _, m := range placeholderRE.FindAllStringSubmatch(command, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	return names
}

// resolveParams builds the ordered Param list for a command, attaching declared
// defaults from the given map.
func resolveParams(command string, defaults map[string]string) []Param {
	names := ParamsIn(command)
	params := make([]Param, 0, len(names))
	for _, n := range names {
		params = append(params, Param{Name: n, Default: defaults[n]})
	}
	return params
}

// Render substitutes the given values into the command. Placeholders without a
// supplied value are left untouched.
func (r Recipe) Render(values map[string]string) string {
	return placeholderRE.ReplaceAllStringFunc(r.Command, func(tok string) string {
		name := tok[1 : len(tok)-1]
		if v, ok := values[name]; ok {
			return v
		}
		return tok
	})
}

// MatchesQuery reports whether the recipe matches a free-text query against its
// name, description, command and tags (case-insensitive substring).
func (r Recipe) MatchesQuery(q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	hay := strings.ToLower(r.Name + " " + r.Description + " " + r.Command + " " + strings.Join(r.Tags, " "))
	return strings.Contains(hay, q)
}
