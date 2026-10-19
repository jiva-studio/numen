package editor

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Assembly is what this window asks whoever put the application together: the
// vault list and the cache of the installation it serves, the ports it reaches
// a vault through, and the scenarios and passes it opens one with.
//
// Everything belonging to the installation is answered as it stands. What
// belongs to a vault is opened here and opened again each time another vault is
// shown.
//
// The cache is answered as the ports asked of it, so this window names no
// store. It is opened before the window is, and the window closes it.
type Assembly interface {
	// OpenBookClient opens a client to the book reader plugin, returning the client and a closer.
	OpenBookClient(ctx context.Context) (any, func(), error)

	// GetPluginDir returns the root directory of an installed plugin by ID, or false if not found.
	GetPluginDir(id string) (string, bool)

	// EmitEvent sends an event notification to listeners and registered plugins.
	EmitEvent(event string)

	// GetRegistry is the list of vaults this installation holds.
	GetRegistry() port.VaultRegistry

	// GetVaultReaders open a vault for reading and GetVaultWriters for
	// changing; GetDerivedStores is the shelf the application keeps its own
	// files on inside one.
	GetVaultReaders() port.VaultReaders
	GetVaultWriters() port.VaultWriters
	GetDerivedStores() port.DerivedStores

	// GetTextExtractor takes the text out of a document and GetPageRenderer
	// draws one, GetVaultIdentity says which vault a folder is, and GetClock is
	// what time it is.
	GetTextExtractor() port.TextExtractor
	GetPageRenderer() port.PageRenderer
	GetVaultIdentity() port.VaultIdentity
	GetClock() port.Clock

	// The cache, as the ports this window asks it, and the close that hands the
	// file back.
	GetNoteQueries() port.NoteQueries
	GetLinkQueries() port.LinkQueries
	GetProblemQueries() port.ProblemQueries
	GetIndexProgress() port.IndexProgress
	GetSourceRepository() port.SourceRepository
	GetSourceQueries() port.SourceQueries
	GetPassageQueries() port.PassageQueries
	CloseIndex() error

	// The use cases that read and write the settings and the file they stand in.
	// The day is where one day of review gives way to the next.
	OpenGetSettings(day flashcards.Day) *settings.Get
	OpenWriteSettings() *settings.Write
	OpenGetSettingsFile() *settings.GetFile
	OpenWriteSettingsFile() *settings.WriteFile
	// The use cases that store and remove the key of a secret a plugin declares.
	OpenWritePluginSecret() *settings.WriteSecret
	OpenDeletePluginSecret() *settings.DeleteSecret
	// OpenRetryPlugin starts a plugin the host holds off once more.
	OpenRetryPlugin() *settings.RetryPlugin
	// ReportPluginFailures puts a task in the list for each plugin the host holds off.
	ReportPluginFailures(tasks *run.Tasks)

	// IsRebuildingIndex is whether this launch was asked to read every file
	// into the cache again. It belongs to the vault the window opened on and to
	// no vault shown later.
	IsRebuildingIndex() bool

	// IsTranscribingUnasked is whether a recording the vault holds no
	// transcript for is listened to without anybody asking.
	IsTranscribingUnasked() bool

	// OpenEmbedders opens the model this installation turns text into vectors
	// with, and the one a query is turned into a vector by. The two are one
	// object where the settings name one provider.
	OpenEmbedders(
		ctx context.Context, tasks *run.Tasks,
	) (indexing, asking port.Embedder, closer func() error, why error)

	// NewSearch is how the window answers a question about the text the vault
	// holds. handleError is where a half that could not run is said.
	NewSearch(asking port.Embedder, handleError func(error)) search.Search

	// OpenNotes, OpenCards, OpenVaults and OpenFlashcards are this
	// installation's scenarios, built against the cache above. index brings
	// what a write touched up to date before the write answers.
	OpenNotes(index note.Levels) note.Scenarios
	OpenCards(index note.Levels) cards.Scenarios
	OpenVaults(moves note.Move) vaults.Scenarios
	OpenFlashcards(index note.Levels) flashcards.Scenarios

	// OpenSourceTexts reads and writes the text made from a file of the vault.
	OpenSourceTexts() source.Texts

	// OpenDownloader reaches the address a link note points at, and is nothing
	// on a machine holding neither of the tools that reach one. OpenImportURL
	// is one such address fetched into the vault.
	OpenDownloader(ctx context.Context) port.Downloader
	OpenImportURL(ctx context.Context, by port.Downloader) source.ImportURL

	// OpenRecognitionWorker reads a scanned document and
	// OpenTranscriptionWorker hears a recording, each for whoever asks. models
	// is what the two take turns at: one run holds this machine's models at a
	// time.
	OpenRecognitionWorker(
		ctx context.Context, tasks *run.Tasks, models *run.Lock,
	) *recognition.Worker
	OpenTranscriptionWorker(
		ctx context.Context, tasks *run.Tasks, models *run.Lock,
	) *transcription.Worker

	// GetRefresh brings named notes up to date for a window standing on no
	// vault. A window showing one asks what that vault was opened with.
	GetRefresh() vaults.Refresh

	// ReadWholeVault is what makes a vault answer: the notes read, the books
	// read, and the vectors made.
	ReadWholeVault(
		ctx context.Context, embedder port.Embedder, v domain.Vault, rebuild bool,
	) (vaults.ReadWholeVault, error)

	// StartVault opens a vault behind the window: the walk that brings the
	// cache level with it, and the watch that keeps it level while it is shown.
	//
	// told is called each time the two are level again, with the notes that are
	// different now, the files that changed and are not notes, and whether the
	// whole vault has to be looked at again. handleError is called with what
	// went wrong along the way, and with nothing when a later attempt succeeds.
	//
	// It answers with the walk, the watch, what brings a note up to date
	// through this opening, and why the vault is not being followed.
	StartVault(
		ctx context.Context, v domain.Vault, rebuild bool,
		told func(paths, assets []string, reload bool),
		handleError port.ErrorHandler,
	) (
		read func(ctx context.Context, during func()) (notes int, err error),
		run func(ctx context.Context),
		refresh vaults.Refresh,
		unwatched error,
	)
}
