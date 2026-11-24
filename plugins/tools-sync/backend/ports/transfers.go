package ports

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

// Uploads is one way to upload: a body that fits one request is one part of one.
type Uploads interface {
	Start(ctx context.Context, vault []byte, storedSize int64) (transfer.Upload, error)
	Resume(ctx context.Context, vault, blob []byte) (transfer.Upload, error)
	PutPart(ctx context.Context, up transfer.Upload, number int, body []byte) error
	Finish(ctx context.Context, up transfer.Upload) error
	Abort(ctx context.Context, up transfer.Upload) error
}

// UploadEntries keeps the unfinished uploads of each vault.
type UploadEntries interface {
	ReadUploads(ctx context.Context, id domain.VaultID) ([]transfer.UploadEntry, error)
	WriteUploads(ctx context.Context, id domain.VaultID, entries []transfer.UploadEntry) error
}

// Digest is the running hash of the stored bytes of a blob: it takes bytes, gives the sum so far, and saves its state
// so a later run can restore it and go on.
type Digest interface {
	Write(p []byte) (int, error)
	Sum() []byte
	Save() ([]byte, error)
	Restore(state []byte) error
}

// Digests makes the digests of the blobs.
type Digests interface {
	Create() Digest
}

// BlobReader reads the blobs on the server.
type BlobReader interface {
	// GetBlob returns a whole blob of up to note.BigFileLimit bytes.
	GetBlob(ctx context.Context, vault, blob []byte) ([]byte, error)
	// OpenBlob streams the blob from the byte from; the caller closes it.
	OpenBlob(ctx context.Context, vault, blob []byte, from int64) (io.ReadCloser, error)
}

// DownloadEntries keeps the unfinished downloads of each vault.
type DownloadEntries interface {
	ReadDownloads(ctx context.Context, id domain.VaultID) ([]transfer.DownloadEntry, error)
	WriteDownloads(ctx context.Context, id domain.VaultID, entries []transfer.DownloadEntry) error
}

// PausedStore keeps the removals of each vault that wait for an answer. ReadPending returns nil where nothing is
// pending.
type PausedStore interface {
	ReadPending(ctx context.Context, id domain.VaultID) (*transfer.PausedRemovals, error)
	WritePending(ctx context.Context, id domain.VaultID, state transfer.PausedRemovals) error
	DeletePending(ctx context.Context, id domain.VaultID) error
}

// Approvals keeps the answers of the person to paused removals. ReadApproval returns nil where there is none.
type Approvals interface {
	ReadApproval(ctx context.Context, id domain.VaultID) (*transfer.Approval, error)
	WriteApproval(ctx context.Context, id domain.VaultID, approval transfer.Approval) error
	DeleteApproval(ctx context.Context, id domain.VaultID) error
}

// Device is the computer the plugin runs on.
type Device interface {
	GetName() string
}
