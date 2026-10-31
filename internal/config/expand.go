package config

import (
	"os"
	"path/filepath"
	"strings"
)

func Expand(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			s = filepath.Join(home, s[2:])
		}
	}
	// env expansion
	return os.Expand(s, os.Getenv)
}
