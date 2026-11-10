package container

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
)

// UnknownVaultError is a vault named at start that the registry does not hold.
type UnknownVaultError = registry.UnknownError

// windowWiring is what the wire_window_<service>.go files share while one editor window is built.
// The state, the scenarios of notes and vaults and the installation are the window's own, and each
// service is handed the ones it uses.
type windowWiring struct {
	cfg    Config
	ports  registry.Ports
	db     *vault.Index
	notes  *note.Scenarios
	vaults *vault.Scenarios

	inst   *editor.Installation
	files  *filewindow.Handler
	server *editor.FileServer
	wrap   func(editor.StartVaultFunc) editor.StartVaultFunc
}

// OpenWindow opens the registry and the cache, builds the services of the editor window over
// them, and opens the vault named, or the one shown last when none is named.
func (c Config) OpenWindow(ctx context.Context, named string, out io.Writer) (*editor.Installation, error) {
	ports, err := c.Registry()
	if err != nil {
		return nil, err
	}
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return nil, err
	}
	return c.OpenWindowOver(ctx, ports, db, named, out)
}

// OpenWindowOver is OpenWindow over a registry and a cache already open, for a caller that opened
// them itself. The window closes the cache.
func (c Config) OpenWindowOver(
	ctx context.Context, ports registry.Ports, db *vault.Index, named string, out io.Writer,
) (*editor.Installation, error) {
	return c.OpenWindowWith(ctx, ports, db, named, out, nil)
}

// OpenWindowWith is OpenWindowOver with the walk of the vault put through wrap, for a caller
// that holds one back or drives it. A nil wrap leaves it as it is.
func (c Config) OpenWindowWith(
	ctx context.Context, ports registry.Ports, db *vault.Index, named string, out io.Writer,
	wrap func(editor.StartVaultFunc) editor.StartVaultFunc,
) (*editor.Installation, error) {
	w := &windowWiring{cfg: c, ports: ports, db: db, wrap: wrap}
	return openWindow(ctx, w, named, out)
}

func openWindow(ctx context.Context, w *windowWiring, named string, out io.Writer) (*editor.Installation, error) {
	first, err := registry.NewGetVaultToShow(w.ports.Lookup, w.ports.Vaults, w.ports.Last).Execute(named)
	if err != nil {
		return nil, errors.Join(err, w.db.Close())
	}

	// One task list serves every operation and the window that displays it.
	// A request is one task regardless of how it was issued. The list is created
	// before any component that reports into it.
	tasks := run.NewTasks()
	models := &run.Lock{}
	w.cfg.ReportPluginFailures(tasks)
	w.cfg.ReportDependencies(tasks)
	w.cfg.SetPluginWork(tasks, models)

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives.
	embedder, asking, closeEmbedder, why := w.cfg.Embedders(ctx, tasks)
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}
	session := getWindowSession(w.cfg, w.db)
	if w.wrap != nil {
		session.StartVault = w.wrap(session.StartVault)
	}
	inst := editor.NewInstallation(editor.Setup{
		Tasks:         tasks,
		Out:           out,
		CheckIdentity: vault.NewCheckIdentity(w.cfg.VaultIdentity()),
		Embedder:      embedder,
		Asking:        asking,
		EmbedderError: why,
		StopEmbedder:  closeEmbedder,
		ReadVault:     getReadVault(w.cfg, w.db),
		CloseIndex:    w.db.Close,
		Progress:      w.db.Progress(),
		EmitEvent:     func(event string) { emitPluginEvent(w.cfg.plugins(), event) },
		IsRebuilding:  w.cfg.ShouldRebuildIndex,
		Closing:       ctx.Done(),
	}, session)
	w.inst = inst

	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	notes := w.cfg.Notes(w.db, vault.NewUpdateIndex(inst.Refresh).Execute).FollowMoves(inst.State.GetWindow())
	w.notes = &notes
	moves := w.cfg.Vaults(w.ports, w.db, w.notes.Move)
	w.vaults = &moves
	inst.UseScenarios(w.notes, w.vaults)

	if err := mountServices(w); err != nil {
		return nil, errors.Join(err, closeEmbedder(), w.db.Close())
	}
	if err := inst.Start(ctx, first, w.server); err != nil {
		return nil, errors.Join(err, closeEmbedder(), w.db.Close())
	}
	return inst, nil
}

func mountServices(w *windowWiring) error {
	mountVault(w)
	if err := mountWorkspace(w); err != nil {
		return err
	}
	mountWindowState(w)
	mountFile(w)
	mountNote(w)
	mountSearch(w)
	mountVaults(w)
	mountSettings(w)
	mountMedia(w)
	mountAgent(w)
	mountCommand(w)
	mountText(w)
	mountPlugin(w)
	mountCatalog(w)
	return nil
}
