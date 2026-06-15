package cli

import (
	"os"
	"path/filepath"
)

// projectMarkers identify a project root when walking up the directory tree.
var projectMarkers = []string{".yorefile", ".git", "go.mod", "package.json", ".hg", "Cargo.toml"}

// projectRoot returns the nearest ancestor of dir (inclusive) that contains a
// project marker, or dir itself if none is found. This is what scopes
// `yore here` to "this project" rather than the exact directory.
func projectRoot(dir string) string {
	d := dir
	for {
		for _, m := range projectMarkers {
			if _, err := os.Stat(filepath.Join(d, m)); err == nil {
				return d
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir // reached filesystem root with no marker
		}
		d = parent
	}
}

// currentDir returns the absolute current working directory, or "" if it cannot
// be determined.
func currentDir() string {
	d, err := os.Getwd()
	if err != nil {
		return ""
	}
	return d
}
