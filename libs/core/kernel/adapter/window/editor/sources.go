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

// Sources are what the passes behind a vault run over. The composition root
// makes them from the installation's adapters.
type Sources struct {
	Reading  VaultReading
	Starting VaultStarting
	Texts    TextVersions
	Plugins  PluginWork
	Index    IndexHandle
}

// VaultReading reads a whole vault into the index.
type VaultReading interface {
	ReadWholeVault(
		ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
	) (vaultreading.ReadWholeVault, error)
}

// VaultStarting starts the walk and the watch of one vault.
type VaultStarting interface {
	StartVault(
		ctx context.Context, v domain.Vault, rebuild bool,
		notified func(paths, assets []string, reload bool),
		handleError window.ErrorHandler,
	) (
		read func(ctx context.Context, during func()) (notes int, err error),
		run func(ctx context.Context),
		refresh vaults.Refresh,
		unwatched error,
	)
}

// TextVersions is how the window follows the text versions of the vault it shows.
type TextVersions interface {
	SetTextVersionListener(listener textport.VersionListener)
	// ReportTextFilesChanged reports to the text version listener each changed file of the vault whose
	// live text version changed.
	ReportTextFilesChanged(ctx context.Context, v domain.Vault, paths []string) error
}

// PluginWork is what the plugins run over the files of a vault.
type PluginWork interface {
	// OpenTriggers is the triggers of the enabled plugins over the files of a vault, nil where the
	// installation runs no plugins.
	OpenTriggers() *trigger.Triggers
	// WatchPluginChanges reports each time the host applied settings, which enables, disables and
	// configures plugins, or forgot the failure of a plugin. The returned function stops the reports.
	WatchPluginChanges() (<-chan struct{}, func())
}

// IndexHandle is the installation's cache, open for as long as the window is.
type IndexHandle interface {
	GetRefresh() vaults.Refresh
	CloseIndex() error
	GetProblemQueries() problemport.ProblemQueries
	GetPassageQueries() sourceport.PassageQueries
}
