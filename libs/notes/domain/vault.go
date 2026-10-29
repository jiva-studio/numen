package domain

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// VaultID names one vault as a ULID.
type VaultID string

// Vault is a folder the user has added to the application. Its identity is a
// ULID carried inside the vault itself.
type Vault struct {
	ID   VaultID
	Name string
	Path string
}

// VaultSummary is how much of a vault the index holds. A read model, shown to
// the user after a scan.
type VaultSummary struct {
	Notes    int
	Headings int
}

// ErrOverlaps reports a folder that is inside a vault on the list, or contains one.
var ErrOverlaps = errors.New("a vault does not lie inside another")

// Vaults is the list of vaults of this installation, and the rules
// that apply to the list as a whole.
type Vaults []Vault

// CheckOverlap reports whether a folder can be a vault of its own. The error
// names the overlapping vault.
func (vs Vaults) CheckOverlap(path string) error {
	for _, other := range vs {
		if other.Path == path {
			continue
		}
		if isBelow(path, other.Path) {
			return fmt.Errorf("%w: %s is inside the vault %s at %s",
				ErrOverlaps, path, other.Name, other.Path)
		}
		if isBelow(other.Path, path) {
			return fmt.Errorf("%w: %s holds the vault %s at %s",
				ErrOverlaps, path, other.Name, other.Path)
		}
	}
	return nil
}

// FindByName returns the vault with this name, excluding the vault `self`,
// which is the vault being named.
//
// Names are compared after folding. A name from a file picker and the same
// name typed at a command line differ in composition and case, and neither
// distinguishes two vaults.
func (vs Vaults) FindByName(name string, self VaultID) (Vault, bool) {
	for _, v := range vs {
		if v.ID != self && FoldName(v.Name) == FoldName(name) {
			return v, true
		}
	}
	return Vault{}, false
}

// FreeName is the name, or the lowest free number appended to it.
func (vs Vaults) FreeName(name string, self VaultID) string {
	if _, found := vs.FindByName(name, self); !found {
		return name
	}
	for n := 2; ; n++ {
		numbered := fmt.Sprintf("%s %d", name, n)
		if _, acquiredVal := vs.FindByName(numbered, self); !acquiredVal {
			return numbered
		}
	}
}

// isBelow reports whether path is below root.
func isBelow(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// IsAbsPath reports whether path is an absolute filesystem path.
func IsAbsPath(path string) bool {
	return filepath.IsAbs(path)
}

// IsRootPath reports whether path is a filesystem root.
func IsRootPath(path string) bool {
	return filepath.Dir(path) == path
}

// CleanPath returns the shortest path name equivalent to path.
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// BasePath returns the last element of path.
func BasePath(path string) string {
	return filepath.Base(path)
}
