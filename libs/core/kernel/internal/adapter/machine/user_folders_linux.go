//go:build linux

package machine

import (
	"os"
	"path/filepath"
)

func readPlatformDocuments(home string) string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(config) {
		config = filepath.Join(home, ".config")
	}
	content, err := os.ReadFile(filepath.Join(config, "user-dirs.dirs"))
	if err == nil {
		if documents, isFound := parseDocumentsFolder(string(content), home); isFound {
			return documents
		}
	}
	return filepath.Join(home, "Documents")
}
