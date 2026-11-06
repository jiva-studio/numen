package vault

import (
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/core/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/usecase/vault/registry"
)

type (
	ChooseFolder   = picker.ChooseFolder
	PickFolder     = picker.PickFolder
	PickedFolder   = picker.PickedFolder
	FolderCheck    = check.FolderCheck
	GetKnownVaults = registry.GetKnownVaults
	KnownVault     = registry.KnownVault
	GetSummary     = registry.GetSummary
	GetVaultToShow = registry.GetVaultToShow
	Find           = registry.Find
	List           = registry.List
	UnknownError   = registry.UnknownError
)

var (
	ErrNoTrash = picker.ErrNoTrash
	ErrNoName  = registry.ErrNoName
	ErrShowing = registry.ErrShowing
	ErrUnknown = registry.ErrUnknown
)

func NewChooseFolder(dialog machineport.FolderDialog) picker.ChooseFolder {
	return picker.NewChooseFolder(dialog)
}

func NewPickFolder(dialog machineport.FolderDialog) picker.PickFolder {
	return picker.NewPickFolder(dialog)
}

func NewFolderCheck(readers vaultport.VaultReaders) check.FolderCheck {
	return check.NewFolderCheck(readers)
}

func NewGetKnownVaults(p0 vaultport.VaultRegistry, readers vaultport.VaultReaders) registry.GetKnownVaults {
	return registry.NewGetKnownVaults(p0, readers)
}

func NewGetSummary(queries noteport.NoteQueries) *registry.GetSummary {
	return registry.NewGetSummary(queries)
}

func NewGetVaultToShow(p0 vaultport.VaultRegistry) registry.GetVaultToShow {
	return registry.NewGetVaultToShow(p0)
}

func NewFind(p0 vaultport.VaultRegistry) registry.Find { return registry.NewFind(p0) }

func NewList(p0 vaultport.VaultRegistry) registry.List { return registry.NewList(p0) }

func IsFolderDialogUnavailable(err error) bool { return picker.IsFolderDialogUnavailable(err) }
