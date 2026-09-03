package domain

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrOverlaps is a folder that lies inside a vault on the list, or holds one.
var ErrOverlaps = errors.New("a vault does not lie inside another")

// Vaults is the vaults this installation knows about, and the rules that hold
// over the list as a whole.
type Vaults []Vault

// Room says whether a folder can be a vault of its own. The vault in the way is
// named, because the person is choosing a folder and needs to know which one.
func (vs Vaults) Room(path string) error {
	for _, other := range vs {
		if other.Path == path {
			continue
		}
		if within(path, other.Path) {
			return fmt.Errorf("%w: %s is inside the vault %s at %s",
				ErrOverlaps, path, other.Name, other.Path)
		}
		if within(other.Path, path) {
			return fmt.Errorf("%w: %s holds the vault %s at %s",
				ErrOverlaps, path, other.Name, other.Path)
		}
	}
	return nil
}

// Called is the vault of this name. The vault self is not an answer: it is the
// one being named.
//
// A name from a file picker and the same name typed at a command line are
// composed differently, and the case is the person's to choose, so neither
// tells two vaults apart.
func (vs Vaults) Called(name string, self VaultID) (Vault, bool) {
	for _, v := range vs {
		if v.ID != self && FoldName(v.Name) == FoldName(name) {
			return v, true
		}
	}
	return Vault{}, false
}

// FreeName is the name, or the lowest free number appended to it.
func (vs Vaults) FreeName(name string, self VaultID) string {
	if _, taken := vs.Called(name, self); !taken {
		return name
	}
	for n := 2; ; n++ {
		numbered := fmt.Sprintf("%s %d", name, n)
		if _, taken := vs.Called(numbered, self); !taken {
			return numbered
		}
	}
}

// within reports whether path lies below root.
func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
