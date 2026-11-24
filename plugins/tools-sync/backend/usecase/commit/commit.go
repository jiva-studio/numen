// Package commit seals the files that changed on this device and commits them to the server as signed
// revisions: whole files, files that moved and files that were removed.
package commit

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Uploader sends a file too big to read whole.
type Uploader interface {
	Execute(ctx context.Context, in transfer.UploadInput) (transfer.UploadResult, error)
	Forget(ctx context.Context, vault domain.VaultID, fileID []byte) error
}

// Deps are what the use case needs.
type Deps struct {
	Revisions ports.Revisions
	Blobs     ports.BlobUploader
	Identity  ports.Identity
	Crypto    ports.Crypto
	Uploads   Uploader
	Random    io.Reader
}

// CommitRevisions commits files in batches of note.BatchFiles; a file too big to read whole goes alone.
type CommitRevisions struct{ deps Deps }

// NewCommitRevisions returns the use case over the dependencies.
func NewCommitRevisions(deps Deps) *CommitRevisions { return &CommitRevisions{deps: deps} }

// Execute returns one outcome for each file, in the order of the input. A note under a folder of shares is sealed under
// the key of that share.
func (u *CommitRevisions) Execute(ctx context.Context, in ports.CommitInput) ([]transfer.Outcome, error) {
	author, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	queue := &commitQueue{
		run: &batchRun{deps: u.deps, vault: in.Vault, author: author.DeviceSignKey, shares: in.Shares}, vaultID: in.VaultID,
		outcomes: make([]transfer.Outcome, 0, len(in.Files)), small: nil,
	}
	for _, file := range in.Files {
		if err := queue.add(ctx, file); err != nil {
			return nil, err
		}
	}
	return queue.outcomes, queue.flush(ctx)
}

type commitQueue struct {
	run      *batchRun
	vaultID  domain.VaultID
	outcomes []transfer.Outcome
	small    []transfer.File
}

func (q *commitQueue) add(ctx context.Context, file transfer.File) error {
	if file.Stat == nil {
		q.small = append(q.small, file)
		if len(q.small) == note.BatchFiles {
			return q.flush(ctx)
		}
		return nil
	}
	if err := q.flush(ctx); err != nil {
		return err
	}
	outcome, err := q.run.executeBig(ctx, q.vaultID, file)
	if err != nil {
		return err
	}
	q.outcomes = append(q.outcomes, outcome)
	return nil
}

func (q *commitQueue) flush(ctx context.Context) error {
	if len(q.small) == 0 {
		return nil
	}
	batch, err := q.run.execute(ctx, q.small)
	q.outcomes, q.small = append(q.outcomes, batch...), nil
	return err
}

var (
	errUnexpectedAnswer = errors.New("the server answered a batch it did not commit with no conflict")
	errNoShareKey       = errors.New("the share has no key for its epoch")
)
