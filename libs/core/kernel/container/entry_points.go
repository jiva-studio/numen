package container

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// OpenEditor opens the registry and the cache, builds the editor window over them, opens the vault named, or
// the one shown last when none is named, and connects the plugins to the vault and the window. It returns the
// window and the configuration with the text versions the plugins share.
func OpenEditor(c Config, ctx context.Context, named string, out io.Writer) (*editor.Installation, Config, error) {
	registry, err := c.Registry()
	if err != nil {
		return nil, c, err
	}
	db, err := OpenIndex(c, ctx)
	if err != nil {
		return nil, c, err
	}
	versions, err := NewTextVersions(c, db)
	if err != nil {
		return nil, c, errors.Join(err, db.Close())
	}
	c.TextVersions = versions

	opened, err := OpenWindowOver(c, ctx, registry, db, named, out)
	if err != nil {
		return nil, c, err
	}
	SetPluginVaultPath(c, opened.State.GetCurrentVault().Path)
	getShownVault := func() (domain.Vault, bool) {
		current := opened.State.GetCurrentVault()
		return current, current.Path != ""
	}
	err = InstallPluginVault(c, registry, db, getShownVault, PluginVaultHooks{
		Written: func() {
			if wake := opened.State.Writes.NotifyNoteWritten; wake != nil {
				wake()
			}
		},
		Notes: opened.Notes(),
	})
	if err != nil {
		return nil, c, err
	}
	SetEditorPluginVaultList(c, opened)
	SetPluginTexts(c)
	SetEditorPluginWindow(c, db, opened)
	return opened, c, nil
}

// Flashcards is what the flashcards application closes when it ends: CloseVaults waits until nothing is still
// writing to the index, CloseIndex closes it, and Themes is why the themes could not be loaded, or nil.
type Flashcards struct {
	CloseVaults func()
	CloseIndex func() error
	Themes error
}

// OpenFlashcards opens the registry and the cache, binds the notes, the vault list, the vault indexer, the
// plugin vault, the change feed and the themes of the flashcards window to api, and reports what it cannot
// follow on errOut.
func OpenFlashcards(c Config, ctx context.Context, api *shell.API, errOut io.Writer, label string) (Flashcards, error) {
	list, err := c.Registry()
	if err != nil {
		return Flashcards{}, err
	}
	// The editor's window may write the same index at the same time.
	db, err := OpenIndex(c, ctx)
	if err != nil {
		return Flashcards{}, err
	}
	vaults := OpenVaults(c, db, errOut, label)
	SetShellNotes(c, api, db, vaults.UpdateIndex)
	api.ListVaults = GetVaultList(c, list).Execute
	BindVaultIndexer(c, ctx, api, vaults)
	started := Flashcards{CloseVaults: vaults.CloseAndWait, CloseIndex: db.Close}
	noCurrentVault := func() (domain.Vault, bool) { return domain.Vault{}, false }
	hooks := PluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchMoves}
	if err := InstallPluginVault(c, list, db, noCurrentVault, hooks); err != nil {
		return started, err
	}
	if moves, err := Moves(c, ctx); err != nil {
		_, _ = fmt.Fprintln(errOut, label+": the index is not being followed:", err)
	} else {
		api.ReportChangesFrom(ctx, moves)
	}
	started.Themes = SetShellThemes(c, api, func(response string) {
		if response != "" {
			_, _ = fmt.Fprintln(errOut, "themes:", response)
		}
	})
	return started, nil
}
