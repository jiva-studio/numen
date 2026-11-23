// Package preview tells what connecting a folder to a synced vault would join, and writes nothing.
package preview

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var errFeedReset = errors.New("the file change feed reset while the notes were read")

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
func (u *PreviewConnect) Execute(ctx context.Context, in Input) (Preview, error) {
	heads, err := u.deps.Heads.ListHeads(ctx, in.RemoteID)
	if err != nil {
		return Preview{}, err
	}
	local, err := listLocal(ctx, u.deps.Host.Files(in.VaultID))
	if err != nil {
		return Preview{}, err
	}
	return pair(local, heads), nil
}

func pair(local, remote []ports.Head) Preview {
	locals, remotes := index(local), index(remote)
	var found Preview
	for key, one := range locals {
		switch note.Meet(one.Hash, remotes[key].Hash) {
		case note.Same:
			found.Same = append(found.Same, one.Path)
		case note.LocalOnly:
			found.LocalOnly = append(found.LocalOnly, one.Path)
		case note.Differ:
			found.Differ = append(found.Differ, one.Path)
		case note.RemoteOnly:
		}
	}
	for key, one := range remotes {
		if _, isLocal := locals[key]; !isLocal {
			found.RemoteOnly = append(found.RemoteOnly, one.Path)
		}
	}
	for _, paths := range [][]string{found.Same, found.LocalOnly, found.RemoteOnly, found.Differ} {
		slices.Sort(paths)
	}
	return found
}

func index(heads []ports.Head) map[string]ports.Head {
	byKey := make(map[string]ports.Head, len(heads))
	for _, one := range heads {
		byKey[note.PathKey(one.Path)] = one
	}
	return byKey
}

func listLocal(ctx context.Context, files ports.Files) ([]ports.Head, error) {
	added := []string{host.EventFileAdded}
	head, err := files.ListFileChanges(ctx, host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: "", Seq: 0}, Events: added, Limit: 0})
	if err != nil {
		return nil, err
	}
	query := host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: head.Next.FeedID, Seq: 0}, Events: added, Limit: 0}
	var listed []ports.Head
	for {
		page, err := files.ListFileChanges(ctx, query)
		if err != nil {
			return nil, err
		}
		if page.IsReset {
			return nil, errFeedReset
		}
		for _, change := range page.Changes {
			if note.IsSyncable(change.Path) {
				listed = append(listed, ports.Head{Path: change.Path, Hash: change.Hash})
			}
		}
		if !page.HasMore {
			return listed, nil
		}
		query.Since = page.Next
	}
}
