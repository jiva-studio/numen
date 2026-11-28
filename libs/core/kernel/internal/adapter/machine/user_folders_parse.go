package machine

import (
	"path/filepath"
	"strings"
)

const documentsKey = "XDG_DOCUMENTS_DIR"

func parseDocumentsFolder(content, home string) (string, bool) {
	for line := range strings.SplitSeq(content, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found || strings.TrimSpace(key) != documentsKey {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if rest, isHome := strings.CutPrefix(value, "$HOME"); isHome {
			value = home + rest
		}
		if !filepath.IsAbs(value) {
			return "", false
		}
		return filepath.Clean(value), true
	}
	return "", false
}
