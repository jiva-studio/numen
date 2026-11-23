// Package fetchfile downloads one absent file of a synced vault and places it in the vault.
package fetchfile

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// RangeSize is how many bytes one request of a download asks for.
const RangeSize = 1 << 20

// ErrBlobMissing means the server answered and holds no blob for the file.
var ErrBlobMissing = errors.New("the server has no blob for the file")

// State is the link and the records of the vault.
type State interface {
	ports.Links
	ports.Records
}

// Deps are what the use case needs.
type Deps struct {
	Revisions ports.Revisions
	Blobs     ports.BlobDownloader
	Ranges    ports.BlobRanges
	Crypto    ports.Crypto
	Host      ports.Host
	State     State
}

// FetchFile downloads the blob of an absent file in ranges, keeps what came in the store folder of the plugin
// so that a download resumes, and fills the file once its hash is the one of the entry.
type FetchFile struct{ deps Deps }

// NewFetchFile returns the use case over the dependencies.
func NewFetchFile(deps Deps) *FetchFile { return &FetchFile{deps: deps} }

// Execute downloads the file at path. Two calls for one path run one download, and the other waits for it.
func (u *FetchFile) Execute(context.Context, domain.VaultID, string) error {
	return errors.New("not implemented")
}
