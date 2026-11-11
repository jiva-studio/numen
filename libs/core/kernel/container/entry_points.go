package container

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// OpenEditor opens the registry and the cache, builds the editor window over them, opens the vault named, or
// the one shown last when none is named, and connects the plugins to the vault and the window. It returns the
// window and the configuration with the text versions the plugins share. frame is the application window
// around the editor, or nil.
func OpenEditor(
	c Config, ctx context.Context, named string, out io.Writer, frame editor.Frame,
) (*editor.Installation, Config, error) {
	ports, err := c.registry()
	if err != nil {
		return nil, c, err
	}
	db, err := OpenIndex(c, ctx)
	if err != nil {
		return nil, c, err
	}
	return openEditorTexts(c, ctx, ports, db, WindowOpening{DB: db, Named: named, Out: out, Frame: frame})
}

func openEditorTexts(
	c Config, ctx context.Context, ports vaultregistry.Ports, db *Index, opening WindowOpening,
) (*editor.Installation, Config, error) {
	versions, err := newTextVersions(vaultOf(c), db)
	if err != nil {
		return nil, c, errors.Join(err, db.Close())
	}
	c.TextVersions = versions
	opened, err := openWindowWith(c, ctx, ports, opening)
	if err != nil {
		return nil, c, err
	}
	return connectEditor(c, ports, db, opened)
}

func connectEditor(c Config, ports vaultregistry.Ports, db *Index, opened *windowWiring) (*editor.Installation, Config, error) {
	state := opened.inst.State
	setPluginVaultPath(plugins(c), state.GetCurrentVault().Path)
	hooks := PluginVaultHooks{Written: state.Writes.NoteWritten, Notes: *opened.cases.Notes}
	if err := installPluginVault(plugins(c), ports, db, state.GetShownVault, hooks); err != nil {
		return nil, c, err
	}
	shareVaultList(c, opened)
	setPluginTexts(vaultOf(c))
	shareEditorWindow(c, db, opened)
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
	ports, err := c.registry()
	if err != nil {
		return Flashcards{}, err
	}
	db, err := OpenIndex(c, ctx)
	if err != nil {
		return Flashcards{}, err
	}
	return bindFlashcards(c, ctx, ports, db, api, flashcardsLog{out: out, label: label, notify: notify})
}

type flashcardsLog struct {
	out    io.Writer
	label  string
	notify func(message string)
}

func bindFlashcards(
	c Config, ctx context.Context, ports vaultregistry.Ports, db *Index, api *shell.API, log flashcardsLog,
) (Flashcards, error) {
	vaults := openVaults(c, db, log.out, log.label, api.ReportVaultRead)
	closers := Flashcards{CloseVaults: vaults.CloseAndWait, CloseIndex: db.Close}
	SetShellNotes(api, db)
	vaultList := vaultregistry.NewList(ports.Vaults)
	api.Vaults = &vaultList
	api.SetVaultIndexer(ctx, vaults.IndexVault)
	hooks := PluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchMoves}
	if err := installPluginVault(plugins(c), ports, db, getNoShownVault, hooks); err != nil {
		return closers, err
	}
	closers.Themes = followFlashcards(c, ctx, api, log.label, log.notify)
	return closers, nil
}

func followFlashcards(c Config, ctx context.Context, api *shell.API, label string, notify func(message string)) error {
	changes, err := vaultMoves(vaultOf(c), ctx)
	if err != nil {
		notify(label + ": the index is not being followed: " + err.Error())
	} else {
		api.ReportChangesFrom(ctx, changes)
	}
	service, themesErr := settingsThemes(settingsOf(c), Scales{}, func(response string) {
		if response != "" {
			notify("themes: " + response)
		}
	})
	api.Themes = service
	return themesErr
}

func getNoShownVault() (domain.Vault, bool) { return domain.Vault{}, false }

// FlushTimeout is how long a page may take to write its unsaved data when the vault closes.
func FlushTimeout() time.Duration { return windowsession.FlushTimeout }
