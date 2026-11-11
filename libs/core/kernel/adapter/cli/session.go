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
// through it: listing vaults opens no index, and a search opens no provider that fills one. Everything a
// command opens is closed by Close, once, when the run ends.
type Session interface {
	// OpenVaults returns the use cases over the list of vaults.
	OpenVaults() (Vaults, error)
	// OpenIndexed opens the index and returns the use cases that read or change it.
	OpenIndexed(ctx context.Context, options IndexOptions) (Indexed, error)
	// GetPluginCommands returns the commands the enabled plugins declare, and what runs one.
	GetPluginCommands() PluginCommands
	// Close ends what the run opened. It returns what the background work of the run reported as failed, one
	// line each, and the failure to close.
	Close() (failures []string, err error)
}

// IndexOptions say how the index of a run is used: whether a scan reads every file again, and where the search
// reports that it runs on words alone.
type IndexOptions struct {
	Rebuild      bool
	ErrorHandler window.ErrorHandler
}

// Indexed is what the commands that read or change the index work through, over the one index a run opened.
type Indexed struct {
	Rename     vault.RenameByName
	Unregister vault.UnregisterByName
	Erase      vault.EraseByName
	Scan       Scan
	Search     Search
	Links      query.ShowLinksByName
	Problems   vault.GetProblemsByName
}

// Vaults is what the vault commands that touch the list alone operate on: the use cases over the list this
// installation keeps.
type Vaults struct {
	Add        vault.Add
	Known      registry.GetKnownVaults
	RecordOpen vault.RecordOpenByName
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
