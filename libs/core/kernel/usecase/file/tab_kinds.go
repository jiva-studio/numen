package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// TabKindChooser gives a note the kind of tab it opens in, or an empty kind where none applies.
type TabKindChooser interface {
	Execute(path string, noteType domain.NoteType, properties map[string]any) string
}

// NoTabKinds is the chooser of a build with no plugin that declares a tab kind: no note has one.
type NoTabKinds struct{}

// Execute gives no kind.
func (NoTabKinds) Execute(string, domain.NoteType, map[string]any) string { return "" }

func chooseTabKinds(folder Folder, chooser TabKindChooser) map[string]string {
	kinds := map[string]string{}
	for path, properties := range folder.Properties {
		if kind := chooser.Execute(path, folder.Types[path], properties); kind != "" {
			kinds[path] = kind
		}
	}
	return kinds
}

// ListTabKinds finds the kind of tab each of the files at some paths opens in.
type ListTabKinds struct {
	kinds   *GetFileKinds
	chooser TabKindChooser
}

// NewListTabKinds creates the listing over the reading of file kinds and the chooser of tab kinds.
func NewListTabKinds(kinds *GetFileKinds, chooser TabKindChooser) *ListTabKinds {
	return &ListTabKinds{kinds: kinds, chooser: chooser}
}

// Execute answers by path, and leaves out a path with no kind of tab.
func (u *ListTabKinds) Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error) {
	if _, none := u.chooser.(NoTabKinds); none || len(paths) == 0 {
		return map[string]string{}, nil
	}
	listing, err := u.kinds.Execute(ctx, v, paths)
	if err != nil {
		return nil, err
	}
	return chooseTabKinds(listing, u.chooser), nil
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

// NewListFolder creates the listing over the reading of a folder and the listing of tab kinds.
func NewListFolder(entries *GetFolderEntries, tabKinds *ListTabKinds) *ListFolder {
	return &ListFolder{entries: entries, tabKinds: tabKinds}
}

// Execute lists the folder at the path.
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

// NewListFileKinds creates the listing over the reading of file kinds and the chooser of tab kinds.
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
