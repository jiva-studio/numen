package source

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetArtifactList lists what was made from one file of the vault: the text read
// out of a scan, a recording transcribed, the words a site published with a
// video, the prose of a page and a copy of a video.
type GetArtifactList struct {
	store artifactStore
}

func NewGetArtifactList(queries port.SourceQueries, derived port.DerivedStores) *GetArtifactList {
	return &GetArtifactList{store: artifactStore{queries: queries, derived: derived}}
}

// Execute returns the artifacts of one file, none for a file the index does not hold.
func (u *GetArtifactList) Execute(ctx context.Context, v domain.Vault, path string) ([]Artifact, error) {
	set, err := u.store.getArtifacts(ctx, v, path)
	return set.Artifacts, err
}
