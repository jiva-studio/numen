package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// TabKindChooser gives a note the kind of tab it opens in, or an empty kind where none applies.
type TabKindChooser interface {
	Execute(path string, noteType domain.NoteType, properties map[string]any) string
}

func chooseTabKinds(folder Folder, chooser TabKindChooser) map[string]string {
	kinds := map[string]string{}
	for path, properties := range folder.Properties {
		if kind := chooser.Execute(path, folder.Types[path], properties); kind != "" {
			kinds[path] = kind
		}
	}
	return kinds
}

// ListTabKinds finds the kind of tab each of the notes at some paths opens in, from the type and the
// frontmatter the index holds, in one question and without touching a file. A path the index holds no
// note at has no kind of tab, as a note whose file cannot be read has none.
type ListTabKinds struct {
	heads   noteport.NoteHeads
	chooser TabKindChooser
}

func NewListTabKinds(heads noteport.NoteHeads, chooser TabKindChooser) *ListTabKinds {
	return &ListTabKinds{heads: heads, chooser: chooser}
}

// Execute answers by path, and leaves out a path with no kind of tab.
func (u *ListTabKinds) Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error) {
	kinds := map[string]string{}
	if len(paths) == 0 || u.heads == nil {
		return kinds, nil
	}
	heads, err := u.heads.GetHeads(ctx, v.ID, paths)
	if err != nil {
		return nil, err
	}
	for path, head := range heads {
		if kind := u.chooser.Execute(path, head.Type, head.Values); kind != "" {
			kinds[path] = kind
		}
	}
	return kinds, nil
}

// FolderListing is what a folder holds and the kind of tab each of its notes opens in.
type FolderListing struct {
	Folder
	TabKinds map[string]string
}

// ListFolder lists what a folder holds with the kind of tab each note in it opens in.
type ListFolder struct {
	entries  *GetFolderEntries
	tabKinds *ListTabKinds
}

func NewListFolder(entries *GetFolderEntries, tabKinds *ListTabKinds) *ListFolder {
	return &ListFolder{entries: entries, tabKinds: tabKinds}
}

func (u *ListFolder) Execute(ctx context.Context, v domain.Vault, path string) (FolderListing, error) {
	folder, err := u.entries.Execute(ctx, v, path)
	if err != nil {
		return FolderListing{}, err
	}
	var notes []string
	for _, one := range folder.Entries {
		if one.Kind == domain.KindNote {
			notes = append(notes, one.Path)
		}
	}
	kinds, err := u.tabKinds.Execute(ctx, v, notes)
	if err != nil {
		return FolderListing{}, err
	}
	return FolderListing{Folder: folder, TabKinds: kinds}, nil
}

// ListFileKinds lists what the vault holds at some paths with the kind of tab each note opens in.
type ListFileKinds struct {
	kinds   *GetFileKinds
	chooser TabKindChooser
}

func NewListFileKinds(kinds *GetFileKinds, chooser TabKindChooser) *ListFileKinds {
	return &ListFileKinds{kinds: kinds, chooser: chooser}
}

// Execute lists the files at the paths in the order given.
func (u *ListFileKinds) Execute(ctx context.Context, v domain.Vault, paths []string) (FolderListing, error) {
	folder, err := u.kinds.Execute(ctx, v, paths)
	if err != nil {
		return FolderListing{}, err
	}
	return FolderListing{Folder: folder, TabKinds: chooseTabKinds(folder, u.chooser)}, nil
}
