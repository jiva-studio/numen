// Package file serves a window's FileService: the tree of the shown vault, the folders in it, and
// the moves and removals that change it.
package file

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Vault names the vault the window shows. A window showing none returns a vault with no ID.
type Vault interface {
	GetCurrentVault() domain.Vault
}

// Writes counts the writes the window waits for before it closes.
type Writes interface {
	BeginWrite() bool
	FinishWrite()
}

// Session is the window a handler serves: the vault it shows and the writes it waits for.
type Session interface {
	Vault
	Writes
}

// Index reports whether the index followed a write.
type Index interface {
	IsIndexOutdated(err error) bool
}

// Tree tells the views drawing the vault which paths changed.
type Tree interface {
	PublishChange(paths ...string)
}

// Events tells the plugins of the window that something happened.
type Events interface {
	EmitFileEvent(event string)
}

var (
	errClosing = broadcast.ErrClosing
	errNoVault = errors.New("the window has no vault")
)

func getShownVault(window Vault) (domain.Vault, error) {
	v := window.GetCurrentVault()
	if v.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	return v, nil
}

// Handler answers FileService. Each question it answers is held by one of the parts it is made of.
type Handler struct {
	*Folders
	*Changes
}

// NewHandler is the handler of FileService made of its parts.
func NewHandler(folders *Folders, changes *Changes) *Handler {
	return &Handler{Folders: folders, Changes: changes}
}

// Folders lists what the folders of the vault hold and makes folders.
type Folders struct {
	vault        Vault
	writes       Writes
	events       Events
	listFolder   *file.ListFolder
	listKinds    *file.ListFileKinds
	createFolder *file.CreateFolder
}

// NewFolders is the part of FileService that lists and makes folders.
func NewFolders(
	session Session, events Events, folder *file.ListFolder, kinds *file.ListFileKinds, create *file.CreateFolder,
) *Folders {
	return &Folders{vault: session, writes: session, events: events, listFolder: folder, listKinds: kinds, createFolder: create}
}

// Changes moves and removes the files of the vault.
type Changes struct {
	vault      Vault
	writes     Writes
	index      Index
	tree       Tree
	move       *vaults.Move
	remove     *note.Remove
	emptyTrash *note.EmptyTrash
}

// NewChanges is the part of FileService that moves and removes files.
func NewChanges(
	session Session, index Index, tree Tree,
	move *vaults.Move, remove *note.Remove, emptyTrash *note.EmptyTrash,
) *Changes {
	return &Changes{
		vault: session, writes: session, index: index, tree: tree, move: move, remove: remove, emptyTrash: emptyTrash,
	}
}
