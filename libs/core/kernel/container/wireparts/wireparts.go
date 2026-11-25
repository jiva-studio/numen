// Package wireparts assembles the parts of the window wiring that need nothing of the container beyond what
// they are given.
package wireparts

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowkeyfile "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/keyfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/remotevault"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfolder"
	machineadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	keyfileusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// NewKeyFileRoute creates the route of the key file service over the key file. Each call goes through counted.
func NewKeyFileRoute(file keyfileusecase.File, counted connect.HandlerOption) editor.Route {
	handler := windowkeyfile.NewHandler(windowkeyfile.UseCases{
		GetState: keyfileusecase.NewGetState(file),
		Choose:   keyfileusecase.NewChoose(file, settingsadapter.WriteKeyFileMode),
		Unlock:   keyfileusecase.NewUnlock(file),
	})
	return editor.NewRoute(windowkeyfile.NewRoute(handler, counted))
}

// RemoteVaults is what the route of the remote vaults is made of.
type RemoteVaults struct {
	Ports     registry.Ports
	Add       vault.Add
	Provider  vault.RemoteVaultProvider
	NewFolder vault.NewVaultsFolder
	Now       machineport.Clock
}

// NewRemoteVaultRoute creates the route of the remote vault service. Each call goes through counted.
func NewRemoteVaultRoute(parts RemoteVaults, counted connect.HandlerOption) editor.Route {
	folders := vaultfolder.Folders{}
	user := machineadapter.UserFolders{}
	handler := remotevault.NewHandler(
		vault.NewListRemoteVaults(parts.Provider),
		vault.NewSuggestVaultPath(parts.NewFolder, user, folders),
		vault.NewOpenRemoteVault(parts.Ports.Vaults, parts.Ports.Remover, folders, user, parts.Add, parts.Provider, parts.Now),
	)
	return editor.NewRoute(remotevault.NewRoute(handler, counted))
}

// NewNewVaultsFolder creates the reading of the setting Folder for new vaults. It reads the settings on
// each call and reports a failed read to handle.
func NewNewVaultsFolder(read func() (settingsadapter.File, error), handle func(error)) vault.NewVaultsFolder {
	return func() string {
		target, err := read()
		if err != nil {
			handle(fmt.Errorf("read settings: %w", err))
		}
		folder, _ := target.GetValue(settingsdomain.AppOwner, "vaults", "folder")
		text, _ := folder.(string)
		return text
	}
}

// NewIndexSession creates the starter of the index session of a vault over the worker.
func NewIndexSession(worker *indexing.Worker) func(domain.Vault) windowsession.IndexSession {
	return func(v domain.Vault) windowsession.IndexSession {
		session := worker.NewSession(v)
		return windowsession.IndexSession{
			Run: func(ctx context.Context, rebuild bool, read windowsession.ReadNotes) {
				session.Run(ctx, rebuild, read)
			},
			ReportVersionsChanged: session.ReportVersionsChanged,
		}
	}
}
