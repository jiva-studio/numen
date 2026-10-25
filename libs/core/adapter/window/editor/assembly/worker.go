package assembly

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/core/usecase/transcription"

	"github.com/jiva-studio/numen/libs/core/port/downloadport"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// WorkerAssembly is what this window asks for background workers, embedding, downloads, and books.
type WorkerAssembly interface {
	// IsTranscribingUnasked is whether a recording the vault holds no
	// transcript for is listened to without anybody asking.
	IsTranscribingUnasked() bool

	// OpenEmbedders opens the model this installation turns text into vectors
	// with, and the one a query is turned into a vector by. The two are one
	// object where the settings name one provider.
	OpenEmbedders(
		ctx context.Context, tasks *run.Tasks,
	) (indexing, asking embeddingport.Embedder, closer func() error, why error)

	// OpenDownloader reaches the address a link note points at, and is nothing
	// on a machine holding neither of the tools that reach one. OpenImportURL
	// is one such address fetched into the vault.
	OpenDownloader(ctx context.Context) downloadport.Downloader
	OpenImportURL(ctx context.Context, downloader downloadport.Downloader) source.ImportURL

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

	// GetBooks is the reader of books that reflow.
	GetBooks() sourceport.Book
}
