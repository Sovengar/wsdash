package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

func Scan(roots []string, depth int, exclude []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, r := range roots {
		r = expand(r)
		filepath.WalkDir(r, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			// depth check relative
			rel, _ := filepath.Rel(r, path)
			if rel == "." {
				return nil
			}
			depthLevel := strings.Count(rel, string(os.PathSeparator)) + 1
			if depthLevel > depth {
				return filepath.SkipDir
			}
			base := d.Name()
			for _, ex := range exclude {
				if base == ex {
					return filepath.SkipDir
				}
			}
			if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
				if !seen[path] {
					seen[path] = true
					out = append(out, path)
				}
				return filepath.SkipDir
			}
			return nil
		})
	}
	return out
}

func expand(s string) string {
	if strings.HasPrefix(s, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, s[2:])
		}
	}
	return s
}

// Name is the project name derived from a repository path.
func Name(path string) string {
	return filepath.Base(path)
}
