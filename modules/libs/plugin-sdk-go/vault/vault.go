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
	ErrEmptyVault    = errors.New("vault root required")
	ErrPathTraversal = errors.New("path traversal outside allowed scope")
	ErrOutsideVault  = errors.New("path outside allowed vault root")
)

// tempDocumentPrefix starts the name of a document the host puts in the temp folder.
const tempDocumentPrefix = "numen-doc-"

// IsTempDocument reports whether a clean path is a regular document file directly
// in the system temp directory.
func IsTempDocument(cleanPath string) bool {
	tempDir := filepath.Clean(os.TempDir())
	realTemp, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		realTemp = tempDir
	}

	dir := filepath.Dir(cleanPath)
	if dir != tempDir && dir != realTemp {
		return false
	}
	if !strings.HasPrefix(filepath.Base(cleanPath), tempDocumentPrefix) {
		return false
	}

	if fi, err := os.Lstat(cleanPath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return false
	}

	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err == nil {
		rel, err := filepath.Rel(realTemp, realPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false
		}
	}
	return true
}

// ValidatePath returns the clean path inside the vault named by NUMEN_VAULT_PATH, as
// ValidatePathIn returns it inside that vault.
func ValidatePath(path string, allow ...func(cleanPath string) bool) (string, error) {
	return ValidatePathIn(os.Getenv("NUMEN_VAULT_PATH"), path, allow...)
}

// ValidatePathIn returns the clean path inside the vault at vaultRoot. A relative path is joined
// to the vault root. An empty vault root refuses every path. A clean path for which an allow
// function returns true is returned without the vault check.
func ValidatePathIn(vaultRoot, path string, allow ...func(cleanPath string) bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", ErrEmptyPath
	}
	cleanPath := filepath.Clean(path)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}
	for _, isAllowed := range allow {
		if isAllowed(cleanPath) {
			return cleanPath, nil
		}
	}

	if strings.TrimSpace(vaultRoot) == "" {
		return "", ErrEmptyVault
	}

	cleanVault, err := filepath.Abs(filepath.Clean(vaultRoot))
	if err != nil {
		return "", err
	}

	var cleanTarget string
	if filepath.IsAbs(cleanPath) {
		cleanTarget = cleanPath
	} else {
		cleanTarget = filepath.Join(cleanVault, cleanPath)
	}
	cleanTarget, err = filepath.Abs(cleanTarget)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(cleanVault, cleanTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		if !filepath.IsAbs(path) || strings.Contains(path, "..") {
			return "", ErrPathTraversal
		}
		return "", ErrOutsideVault
	}

	realVault, errV := filepath.EvalSymlinks(cleanVault)
	if errV != nil {
		return "", errV
	}

	if err := checkSymlinkContainment(realVault, cleanTarget); err != nil {
		return "", err
	}

	return cleanTarget, nil
}

func checkSymlinkContainment(realVault, cleanTarget string) error {
	realTarget, err := filepath.EvalSymlinks(cleanTarget)
	if err == nil {
		rel, errRel := filepath.Rel(realVault, realTarget)
		if errRel != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return ErrPathTraversal
		}
		return nil
	}

	checkPath := cleanTarget
	if fi, errL := os.Lstat(cleanTarget); errL == nil && fi.Mode()&os.ModeSymlink != 0 {
		dest, errR := os.Readlink(cleanTarget)
		if errR != nil {
			return ErrPathTraversal
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(cleanTarget), dest)
		}
		checkPath = filepath.Clean(dest)
	}

	parent := filepath.Dir(checkPath)
	realParent, errP := filepath.EvalSymlinks(parent)
	if errP != nil {
		at := parent
		for {
			rp, err := filepath.EvalSymlinks(at)
			if err == nil {
				realParent = rp
				break
			}
			next := filepath.Dir(at)
			if next == at {
				return ErrPathTraversal
			}
			at = next
		}
	}

	rel, errRel := filepath.Rel(realVault, realParent)
	if errRel != nil || (rel != "." && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
		return ErrPathTraversal
	}
	return nil
}
