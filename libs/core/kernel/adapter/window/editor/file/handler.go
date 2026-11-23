// Package file serves a window's FileService: the tree of the shown vault, the folders in it, and
// the moves and removals that change it.
package file

import (
	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Vault names the vault the window shows. A window showing none returns a vault with no ID.
type Vault interface {
	GetCurrentVault() domain.Vault
}

// Tree tells the views drawing the vault which paths changed.
type Tree interface {
	PublishChange(paths ...string)
}

var errClosing = broadcast.ErrClosing

func getShownVault(window Vault) (domain.Vault, error) {
	v := window.GetCurrentVault()
	if v.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, wire.ErrNoVault)
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
	listFolder   *file.ListFolder
	listKinds    *file.ListFileKinds
	createFolder *file.CreateFolder
}

// NewFolders is the part of FileService that lists and makes folders.
func NewFolders(
	window Vault, folder *file.ListFolder, kinds *file.ListFileKinds, create *file.CreateFolder,
) *Folders {
	return &Folders{vault: window, listFolder: folder, listKinds: kinds, createFolder: create}
}

// Changes moves and removes the files of the vault.
type Changes struct {
	vault      Vault
	tree       Tree
	move       *note.MoveFile
	remove     *note.RemoveFile
	emptyTrash *note.EmptyTrash
}

// NewChanges is the part of FileService that moves and removes files.
func NewChanges(
	window Vault, tree Tree,
	move *note.MoveFile, remove *note.RemoveFile, emptyTrash *note.EmptyTrash,
) *Changes {
	return &Changes{vault: window, tree: tree, move: move, remove: remove, emptyTrash: emptyTrash}
}
