package editor

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/assembly"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// The use cases a handler holds, for the tests that wrap one in a fake.

func (s *Services) NoteCreate() *note.Create { return s.Note.create }
func (s *Services) NoteWrite() *note.Write   { return s.Note.write }
func (s *Services) NoteRename() *note.Rename { return s.Note.rename }
func (s *Services) FileRemove() *note.Remove { return s.File.remove }
func (s *Services) FileMove() *vaults.Move   { return s.File.move }

func (s *Services) SetFileMove(move *vaults.Move)  { s.File.move = move }
func (s *Services) SetNoteWrite(write *note.Write) { s.Note.write = write }

func (s *Services) SetTabKindChoice(choose *tabkind.Choose) { s.File.chooseTabKind = choose }

// NewWireModels is the models as the wire carries them.
func NewWireModels(target []settingsdomain.Model) []*v1.Model { return settings.NewWireModels(target) }

// NewAssembly is the window's assembly over a configuration, bound the way an
// application binds one. The index it opens is the window's to close.
func NewAssembly(t *testing.T, cfg container.Config) assembly.Assembly {
	t.Helper()

	made, err := cfg.GetEditorAssembly(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// watched is the window's assembly with the walk and the watch a test drives,
// in place of the ones an installation is bound to. Everything else is the
// installation's own.
type watched struct {
	container.EditorAssembly

	cfg     container.Config
	db      *container.Index
	readers vaultport.VaultReaders
	watcher vaultport.VaultWatcher
}

func newWatched(
	cfg container.Config, db *container.Index,
	readers vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) watched {
	return watched{
		EditorAssembly: container.NewEditorAssembly(cfg, nil, db),
		cfg:            cfg,
		db:             db,
		readers:        readers,
		watcher:        watcher,
	}
}

func (w watched) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError window.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
) {
	request := w.cfg
	request.ShouldRebuildIndex = rebuild

	opening := request.VaultOpenerWith(w.db, w.readers, w.watcher)
	opening.ShouldRebuild = rebuild
	opening.ErrorHandler = handleError
	if notified != nil {
		opening.Told = func(m vaults.VaultChanges) { notified(m.Paths, m.Assets, m.ShouldReload) }
	}

	open := opening.Begin(ctx, v)
	return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
}
