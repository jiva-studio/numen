// Package commit seals the notes that changed on this device and commits them to the server as signed
// revisions.
package commit

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const batchSize = 100

// Deps are what the use case needs.
type Deps struct {
	Revisions ports.Revisions
	Blobs     ports.BlobUploader
	Identity  ports.Identity
	Random    io.Reader
}

// CommitRevisions commits notes in batches of one hundred.
type CommitRevisions struct{ deps Deps }

// NewCommitRevisions returns the use case over the dependencies.
func NewCommitRevisions(deps Deps) *CommitRevisions { return &CommitRevisions{deps: deps} }

// Execute returns one outcome for each file, in the order of the input. A note under a folder of shares is sealed under
// the key of that share.
func (u *CommitRevisions) Execute(ctx context.Context, in transfer.CommitInput, shares []share.Root) ([]transfer.Outcome, error) {
	author, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	run := &batchRun{deps: u.deps, vault: in.Vault, author: author.DeviceSignKey, shares: shares}
	outcomes := make([]transfer.Outcome, 0, len(in.Files))
	for start := 0; start < len(in.Files); start += batchSize {
		batch, err := run.execute(ctx, in.Files[start:min(start+batchSize, len(in.Files))])
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, batch...)
	}
	return outcomes, nil
}

var (
	errUnexpectedAnswer = errors.New("the server answered a batch it did not commit with no conflict")
	errNoShareKey       = errors.New("the share has no key for its epoch")
)
