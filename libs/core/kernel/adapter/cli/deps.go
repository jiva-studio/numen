package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	machineport "github.com/jiva-studio/numen/libs/core/kernel/usecase/file/machinecontract"
	problemport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/problemcontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	vaultport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/vaultcontract"
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
	Vaults func() (Vaults, error)

	Scan func(ctx context.Context, v domain.Vault, rebuild bool) (Scan, error)

	Search func(ctx context.Context, v domain.Vault, errorHandler window.ErrorHandler) (Search, error)

	Links func(ctx context.Context) (Links, error)

	Problems func(ctx context.Context) (Problems, error)

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

	Rows func(ctx context.Context) (VaultRows, error)
}

func (v Vaults) find(nameOrPath string) (domain.Vault, error) {
	target, err := registry.NewFind(v.Registry).Execute(nameOrPath)
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
	Read       vault.ReadWholeVault
	Summary    func(ctx context.Context) (domain.VaultSummary, error)
	Unembedded error
	Close      func() error
}

// Search is one question answered.
type Search struct {
	Search search.Search
	Words  error
	Close  func() error
}

// Links is what one note points at and what points at it.
type Links struct {
	Show  query.ShowLinks
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

func closeInto(err *error, closer func() error) {
	if closer == nil {
		return
	}
	if closeErr := closer(); closeErr != nil {
		*err = errors.Join(*err, closeErr)
	}
}
