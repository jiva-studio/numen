package vault

import (
	"github.com/jiva-studio/numen/libs/core/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/core/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/usecase/vault/registry"
)

type (
	ChooseFolder   = picker.ChooseFolder
	PickFolder     = picker.PickFolder
	PickedFolder   = picker.PickedFolder
	CheckRoot      = check.CheckRoot
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
	NewChooseFolder   = picker.NewChooseFolder
	NewPickFolder     = picker.NewPickFolder
	NewCheckRoot      = check.NewCheckRoot
	NewFolderCheck    = check.NewFolderCheck
	NewGetKnownVaults = registry.NewGetKnownVaults
	NewGetSummary     = registry.NewGetSummary
	NewGetVaultToShow = registry.NewGetVaultToShow
	NewFind           = registry.NewFind
	NewList           = registry.NewList

	IsFolderDialogUnavailable = picker.IsFolderDialogUnavailable

	ErrNoTrash = picker.ErrNoTrash
	ErrNoName  = registry.ErrNoName
	ErrShowing = registry.ErrShowing
	ErrUnknown = registry.ErrUnknown
)
