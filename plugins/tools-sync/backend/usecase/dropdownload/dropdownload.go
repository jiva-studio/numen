// Package dropdownload turns a downloaded file of a synced vault back into an absent one.
package dropdownload

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrChanged means the file has changes that are not sent, so it is not dropped.
var ErrChanged = host.ErrChanged

// Deps are what the use case needs.
type Deps struct {
	Host  ports.Host
	State ports.Records
}

// DropDownload asks the host to evict a file whose hash is the base of its record.
type DropDownload struct{ deps Deps }

// NewDropDownload returns the use case over the dependencies.
func NewDropDownload(deps Deps) *DropDownload { return &DropDownload{deps: deps} }

// Execute drops the download of the file at path. The record of the file stays, so the pass neither sends the
// file nor lists it as deleted.
func (u *DropDownload) Execute(ctx context.Context, id domain.VaultID, path string) error {
	records, err := u.deps.State.ReadRecords(ctx, id)
	if err != nil {
		return err
	}
	at := slices.IndexFunc(records, func(one note.FileRecord) bool { return one.Path == path })
	if at < 0 {
		return fmt.Errorf("drop the download of %s: the file is not synced: %w", path, host.ErrNotExist)
	}
	return u.deps.Host.Absent(id).EvictFile(ctx, path, records[at].BaseHash)
}
