package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
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

	Scan func(ctx context.Context, rebuild bool) (Scan, error)

	Search func(ctx context.Context, errorHandler window.ErrorHandler) (Search, error)

	Links func(ctx context.Context) (Links, error)

	Problems func(ctx context.Context) (Problems, error)

	PluginCommands func() PluginCommands
}

// Vaults is what the vault commands operate on: the use cases over the list this
// installation keeps. The commands that also change what the index
// remembers about a vault open it when they run.
type Vaults struct {
	Add        vault.Add
	Known      registry.GetKnownVaults
	RecordOpen vault.RecordOpenByName

	OpenRename     func(ctx context.Context) (Renaming, error)
	OpenUnregister func(ctx context.Context) (Unregistering, error)
	OpenErase      func(ctx context.Context) (Erasing, error)
}

func formatVaultError(err error) error {
	if unknown, ok := errors.AsType[*registry.UnknownError](err); ok {
		return fmt.Errorf("%w — add it with: numen-cli vault add %s", err, unknown.Name)
	}
	if lookup, ok := errors.AsType[*registry.LookupError](err); ok {
		return fmt.Errorf("%w — add it with: numen-cli vault add %s", err, lookup.Name)
	}
	return err
}

// Renaming is the rename of a vault over an open index.
type Renaming struct {
	Rename vault.RenameByName
	Close  func() error
}

// Unregistering takes a vault off the list over an open index.
type Unregistering struct {
	Unregister vault.UnregisterByName
	Close      func() error
}

// Erasing removes a vault and moves its folder to the trash, over an open index.
type Erasing struct {
	Erase vault.EraseByName
	Close func() error
}

// Scan is one vault made searchable.
type Scan struct {
	Read       indexing.ReadWholeVault
	Summary    func(ctx context.Context, v domain.Vault) (domain.VaultSummary, error)
	Now        func() time.Time
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
	Show  query.ShowLinksByName
	Close func() error
}

// Problems is what a scan could not act on, as reported by the index.
//
// The set of checks applied to a vault is part of the core's vocabulary and not of
// an installation, so the terminal uses the standard set.
type Problems struct {
	Problems vault.GetProblemsByName
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
