package container

import (
	"context"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/startup"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// OpenEditor opens the registry and the cache, builds the editor window over them, opens the vault named, or
// the one shown last when none is named, and connects the plugins to the vault and the window. It returns the
// window and the configuration with the text versions the plugins share.
func OpenEditor(c Config, ctx context.Context, named string, out io.Writer) (*editor.Installation, Config, error) {
	start := startup.OpenEditor[vaultregistry.Ports, *vault.Index, *windowWiring]{
		OpenRegistry: c.Registry,
		OpenIndex:    func(ctx context.Context) (*vault.Index, error) { return OpenIndex(c, ctx) },
		OpenTexts: func(db *vault.Index) error {
			versions, err := newTextVersions(vaultOf(c), db)
			c.TextVersions = versions
			return err
		},
		OpenWindow: func(ctx context.Context, ports vaultregistry.Ports, db *vault.Index) (*windowWiring, error) {
			return openWindowOver(c, ctx, ports, db, named, out)
		},
		FollowVault: func(opened *windowWiring) {
			setPluginVaultPath(plugins(c), opened.inst.State.GetCurrentVault().Path)
		},
		InstallVault: func(ports vaultregistry.Ports, db *vault.Index, opened *windowWiring) error {
			return installPluginVault(plugins(c), ports, db, opened.inst.State.GetShownVault, PluginVaultHooks{
				Written: opened.inst.State.Writes.NoteWritten,
				Notes:   *opened.cases.Notes,
			})
		},
		ShareWindow: func(db *vault.Index, opened *windowWiring) {
			shareVaultList(c, opened)
			setPluginTexts(vaultOf(c))
			shareEditorWindow(c, db, opened)
		},
	}
	opened, err := start.Execute(ctx)
	if err != nil {
		return nil, c, err
	}
	return opened.inst, c, nil
}

// Flashcards is what the flashcards application closes when it ends: CloseVaults waits until nothing is still
// writing to the index, CloseIndex closes it, and Themes is why the themes could not be loaded, or nil.
type Flashcards struct {
	CloseVaults func()
	CloseIndex  func() error
	Themes      error
}

// OpenFlashcards opens the registry and the cache, binds the notes, the vault list, the vault indexer, the
// plugin vault, the change feed and the themes of the flashcards window to api, and tells notify what it
// cannot follow.
func OpenFlashcards(c Config,
	ctx context.Context, api *shell.API, out io.Writer, label string, notify func(message string),
) (Flashcards, error) {
	start := startup.OpenFlashcards[vaultregistry.Ports, *vault.Index, *vault.OpenVaults[*vault.Index]]{
		Label:        label,
		OpenRegistry: c.Registry,
		OpenIndex:    func(ctx context.Context) (*vault.Index, error) { return OpenIndex(c, ctx) },
		OpenVaults:   func(db *vault.Index) *vault.OpenVaults[*vault.Index] { return openVaults(c, db, out, label) },
		BindWindow: func(ctx context.Context, ports vaultregistry.Ports, db *vault.Index, vaults *vault.OpenVaults[*vault.Index]) {
			SetShellNotes(c, api, db, vaults.UpdateIndex)
			api.ListVaults = vaultregistry.NewList(ports.Vaults).Execute
			BindVaultIndexer(c, ctx, api, vaults)
		},
		InstallVault: func(ports vaultregistry.Ports, db *vault.Index) error {
			return installPluginVault(plugins(c), ports, db, getNoShownVault,
				PluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchMoves})
		},
		FollowChanges: func(ctx context.Context) (<-chan struct{}, error) { return vaultMoves(vaultOf(c), ctx) },
		ReportChanges: api.ReportChangesFrom,
		BindThemes: func(say func(string)) error {
			service, err := settingsThemes(settingsOf(c), say)
			api.Themes = service
			return err
		},
		Notify: notify,
	}
	opened, err := start.Execute(ctx)
	return Flashcards{CloseVaults: opened.Vaults.CloseAndWait, CloseIndex: opened.Index.Close, Themes: opened.Themes}, err
}

func getNoShownVault() (domain.Vault, bool) { return domain.Vault{}, false }

// FlushTimeout is how long a page may take to write its unsaved data when the vault closes.
func FlushTimeout() time.Duration { return windowsession.FlushTimeout }

// FollowVaultSwitch makes the window call after each time it puts a vault in its place, with the error
// of that attempt. The plugins follow the vault by themselves.
func FollowVaultSwitch(c Config, opened *editor.Installation, after func(v domain.Vault, err error)) {
	opened.State.OpenVault = startup.SwitchVault[*windowsession.PluginOpen]{
		Show:        opened.Show,
		FollowVault: func(path string) { setPluginVaultPath(plugins(c), path) },
		After:       after,
	}.Execute
}
