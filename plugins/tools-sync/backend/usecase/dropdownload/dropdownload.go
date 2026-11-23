// Package dropdownload turns a downloaded file of a synced vault back into an absent one.
package dropdownload

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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

// Execute drops the download of the file at path.
func (u *DropDownload) Execute(context.Context, domain.VaultID, string) error {
	return errors.New("not implemented")
}
