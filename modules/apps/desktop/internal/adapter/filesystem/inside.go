package filesystem

import (
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ErrOutside is the core's sentinel, so a caller that never names this package
// still recognises it.
var ErrOutside = port.ErrOutside

// ErrNotANote is what a path inside the vault gets when the vault does not hold
// it as a note: an attachment, a folder some tool keeps its state in, whatever
// the vault's own rules say to leave alone. It is the core's sentinel: the same
// answer reaches a use case whichever vault it came from.
var ErrNotANote = port.ErrNotANote

// inside turns a path from a vault root into a path on this machine, and
// refuses one that leaves.
//
// The check is on the cleaned path. `notes/../../etc` begins with neither a
// slash nor a dot-dot and reaches outside all the same. Everything that
// reaches a vault from outside the application arrives here first.
func inside(root, path, serviceDir string) (string, error) {
	target, _, err := within(root, path, serviceDir)
	return target, err
}

// followed is where the bytes of a note are: the path with every link on the
// way to it resolved. A write renames over this one, and a note kept as a link
// to another file in the vault is a link afterwards.
func followed(root, path, serviceDir string) (string, error) {
	_, real, err := within(root, path, serviceDir)
	return real, err
}

// within is the vault as the person's: a path that stays inside it, and that is
// not in the folder belonging to the application.
func within(root, path, serviceDir string) (target, real string, err error) {
	clean, err := cleaned(path)
	if err != nil {
		return "", "", err
	}
	if ours(clean, serviceDir) {
		return "", "", fmt.Errorf("%s belongs to the application, not to the vault", path)
	}
	return contained(root, clean)
}

// service is the exact complement of within: a path that stays inside the
// vault, and that is in the folder belonging to the application.
//
// The two together cover the vault once and overlap nowhere, which is what lets
// one type write a derived file where no writer of notes can reach, without
// making any note's refusal weaker. That property is the subject of a test.
func service(root, path, serviceDir string) (target, real string, err error) {
	clean, err := cleaned(path)
	if err != nil {
		return "", "", err
	}
	if !ours(clean, serviceDir) {
		return "", "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	return contained(root, clean)
}

// ours says whether a path is in the folder the application keeps for itself.
func ours(clean, serviceDir string) bool {
	return clean == serviceDir || strings.HasPrefix(clean, serviceDir+"/")
}

// cleaned is a path a vault could hold, in the one form the rules are written
// against. A path that could not name anything inside a vault is refused here
// and never reaches the filesystem.
func cleaned(path string) (string, error) {
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
	return clean, nil
}

// contained is the containment rule and nothing else: where a cleaned path
// lands on this machine, and where it lands once every link on the way to it is
// resolved, both of them under the root or neither of them anything.
func contained(root, clean string) (target, real string, err error) {
	target = filepath.Join(root, filepath.FromSlash(clean))

	// A folder inside the vault may be a link to somewhere else — a synced
	// folder, a shared one — and a rename or a write through it lands outside.
	// The text says nothing about that; only the filesystem knows, so it is
	// asked about the deepest part of the path that exists.
	real, err = deepest(target)
	if err != nil {
		return "", "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", err
	}
	if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
		return "", "", fmt.Errorf("%s: %w", clean, ErrOutside)
	}
	return target, real, nil
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
