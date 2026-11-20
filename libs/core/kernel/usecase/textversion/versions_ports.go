package textversion

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// VersionStores opens the versions of one vault to read them and to change them, and
// the content hashes recorded beside them.
type VersionStores interface {
	OpenVersionReader(v domain.Vault) (textport.VersionReader, error)
	OpenVersionWriter(v domain.Vault) (textport.VersionWriter, error)
	OpenContentHashes(v domain.Vault) (textport.ContentHashes, error)
}

// TextFileStores opens the version files of one vault.
type TextFileStores interface {
	Open(v domain.Vault) (textport.TextFiles, error)
}
