package editor

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
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

	// ReadSettings reads every setting as JSON and the file it stands in,
	// ReadSettingsFile that file as its person wrote it, and WriteSettingsFile
	// replaces it whole against the copy the caller last read. GetModels are
	// the models a setting that names one can be set to, and TurnSettings
	// writes settings into the file.
	ReadSettings() (written string, at string, err error)
	ReadSettingsFile() (written string, at string, err error)
	WriteSettingsFile(written string, seen *string) error
	GetModels() []port.Model
	TurnSettings(written []port.Setting) error

	// GetPartsUnderANodeBounds is how many parts a node may be asked to hang,
	// at each end, and GetLatestDayStarts how late in the day a day of review
	// may be made to begin.
	GetPartsUnderANodeBounds() (least, most float64)
	GetLatestDayStarts() string

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
		ctx context.Context, tasks *task.Tasks,
	) (indexing, asking port.Embedder, close func() error, why error)

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
		ctx context.Context, tasks *task.Tasks, models *source.Lock,
	) *source.RecognitionWorker
	OpenTranscriptionWorker(
		ctx context.Context, tasks *task.Tasks, models *source.Lock,
	) *source.TranscriptionWorker

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
