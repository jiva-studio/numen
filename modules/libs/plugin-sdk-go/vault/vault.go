// Package vault checks that a path a caller names lies inside the vault.
package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrEmptyPath     = errors.New("path required")
	ErrPathTraversal = errors.New("path traversal outside allowed scope")
	ErrOutsideVault  = errors.New("path outside allowed vault root")
)

// tempDocumentPrefix starts the name of a document the host puts in the temp folder.
const tempDocumentPrefix = "numen-doc-"

// IsTempDocument says whether a clean path is a document the host put directly
// in the temp folder.
func IsTempDocument(cleanPath string) bool {
	return filepath.Dir(cleanPath) == filepath.Clean(os.TempDir()) &&
		strings.HasPrefix(filepath.Base(cleanPath), tempDocumentPrefix)
}

// ValidatePath returns the clean path inside the vault named by NUMEN_VAULT_PATH.
// A relative path is joined to the vault root, and with no vault root the clean
// path is returned as given. A clean path for which an allow function returns
// true is returned without the vault check.
func ValidatePath(path string, allow ...func(cleanPath string) bool) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return "", ErrPathTraversal
	}
	for _, isAllowed := range allow {
		if isAllowed(cleanPath) {
			return cleanPath, nil
		}
	}

	vaultRoot := os.Getenv("NUMEN_VAULT_PATH")
	if vaultRoot == "" {
		return cleanPath, nil
	}
	cleanVault := filepath.Clean(vaultRoot)
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(cleanVault, cleanPath)
	}
	prefix := cleanVault
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	if cleanPath != cleanVault && !strings.HasPrefix(cleanPath, prefix) {
		return "", ErrOutsideVault
	}
	return cleanPath, nil
}
