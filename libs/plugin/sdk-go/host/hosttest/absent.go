package hosttest

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

var errAbsentNotImplemented = errors.New("not implemented")

var _ host.AbsentFiles = (*Vault)(nil)

// PutAbsentFiles lists the files, replacing the entry of each path.
func (v *Vault) PutAbsentFiles(context.Context, []host.AbsentFile) error {
	return errAbsentNotImplemented
}

// ListAbsentFiles returns the files listed.
func (v *Vault) ListAbsentFiles(context.Context) ([]host.AbsentFile, error) {
	return nil, errAbsentNotImplemented
}

// FillFile places the store file at the path of a listed file.
func (v *Vault) FillFile(context.Context, string, string) error { return errAbsentNotImplemented }

// EvictFile deletes the bytes of a file and lists it.
func (v *Vault) EvictFile(context.Context, string, string) error { return errAbsentNotImplemented }
