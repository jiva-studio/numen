package cli

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
	problemport "github.com/jiva-studio/numen/libs/core/usecase/note/problemcontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
	windowport "github.com/jiva-studio/numen/libs/core/usecase/vault/windowcontract"
)

// Locations are the paths a user may set for this installation, as typed. An
// empty value selects the platform default.
type Locations struct {
	Index      string
	Registry   string
	ServiceDir string
	SafeMode   bool
}

// Deps is what one run of the terminal depends on: one opener per command.
// Nothing is opened until the command that needs it calls it.
//
// A command opens only its own machinery: listing vaults opens no index, and a
// search opens no provider that fills one.
type Deps struct {
	// Vaults is the list of vaults this installation keeps, which the vault
	// commands operate on. Every command opens it: the typed argument is a name,
	// a path or an identity, and the list resolves it to a vault.
	Vaults func() (Vaults, error)

	// Scan opens the walk that brings the index level with one vault, the
	// extraction of the text of its other files and the making of its vectors. rebuild reads every
	// file, whatever the index remembers.
	Scan func(ctx context.Context, v domain.Vault, rebuild bool) (Scan, error)

	// Search opens the search that answers a query about v. errorHandler receives
	// errors from a part of the search that could not run.
	Search func(ctx context.Context, v domain.Vault, errorHandler windowport.ErrorHandler) (Search, error)

	// Links opens what a note points at and what points at it.
	Links func(ctx context.Context) (Links, error)

	// Problems opens what a scan could not act on.
	Problems func(ctx context.Context) (Problems, error)

	// PluginCommands opens the commands the enabled plugins declare. Left nil, there are none.
	PluginCommands func() PluginCommands
}

// Vaults is what the vault commands operate on: the list this installation keeps,
// the folders it names, the folder identity, the trash location of this
// machine, and the clock.
type Vaults struct {
	Registry vaultport.VaultRegistry
	Readers  vaultport.VaultReaders
	Identity vaultport.VaultIdentity
	Trash    machineport.Trash
	Now      machineport.Clock

	// Rows opens what the index remembers about vaults, which a rename, a
	// forget and an erasure change together with the list. Listing vaults calls
	// it not at all, and opens no database.
	Rows func(ctx context.Context) (VaultRows, error)
}

// find resolves the typed argument against the list and, when it resolves
// to nothing, adds a hint on how to add the vault. User-facing messages belong here.
func (v Vaults) find(nameOrPath string) (domain.Vault, error) {
	target, err := vault.NewFind(v.Registry).Execute(nameOrPath)
	if err != nil {
		return domain.Vault{}, fmt.Errorf("%w — add it with: numen-cli vault add %s", err, nameOrPath)
	}
	return target, nil
}

// VaultRows is what the index remembers about vaults, open.
type VaultRows struct {
	Vaults vaultport.VaultRepository
	Close  func() error
}

// Scan is one vault made searchable.
type Scan struct {
	// Read is the walk, the text of the other files and the vectors, in the order they run in.
	Read vault.ReadWholeVault
	// Summary is what the index holds about the vault, queried after the walk
	// ends.
	Summary func(ctx context.Context) (domain.VaultSummary, error)
	// Unembedded is why no vectors are being made, and nothing where they are.
	Unembedded error
	Close      func() error
}

// Search is one question answered.
type Search struct {
	Search search.Search
	// Words is why the question is answered by its words alone, and nothing
	// where a model answers it too.
	Words error
	Close func() error
}

// Links is what one note points at and what points at it.
type Links struct {
	Show  note.ShowLinks
	Close func() error
}

// Problems is what a scan could not act on, as reported by the index.
//
// The checks are not opened here. The set of checks applied to a vault is part of
// the core's vocabulary and not of an installation, so the terminal uses the
// standard set and this opens what evaluates them.
type Problems struct {
	Problems problemport.ProblemQueries
	Close    func() error
}

// closeIfOpen gives back what an opener opened, and does nothing where it
// opened nothing.
func closeIfOpen(closer func() error) {
	if closer != nil {
		_ = closer()
	}
}
