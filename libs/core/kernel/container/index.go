package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

// vaultIndex is an open index database with the state that the vaults read into it share. Each use case
// is handed the ports it uses from the database; no use case package names this type.
type vaultIndex struct {
	*index.DB
	File     string
	Walks    indexing.Walks
	Ledger   *indexing.Ledger
	Versions textversion.VersionStores
}
