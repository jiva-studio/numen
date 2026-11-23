// Package commit seals the notes that changed on this device and commits them to the server as signed
// revisions.
package commit

import (
	"context"
	"errors"
	"io"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const batchSize = 100

// Status is how the server took a note.
type Status int

const (
	// Committed means the revision is the head of its file now.
	Committed Status = iota + 1
	// ParentMismatch means the head of the file is not the revision the note was changed from.
	ParentMismatch
	// PathTaken means another file of the vault has the path.
	PathTaken
	// TooLarge means the sealed note is larger than the server stores.
	TooLarge
)

// Deps are what the use case needs.
type Deps struct {
	Revisions ports.Revisions
	Blobs     ports.BlobUploader
	Identity  ports.Identity
	Random    io.Reader
}

// Vault is the synced vault: its id on the server and its key.
type Vault struct {
	ID  []byte
	Key [32]byte
}

// File is a note to commit. A note the server knows has its FileID and the revision it was changed from.
type File struct {
	FileID     []byte
	ParentRev  []byte
	Path       string
	Content    []byte
	ModifiedAt int64
}

// Input is the notes to commit to a vault.
type Input struct {
	Vault Vault
	Files []File
}

// Outcome is what became of one note. Head is the head of the file the server holds for ParentMismatch and for
// PathTaken, and ExistingFileID is the file that has the path for PathTaken.
type Outcome struct {
	Path           string
	Status         Status
	FileID         []byte
	RevID          []byte
	Head           *syncv1.StoredRevision
	ExistingFileID []byte
}

// CommitRevisions commits notes in batches of one hundred.
type CommitRevisions struct{ deps Deps }

// NewCommitRevisions returns the use case over the dependencies.
func NewCommitRevisions(deps Deps) *CommitRevisions { return &CommitRevisions{deps: deps} }

// Execute returns one outcome for each file, in the order of the input.
func (u *CommitRevisions) Execute(ctx context.Context, in Input) ([]Outcome, error) {
	author, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	run := &batchRun{deps: u.deps, vault: in.Vault, author: author.DeviceSignKey}
	outcomes := make([]Outcome, 0, len(in.Files))
	for start := 0; start < len(in.Files); start += batchSize {
		batch, err := run.execute(ctx, in.Files[start:min(start+batchSize, len(in.Files))])
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, batch...)
	}
	return outcomes, nil
}

var errUnexpectedAnswer = errors.New("the server answered a batch it did not commit with no conflict")
