package editor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	sourceport "github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext/sourcecontract"
	problemport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/problemcontract"
	textport "github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion/textcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors/embeddingcontract"
)

// ReadVaultFunc reads a whole vault into the index.
type ReadVaultFunc func(
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (vaultreading.ReadWholeVault, error)

// StartVaultFunc starts the walk and the watch of one vault.
type StartVaultFunc func(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError window.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
)

// Sources are what the passes behind a vault run over. The composition root makes them from the
// installation's adapters.
type Sources struct {
	ReadVault  ReadVaultFunc
	StartVault StartVaultFunc

	// SetTextVersionListener sets who is told of each change of the text versions.
	SetTextVersionListener func(listener textport.VersionListener)
	// ReportTextFilesChanged reports to the text version listener each changed file of the vault whose
	// live text version changed.
	ReportTextFilesChanged func(ctx context.Context, v domain.Vault, paths []string) error

	// OpenTriggers is the triggers of the enabled plugins over the files of a vault, nil where the
	// installation runs no plugins.
	OpenTriggers func() *trigger.Triggers
	// WatchPluginChanges reports each time the host applied settings, which enables, disables and
	// configures plugins, or forgot the failure of a plugin. The returned function stops the reports.
	WatchPluginChanges func() (<-chan struct{}, func())

	// The installation's cache, open for as long as the window is.
	GetRefresh        func() vaults.Refresh
	CloseIndex        func() error
	GetProblemQueries func() problemport.ProblemQueries
	GetPassageQueries func() sourceport.PassageQueries
}
