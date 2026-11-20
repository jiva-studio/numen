package note

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// TrashEmptiers opens a vault for emptying its trash.
type TrashEmptiers interface {
	OpenTrashEmptier(v domain.Vault) (vaultport.TrashEmptier, error)
}
