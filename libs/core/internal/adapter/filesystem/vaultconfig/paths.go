package vaultconfig

import (
	"fmt"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrOutside is the core's sentinel, so a caller that never names this package
// still recognises it.
var ErrOutside = vaultport.ErrOutside

// ErrNotANote is returned for a path inside the vault that the vault does not hold
// as a note: an attachment, a folder where a tool keeps its state, or anything the
// vault's own rules exclude. It is the core's sentinel, so a use case sees the same
// error from any vault.
var ErrNotANote = vaultport.ErrNotANote

// GetContainedPath converts a path relative to a vault root into a path on this
// machine and rejects one that leaves the root.
//
// The check runs on the cleaned path: `notes/../../etc` starts with neither a slash
// nor a dot-dot yet escapes the root. Every access to a vault from outside the
// application passes through here first.
func GetContainedPath(root, path, serviceDir string) (string, error) {
	target, _, err := resolveVaultPath(root, path, serviceDir)
	return target, err
}

// ResolveLinks is where the bytes of a note are: the path with every link on the
// way to it resolved. A write renames over this one, and a note kept as a link
// to another file in the vault is a link afterwards.
func ResolveLinks(root, path, serviceDir string) (string, error) {
	_, resolved, err := resolveVaultPath(root, path, serviceDir)
	return resolved, err
}

// resolveVaultPath returns the user's vault path: one that stays inside the vault
// and is not in the application's own folder, either as spelled or after resolving
// links.
//
// A link is an alternate spelling of a location, so the folder check applies to
// both. `link/ocr/abc.txt`, where `link` links to the service folder, looks like
// vault content but belongs to the application; the resolved path decides.
func resolveVaultPath(root, path, serviceDir string) (target, resolved string, err error) {
	clean, err := CleanPath(path)
	if err != nil {
		return "", "", err
	}
	if isOurs(clean, serviceDir) {
		return "", "", fmt.Errorf("%s belongs to the application, not to the vault", path)
	}
	target, resolved, landed, err := resolveContained(root, clean)
	if err != nil {
		return "", "", err
	}
	if isOurs(landed, serviceDir) {
		return "", "", fmt.Errorf("%s belongs to the application, not to the vault", path)
	}
	return target, resolved, nil
}

// isOurs reports whether a path is in the folder the application reserves for
// itself.
//
// The name is compared case-insensitively, matching how macOS and Windows open it.
func isOurs(clean, serviceDir string) bool {
	if len(clean) < len(serviceDir) || !strings.EqualFold(clean[:len(serviceDir)], serviceDir) {
		return false
	}
	return len(clean) == len(serviceDir) || clean[len(serviceDir)] == '/'
}

// CleanPath returns a path a vault could hold, in the canonical form the rules use.
// It never accesses the filesystem.
func CleanPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%w: it is empty", ErrOutside)
	}
	// filepath.IsLocal rejects a path that is absolute or that climbs out, and on
	// Windows one that is rooted, drive-relative, or names a reserved device.
	if !filepath.IsLocal(path) || strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if clean == "." || (runtime.GOOS == "windows" && hasReservedDeviceName(clean)) {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	return clean, nil
}

// hasReservedDeviceName reports whether any element of a slash-separated path
// names a Windows device, with or without an extension.
func hasReservedDeviceName(clean string) bool {
	for _, element := range strings.Split(clean, "/") {
		name := strings.TrimRight(element, " .")
		if dot := strings.IndexByte(name, '.'); dot >= 0 {
			name = strings.TrimRight(name[:dot], " ")
		}
		switch strings.ToUpper(name) {
		case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
			return true
		}
		if len(name) == 4 && name[3] >= '1' && name[3] <= '9' && (strings.EqualFold(name[:3], "COM") || strings.EqualFold(name[:3], "LPT")) {
			return true
		}
	}
	return false
}

// resolveContained is the containment rule and nothing else: where a cleaned path
// lands on this machine, where it lands once every link on the way to it is
// resolved, and that landing named from the root, all three of them under the
// root or none of them anything.
func resolveContained(root, clean string) (target, resolved, landed string, err error) {
	target = filepath.Join(root, filepath.FromSlash(clean))

	// A folder inside the vault may be a link to another location, such as a synced or
	// shared folder, and a rename or write through it lands outside the vault. The
	// path text cannot show this; only the filesystem can, so the deepest existing
	// part of the path is resolved.
	resolved, err = GetDeepestExisting(target)
	if err != nil {
		return "", "", "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", "", err
	}
	if !IsUnderRoot(resolved, root) {
		return "", "", "", fmt.Errorf("%s: %w", clean, ErrOutside)
	}
	return target, resolved, getRootRelativeName(resolved, root), nil
}

// getRootRelativeName is a resolved path as a name under the root, in the form the rules
// are written against. The root itself lands nowhere and is named by nothing.
func getRootRelativeName(resolved, root string) string {
	if len(resolved) <= len(root) {
		return ""
	}
	return filepath.ToSlash(resolved[len(root)+1:])
}

// IsUnderRoot reports whether a resolved path is the root or inside it.
func IsUnderRoot(resolved, root string) bool {
	return resolved == root || strings.HasPrefix(resolved, root+string(filepath.Separator))
}

// GetDeepestExisting resolves as much of a path as exists, so a file about to be
// created is judged by the folder it will land in.
func GetDeepestExisting(target string) (string, error) {
	pos := target
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(pos)
		if err == nil {
			return filepath.Join(append([]string{resolved}, missing...)...), nil
		}
		parent := filepath.Dir(pos)
		if parent == pos {
			return "", err
		}
		missing = append([]string{filepath.Base(pos)}, missing...)
		pos = parent
	}
}
