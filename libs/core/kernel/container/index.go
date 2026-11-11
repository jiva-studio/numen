package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// Index is an open index database with the state that the vaults read into it share. Each use case
// is handed the ports it uses from the database; no use case package names this type.
type Index struct {
	*index.DB
	File     string
	Walks    vault.Walks
	Ledger   *vault.Ledger
	Versions textport.VersionStores
}
