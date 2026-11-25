package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/operationresult"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
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

// Session is the installation one run of the terminal works with. A command opens only its own machinery
// through it: listing vaults opens no index, and a search opens no provider that fills one. OpenIndexed opens
// the index and everything a command opens is closed by Close, once, when the run ends. Close returns what
// the background work of the run reported as failed, one line each, and the failure to close.
type Session struct {
	openVaults        func() (Vaults, error)
	openIndexed       func(ctx context.Context, options IndexOptions) (IndexRun, error)
	getPluginCommands func() PluginCommands
	close             func() (failures []string, err error)
}

// NewSession takes no nil function: the methods of Session call them without a check.
func NewSession(
	openVaults func() (Vaults, error),
	openIndexed func(ctx context.Context, options IndexOptions) (IndexRun, error),
	getPluginCommands func() PluginCommands,
	closeRun func() (failures []string, err error),
) Session {
	return Session{openVaults: openVaults, openIndexed: openIndexed, getPluginCommands: getPluginCommands, close: closeRun}
}

func (s Session) OpenVaults() (Vaults, error) { return s.openVaults() }

func (s Session) OpenIndexed(ctx context.Context, options IndexOptions) (IndexRun, error) {
	return s.openIndexed(ctx, options)
}

func (s Session) GetPluginCommands() PluginCommands { return s.getPluginCommands() }

func (s Session) Close() (failures []string, err error) { return s.close() }

// IndexOptions say how the index of a run is used: whether a scan reads every file again, and where the search
// reports that it runs on words alone.
type IndexOptions struct {
	Rebuild      bool
	ErrorHandler window.ErrorHandler
}

// IndexRun is what the commands that read or change the index work through, over the one index a run opened.
type IndexRun struct {
	Changes          VaultChanges
	Scan             Scan
	Search           Search
	Links            query.ShowLinksByName
	Problems         vault.GetProblemsByName
	Query            searchqueryusecase.FindSearchResultsByName
	OperationResults *operationresult.ClearOperationResults
}

// VaultChanges is the commands that rename, unregister or erase a vault named on the command line.
type VaultChanges struct {
	Rename     vault.RenameByName
	Unregister vault.UnregisterByName
	Erase      vault.EraseByName
}

// Vaults is what the vault commands that touch the list alone operate on: the use cases over the list this
// installation keeps.
type Vaults struct {
	Add        vault.Add
	Known      registry.GetKnownVaults
	RecordOpen vault.RecordOpenByName
}

func formatVaultError(err error) error {
	if unknown, ok := errors.AsType[*vaultdomain.UnknownError](err); ok {
		return fmt.Errorf("%w — add it with: numen-cli vault add %s", err, unknown.Name)
	}
	if lookup, ok := errors.AsType[*vaultdomain.LookupError](err); ok {
		return fmt.Errorf("%w — add it with: numen-cli vault add %s", err, lookup.Name)
	}
	return err
}

// Scan is one vault made searchable.
type Scan struct {
	Scan indexing.ScanVault
	Now  func() time.Time
}

// Search is one question answered.
type Search struct {
	Search search.Search
	Words  error
}
