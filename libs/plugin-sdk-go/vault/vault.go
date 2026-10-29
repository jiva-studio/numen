// Package vault checks that a path supplied by a caller lies inside the vault.
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

// ValidatePath returns the clean path inside the vault named by NUMEN_VAULT_PATH, as
// ValidatePathIn returns it inside that vault.
func ValidatePath(path string, allow ...func(cleanPath string) bool) (string, error) {
	return ValidatePathIn(os.Getenv("NUMEN_VAULT_PATH"), path, allow...)
}

// ValidatePathIn returns the clean path inside the vault at vaultRoot. A relative
// path is joined to the vault root. An empty vault root rejects every path. A
// clean path for which an allow function returns true is returned without the
// vault check.
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

	cleanVault, cleanTarget, err := resolveTarget(vaultRoot, cleanPath)
	if err != nil {
		return "", err
	}

	if err := checkRelativeContainment(cleanVault, cleanTarget, path); err != nil {
		return "", err
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

func resolveTarget(vaultRoot, cleanPath string) (string, string, error) {
	cleanVault, err := filepath.Abs(filepath.Clean(vaultRoot))
	if err != nil {
		return "", "", err
	}
	cleanTarget := cleanPath
	if !filepath.IsAbs(cleanPath) {
		cleanTarget = filepath.Join(cleanVault, cleanPath)
	}
	cleanTarget, err = filepath.Abs(cleanTarget)
	if err != nil {
		return "", "", err
	}
	return cleanVault, cleanTarget, nil
}

func checkRelativeContainment(cleanVault, cleanTarget, rawPath string) error {
	rel, err := filepath.Rel(cleanVault, cleanTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		if !filepath.IsAbs(rawPath) || strings.Contains(rawPath, "..") {
			return ErrPathTraversal
		}
		return ErrOutsideVault
	}
	return nil
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

	checkPath, err := resolveSymlinkTarget(cleanTarget)
	if err != nil {
		return err
	}

	realParent, errP := findExistingAncestor(filepath.Dir(checkPath))
	if errP != nil {
		return errP
	}

	rel, errRel := filepath.Rel(realVault, realParent)
	if errRel != nil || (rel != "." && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
		return ErrPathTraversal
	}
	return nil
}

func resolveSymlinkTarget(cleanTarget string) (string, error) {
	fi, err := os.Lstat(cleanTarget)
	if err == nil {
		if fi.Mode()&os.ModeSymlink == 0 {
			return cleanTarget, nil
		}
		dest, errR := os.Readlink(cleanTarget)
		if errR != nil {
			return "", ErrPathTraversal
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(cleanTarget), dest)
		}
		return filepath.Clean(dest), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return cleanTarget, nil
}

func findExistingAncestor(start string) (string, error) {
	current := start
	for {
		rp, err := filepath.EvalSymlinks(current)
		if err == nil {
			return rp, nil
		}
		next := filepath.Dir(current)
		if next == current {
			return "", ErrPathTraversal
		}
		current = next
	}
}
