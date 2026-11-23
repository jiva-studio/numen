package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

// FolderContents is what a folder holds and the kind of tab each of its notes opens in.
type FolderContents struct {
	Folder
	TabKinds map[string]string
}

// ListFolder lists what a folder holds with the kind of tab each note in it opens in.
type ListFolder struct {
	entries  *GetFolderEntries
	tabKinds windowport.TabKinds
}

func NewListFolder(entries *GetFolderEntries, tabKinds windowport.TabKinds) *ListFolder {
	return &ListFolder{entries: entries, tabKinds: tabKinds}
}

func (u *ListFolder) Execute(ctx context.Context, v domain.Vault, path string) (FolderContents, error) {
	folder, err := u.entries.Execute(ctx, v, path)
	if err != nil {
		return FolderContents{}, err
	}
	var notes []string
	for _, one := range folder.Entries {
		if one.Kind == domain.KindNote {
			notes = append(notes, one.Path)
		}
	}
	kinds, err := u.tabKinds.ListTabKinds(ctx, v, notes)
	if err != nil {
		return FolderContents{}, err
	}
	return FolderContents{Folder: folder, TabKinds: kinds}, nil
}

// ListFileKinds lists what the vault holds at some paths with the kind of tab each note opens in.
type ListFileKinds struct {
	kinds  *GetFileKinds
	choice *plugin.TabKindChoice
}

func NewListFileKinds(kinds *GetFileKinds, choice *plugin.TabKindChoice) *ListFileKinds {
	return &ListFileKinds{kinds: kinds, choice: choice}
}

// Execute lists the files at the paths in the order given.
func (u *ListFileKinds) Execute(ctx context.Context, v domain.Vault, paths []string) (FolderContents, error) {
	folder, err := u.kinds.Execute(ctx, v, paths)
	if err != nil {
		return FolderContents{}, err
	}
	heads := make(map[string]frontmatter.Head, len(folder.Properties))
	for path, properties := range folder.Properties {
		heads[path] = frontmatter.Head{Type: folder.Types[path], Values: properties}
	}
	return FolderContents{Folder: folder, TabKinds: u.choice.ChooseMany(heads)}, nil
}
