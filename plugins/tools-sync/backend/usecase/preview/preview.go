// Package preview tells what connecting a folder to a synced vault would join, and writes nothing.
package preview

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var errNotImplemented = errors.New("not implemented")

// Deps are what the use case needs.
type Deps struct {
	Heads ports.Heads
	Host  ports.Host
}

// Input names the local vault and the synced vault it would join.
type Input struct {
	VaultID  domain.VaultID
	RemoteID string
}

// Preview is the notes of both sides by how they meet, as sorted paths.
type Preview struct {
	Same, LocalOnly, RemoteOnly, Differ []string
}

// PreviewConnect reads both sides and sorts their notes.
type PreviewConnect struct{ deps Deps }

// NewPreviewConnect returns the use case over the dependencies.
func NewPreviewConnect(deps Deps) *PreviewConnect { return &PreviewConnect{deps: deps} }

// Execute reads the heads of the synced vault and the notes of the folder and pairs them by path.
func (u *PreviewConnect) Execute(_ context.Context, _ Input) (Preview, error) {
	return Preview{}, errNotImplemented
}
