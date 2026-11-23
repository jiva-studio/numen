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

// Execute returns one outcome for each file, in the order of the input.
func (u *CommitRevisions) Execute(ctx context.Context, in transfer.CommitInput) ([]transfer.Outcome, error) {
	author, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	run := &batchRun{deps: u.deps, vault: in.Vault, author: author.DeviceSignKey}
	outcomes := make([]transfer.Outcome, 0, len(in.Files))
	var small []transfer.File
	flush := func() error {
		if len(small) == 0 {
			return nil
		}
		batch, err := run.execute(ctx, small)
		outcomes, small = append(outcomes, batch...), nil
		return err
	}
	for _, file := range in.Files {
		if file.Stat == nil {
			if small = append(small, file); len(small) == note.BatchFiles {
				if err := flush(); err != nil {
					return nil, err
				}
			}
			continue
		}
		if err := flush(); err != nil {
			return nil, err
		}
		outcome, err := run.executeBig(ctx, in.VaultID, file)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, flush()
}

var errUnexpectedAnswer = errors.New("the server answered a batch it did not commit with no conflict")
