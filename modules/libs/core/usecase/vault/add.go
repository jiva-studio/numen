package vault

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrUnreadable is a path that is not a folder, or a folder this application
// cannot read as the vault it was asked about.
var ErrUnreadable = errors.New("this folder cannot be read as a vault")

// ErrOverlaps is a folder that lies inside a vault on the list, or holds one.
var ErrOverlaps = errors.New("a vault does not lie inside another")

// ErrCopy is one identity carried by two folders that are both there. One index
// cannot hold both.
var ErrCopy = errors.New("one vault cannot be in two places")

// Add turns a folder into a vault this installation knows about.
//
// Two things happen, in this order and no other: the folder is given an
// identity that stays with it, and that identity is remembered here. A folder
// is recognised after it moves by the identity it carries.
//
// A name another vault has gets a number appended, and the person renames it
// afterwards.
type Add struct {
	Identity port.VaultIdentity
	Registry port.VaultRegistry
	Now      func() time.Time
}

// Execute takes a path that is already absolute: resolving one against the
// process working directory is something only the caller knows how to do, and a
// use case whose result depends on where it was invoked from is not one.
func (u Add) Execute(root, name string) (domain.Vault, error) {
	if !filepath.IsAbs(root) {
		return domain.Vault{}, fmt.Errorf("vault path must be absolute, got %q", root)
	}
	// One folder has one name here, whichever route reached it.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if err := u.Identity.Readable(root); err != nil {
		return domain.Vault{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	known, err := u.Registry.All()
	if err != nil {
		return domain.Vault{}, err
	}
	if err := roomFor(root, known); err != nil {
		return domain.Vault{}, err
	}

	id, err := u.Identity.Ensure(root, u.Now())
	if err != nil {
		return domain.Vault{}, fmt.Errorf("give %s an identity: %w", root, err)
	}

	// This identity may already be registered somewhere else, and that is either
	// a move or a copy. A move is ordinary and the whole reason identity does
	// not depend on the path; a copy cannot be indexed, because two folders
	// claiming one identity would write each other's notes under the same rows.
	//
	// They are told apart by looking: if the old location still carries this
	// identity, both exist and it is a copy.
	existing, found, err := u.Registry.Find(id)
	if err != nil {
		return domain.Vault{}, err
	}
	if found && existing.Path != root {
		stillThere, carriesIt, err := u.Identity.Of(existing.Path)
		if err != nil {
			return domain.Vault{}, err
		}
		if carriesIt && stillThere == id {
			return domain.Vault{}, fmt.Errorf(
				"%w: %s and %s both carry the identity %s, so they are copies of one "+
					"vault and cannot both be indexed. To make this a separate "+
					"vault, delete its service folder and add it again",
				ErrCopy, root, existing.Path, id)
		}
		// The old location is gone, or is no longer this vault: the folder
		// moved, and the registry is what has to catch up.
	}

	v := domain.Vault{ID: id, Name: name, Path: root}
	if v.Name == "" {
		// The folder name is what the user already calls this collection.
		v.Name = filepath.Base(root)
	}
	v.Name = free(v.Name, known, id)
	if err := u.Registry.Save(v); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}

// roomFor refuses a root that lies inside a vault already registered, and one
// that holds such a vault.
func roomFor(root string, known []domain.Vault) error {
	for _, other := range known {
		if other.Path == root {
			continue
		}
		if within(root, other.Path) {
			return fmt.Errorf("%w: %s is inside the vault %s at %s",
				ErrOverlaps, root, other.Name, other.Path)
		}
		if within(other.Path, root) {
			return fmt.Errorf("%w: %s holds the vault %s at %s",
				ErrOverlaps, root, other.Name, other.Path)
		}
	}
	return nil
}

// within reports whether path lies below root.
func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// free answers with the name, or with the lowest free number appended to it.
// The vault named self keeps the name it has: it is being added again, from
// wherever it moved to.
func free(name string, known []domain.Vault, self string) string {
	taken := func(candidate string) bool {
		for _, v := range known {
			if v.ID != self && sameName(v.Name, candidate) {
				return true
			}
		}
		return false
	}
	if !taken(name) {
		return name
	}
	for n := 2; ; n++ {
		numbered := fmt.Sprintf("%s %d", name, n)
		if !taken(numbered) {
			return numbered
		}
	}
}

// sameName reports whether two names name one vault. A name from a file picker
// and the same name typed at a command line are composed differently, and the
// case is the person's to choose.
func sameName(a, b string) bool {
	return strings.EqualFold(norm.NFC.String(a), norm.NFC.String(b))
}
