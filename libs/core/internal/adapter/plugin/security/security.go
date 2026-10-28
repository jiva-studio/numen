package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var (
	errEmptyPath         = errors.New("path cannot be empty")
	ErrEmptyVaultRoot    = errors.New("vault root cannot be empty")
	ErrProhibitedTempDir = errors.New("executing plugin binaries from temporary directory is prohibited")
	errEntrypointEscape  = errors.New("plugin entrypoint escapes plugin directory")
	ErrEmptyEntrypoint   = errors.New("plugin entrypoint is empty")
)

// VaultPathValidator verifies that file paths accessed by plugins reside within the active vault.
type VaultPathValidator struct{}

// NewVaultPathValidator creates a new path validator.
func NewVaultPathValidator() VaultPathValidator {
	return VaultPathValidator{}
}

// ValidatePath checks that targetPath is located within vaultRoot and contains no traversal outside it.
func (v VaultPathValidator) ValidatePath(vaultRoot, targetPath string) (string, error) {
	if strings.TrimSpace(targetPath) == "" {
		return "", errEmptyPath
	}
	if strings.TrimSpace(vaultRoot) == "" {
		return "", ErrEmptyVaultRoot
	}

	cleanVault, err := filepath.Abs(filepath.Clean(vaultRoot))
	if err != nil {
		return "", fmt.Errorf("resolve vault root: %w", err)
	}

	var cleanTarget string
	if filepath.IsAbs(targetPath) {
		cleanTarget = filepath.Clean(targetPath)
	} else {
		cleanTarget = filepath.Join(cleanVault, targetPath)
	}

	cleanTarget, err = filepath.Abs(cleanTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}

	rel, err := filepath.Rel(cleanVault, cleanTarget)
	if err != nil {
		return "", pluginport.ErrPathTraversal
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", pluginport.ErrPathTraversal
	}

	realVault, errV := filepath.EvalSymlinks(cleanVault)
	if errV != nil {
		return "", fmt.Errorf("resolve vault root symlinks: %w", errV)
	}

	if err := checkVaultSymlinks(realVault, cleanTarget); err != nil {
		return "", err
	}

	return cleanTarget, nil
}

func checkVaultSymlinks(realVault, cleanTarget string) error {
	realTarget, err := filepath.EvalSymlinks(cleanTarget)
	if err == nil {
		return checkPathInside(realVault, realTarget)
	}

	checkPath, err := resolveSymlinkTarget(cleanTarget)
	if err != nil {
		return pluginport.ErrPathTraversal
	}

	realParent, err := resolveDeepestParent(checkPath)
	if err != nil {
		return pluginport.ErrPathTraversal
	}

	return checkParentInside(realVault, realParent)
}

func checkPathInside(realVault, target string) error {
	relReal, errRel := filepath.Rel(realVault, target)
	if errRel != nil || relReal == ".." || strings.HasPrefix(relReal, ".."+string(filepath.Separator)) {
		return pluginport.ErrPathTraversal
	}
	return nil
}

func checkParentInside(realVault, parent string) error {
	relReal, errRel := filepath.Rel(realVault, parent)
	if errRel != nil || (relReal != "." && (relReal == ".." || strings.HasPrefix(relReal, ".."+string(filepath.Separator)))) {
		return pluginport.ErrPathTraversal
	}
	return nil
}

func resolveSymlinkTarget(cleanTarget string) (string, error) {
	if fi, err := os.Lstat(cleanTarget); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		dest, err := os.Readlink(cleanTarget)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(cleanTarget), dest)
		}
		return filepath.Clean(dest), nil
	}
	return cleanTarget, nil
}

func resolveDeepestParent(path string) (string, error) {
	pos := filepath.Dir(path)
	for {
		rp, err := filepath.EvalSymlinks(pos)
		if err == nil {
			return rp, nil
		}
		next := filepath.Dir(pos)
		if next == pos {
			return "", err
		}
		pos = next
	}
}

// IsSystemTempDir reports whether the given path resides within an operating system temporary directory.
func IsSystemTempDir(path string) bool {
	clean, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	eval, _ := filepath.EvalSymlinks(clean)

	for _, tempDir := range getSystemTempDirs() {
		relClean, errClean := filepath.Rel(tempDir, clean)
		if errClean == nil && relClean != ".." && !strings.HasPrefix(relClean, ".."+string(filepath.Separator)) {
			return true
		}
		if eval != "" {
			relEval, errEval := filepath.Rel(tempDir, eval)
			if errEval == nil && relEval != ".." && !strings.HasPrefix(relEval, ".."+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}

func getSystemTempDirs() []string {
	observed := make(map[string]struct{})
	var dirs []string
	record := func(p string) {
		if strings.TrimSpace(p) == "" {
			return
		}
		clean, err := filepath.Abs(filepath.Clean(p))
		if err == nil {
			if _, ok := observed[clean]; !ok {
				observed[clean] = struct{}{}
				dirs = append(dirs, clean)
			}
		}
		if eval, err := filepath.EvalSymlinks(clean); err == nil {
			if _, ok := observed[eval]; !ok {
				observed[eval] = struct{}{}
				dirs = append(dirs, eval)
			}
		}
	}

	record(os.TempDir())
	record(os.Getenv("TEMP"))
	record(os.Getenv("TMP"))
	if runtime.GOOS != "windows" {
		record("/tmp")
		record("/var/tmp")
	}
	return dirs
}

// ValidateExecutablePath ensures an entrypoint stays within the plugin directory and outside temporary locations.
func ValidateExecutablePath(pluginDir, entrypoint string) (string, error) {
	if strings.TrimSpace(entrypoint) == "" {
		return "", ErrEmptyEntrypoint
	}

	cleanDir, err := filepath.Abs(filepath.Clean(pluginDir))
	if err != nil {
		return "", fmt.Errorf("resolve plugin directory: %w", err)
	}

	binPath := filepath.Join(cleanDir, entrypoint)
	cleanBin, err := filepath.Abs(filepath.Clean(binPath))
	if err != nil {
		return "", fmt.Errorf("resolve binary path: %w", err)
	}

	rel, err := filepath.Rel(cleanDir, cleanBin)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errEntrypointEscape
	}

	if IsSystemTempDir(cleanBin) || IsSystemTempDir(cleanDir) {
		return "", ErrProhibitedTempDir
	}

	realDir := cleanDir
	if evalDir, err := filepath.EvalSymlinks(cleanDir); err == nil {
		realDir = evalDir
	}

	return validateExecutableSymlink(realDir, cleanBin)
}

func validateExecutableSymlink(realDir, cleanBin string) (string, error) {
	realBin, err := filepath.EvalSymlinks(cleanBin)
	if err == nil {
		if IsSystemTempDir(realBin) {
			return "", ErrProhibitedTempDir
		}
		relReal, errRel := filepath.Rel(realDir, realBin)
		if errRel != nil || relReal == ".." || strings.HasPrefix(relReal, ".."+string(filepath.Separator)) {
			return "", errEntrypointEscape
		}
		return realBin, nil
	}

	fi, lerr := os.Lstat(cleanBin)
	if lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
		return validateBrokenExecutableSymlink(realDir, cleanBin, err)
	}

	return cleanBin, nil
}

func validateBrokenExecutableSymlink(realDir, cleanBin string, evalErr error) (string, error) {
	target, rerr := os.Readlink(cleanBin)
	if rerr != nil {
		return "", fmt.Errorf("resolve entrypoint symlink: %w", evalErr)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(cleanBin), target)
	}
	cleanTarget, _ := filepath.Abs(filepath.Clean(target))
	if IsSystemTempDir(cleanTarget) {
		return "", ErrProhibitedTempDir
	}
	relTarget, errRel := filepath.Rel(realDir, cleanTarget)
	if errRel != nil || relTarget == ".." || strings.HasPrefix(relTarget, ".."+string(filepath.Separator)) {
		return "", errEntrypointEscape
	}
	return "", fmt.Errorf("resolve entrypoint symlink: %w", evalErr)
}
