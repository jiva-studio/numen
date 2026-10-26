package source

import (
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrOutside is a path that does not stay in the vault.
var ErrOutside = vaultport.ErrOutside

// IsNoNote reports whether an error says there is no note at the path.
func IsNoNote(err error) bool { return vaultport.IsNoNote(err) }
