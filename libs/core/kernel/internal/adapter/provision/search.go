package provision

import (
	"os"
	"path/filepath"
)

func findInSearch(patterns []string, getenv func(string) string) string {
	for _, pattern := range patterns {
		expanded, ok := expandPattern(pattern, getenv)
		if !ok || !filepath.IsAbs(expanded) {
			continue
		}
		matches, err := filepath.Glob(expanded)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
				return match
			}
		}
	}
	return ""
}

func expandPattern(pattern string, getenv func(string) string) (string, bool) {
	isSet := true
	expanded := os.Expand(pattern, func(name string) string {
		value := getenv(name)
		if value == "" {
			isSet = false
		}
		return value
	})
	return expanded, isSet
}
