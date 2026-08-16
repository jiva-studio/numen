package filesystem

import (
	"errors"
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strings"
)

// ErrOutside is what a path that does not stay in the vault gets. It is not
// "not found": a caller asking for something outside has made a mistake worth
// hearing about, and answering "no such note" would hide it.
var ErrOutside = errors.New("not a path inside the vault")

// ErrNotANote is what a path inside the vault gets when the vault does not hold
// it as a note: an attachment, a folder some tool keeps its state in, whatever
// the vault's own rules say to leave alone.
var ErrNotANote = errors.New("not a note this vault holds")

// inside turns a path from a vault root into a path on this machine, and
// refuses one that leaves.
//
// The check is on the cleaned path. `notes/../../etc` begins with neither a
// slash nor a dot-dot and reaches outside all the same. Everything that
// reaches a vault from outside the application arrives here first.
func inside(root, path, serviceDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%w: it is empty", ErrOutside)
	}
	if filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	if clean == serviceDir || strings.HasPrefix(clean, serviceDir+"/") {
		return "", fmt.Errorf("%s belongs to the application, not to the vault", path)
	}

	target := filepath.Join(root, filepath.FromSlash(clean))

	// A folder inside the vault may be a link to somewhere else — a synced
	// folder, a shared one — and a rename or a write through it lands outside.
	// The text says nothing about that; only the filesystem knows, so it is
	// asked about the deepest part of the path that exists.
	real, err := deepest(target)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	return target, nil
}

// deepest resolves as much of a path as exists, so that a file about to be
// created is judged by the folder it would land in.
func deepest(target string) (string, error) {
	at := target
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(at)
		if err == nil {
			return filepath.Join(append([]string{resolved}, missing...)...), nil
		}
		parent := filepath.Dir(at)
		if parent == at {
			return "", err
		}
		missing = append([]string{filepath.Base(at)}, missing...)
		at = parent
	}
}
