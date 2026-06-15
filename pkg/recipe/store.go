package recipe

import (
	"os"
	"path/filepath"
)

// ProjectFile is the name of the per-project recipe file. Commit it to share a
// repository's runbook with your team.
const ProjectFile = ".yorefile"

// GlobalPath returns the personal recipe file path, honouring XDG_CONFIG_HOME
// ($XDG_CONFIG_HOME/yore/recipes, falling back to ~/.config/yore/recipes).
func GlobalPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "yore", "recipes"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "yore", "recipes"), nil
}

// FindProjectFile walks up from start looking for a ProjectFile, returning its
// path and true if found.
func FindProjectFile(start string) (string, bool) {
	d := start
	for {
		p := filepath.Join(d, ProjectFile)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", false
		}
		d = parent
	}
}

// LoadFile parses recipes from a file, tagging each with source. A missing file
// is not an error: it returns nil.
func LoadFile(path, source string) ([]Recipe, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	recipes, err := Parse(f)
	if err != nil {
		return nil, err
	}
	for i := range recipes {
		recipes[i].Source = source
	}
	return recipes, nil
}

// Merge combines global and project recipes into one ordered list. Project
// recipes come first and shadow global recipes with the same name, so a team
// runbook overrides personal defaults.
func Merge(global, project []Recipe) []Recipe {
	out := make([]Recipe, 0, len(global)+len(project))
	seen := map[string]bool{}
	for _, r := range project {
		if !seen[r.Name] {
			seen[r.Name] = true
			out = append(out, r)
		}
	}
	for _, r := range global {
		if !seen[r.Name] {
			seen[r.Name] = true
			out = append(out, r)
		}
	}
	return out
}

// Find returns the recipe with the given name, or false.
func Find(recipes []Recipe, name string) (Recipe, bool) {
	for _, r := range recipes {
		if r.Name == name {
			return r, true
		}
	}
	return Recipe{}, false
}

// AppendToFile appends a recipe block to the file at path, creating the file and
// its parent directory if needed. A blank line separates recipes.
func AppendToFile(path string, r Recipe) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	block := "\n" + Format(r)
	_, err = f.WriteString(block)
	return err
}
