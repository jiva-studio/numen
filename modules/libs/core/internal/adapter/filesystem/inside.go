package filesystem

import (
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
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
// not in the folder belonging to the application — neither as it is spelled nor
// where it lands.
//
// A link is a second spelling for a place, so the folder is asked about both.
// `link/ocr/abc.txt`, where `link` is a link to the service folder, reads as the
// vault's and is the application's, and it is the second that decides.
func within(root, path, serviceDir string) (target, real string, err error) {
	clean, err := cleaned(path)
	if err != nil {
		return "", "", err
	}
	if ours(clean, serviceDir) {
		return "", "", fmt.Errorf("%s belongs to the application, not to the vault", path)
	}
	target, real, landed, err := contained(root, clean)
	if err != nil {
		return "", "", err
	}
	if ours(landed, serviceDir) {
		return "", "", fmt.Errorf("%s belongs to the application, not to the vault", path)
	}
	return target, real, nil
}

// service is the complement of within: a path that stays inside the vault, and
// that is in the folder belonging to the application by both of the same
// measures.
//
// The two overlap nowhere, which is what lets one type write a derived file
// where no writer of notes can reach, without making any note's refusal weaker.
// That property is the subject of a test. A path whose spelling and whose
// landing disagree is neither's, and both refuse it.
func service(root, path, serviceDir string) (target, real string, err error) {
	clean, err := cleaned(path)
	if err != nil {
		return "", "", err
	}
	if !ours(clean, serviceDir) {
		return "", "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	target, real, landed, err := contained(root, clean)
	if err != nil {
		return "", "", err
	}
	if !ours(landed, serviceDir) {
		return "", "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	return target, real, nil
}

// ours says whether a path is in the folder the application keeps for itself.
//
// The name is compared without regard to case, which is how macOS and Windows
// open it.
func ours(clean, serviceDir string) bool {
	if len(clean) < len(serviceDir) || !strings.EqualFold(clean[:len(serviceDir)], serviceDir) {
		return false
	}
	return len(clean) == len(serviceDir) || clean[len(serviceDir)] == '/'
}

// cleaned is a path a vault could hold, in the one form the rules are written
// against. A path that could not name anything inside a vault is refused here
// and never reaches the filesystem.
func cleaned(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%w: it is empty", ErrOutside)
	}
	// filepath.IsLocal refuses a path that is absolute or that climbs out, and
	// on Windows one that is rooted, drive-relative, or names a reserved
	// device.
	if !filepath.IsLocal(path) || strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if clean == "." {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	return clean, nil
}

// contained is the containment rule and nothing else: where a cleaned path
// lands on this machine, where it lands once every link on the way to it is
// resolved, and that landing named from the root, all three of them under the
// root or none of them anything.
func contained(root, clean string) (target, real, landed string, err error) {
	target = filepath.Join(root, filepath.FromSlash(clean))

	// A folder inside the vault may be a link to somewhere else — a synced
	// folder, a shared one — and a rename or a write through it lands outside.
	// The text says nothing about that; only the filesystem knows, so it is
	// asked about the deepest part of the path that exists.
	real, err = deepest(target)
	if err != nil {
		return "", "", "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", "", err
	}
	if !under(real, root) {
		return "", "", "", fmt.Errorf("%s: %w", clean, ErrOutside)
	}
	return target, real, landing(real, root), nil
}

// landing is a resolved path as a name under the root, in the form the rules
// are written against. The root itself lands nowhere and is named by nothing.
func landing(real, root string) string {
	if len(real) <= len(root) {
		return ""
	}
	return filepath.ToSlash(real[len(root)+1:])
}

// under says whether a resolved path is a root or lies inside it.
func under(real, root string) bool {
	return real == root || strings.HasPrefix(real, root+string(filepath.Separator))
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
