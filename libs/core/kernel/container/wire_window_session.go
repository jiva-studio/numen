package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
)

// windowParts are what the use cases of one window are built over besides its state.
type windowParts struct {
	indexer       *indexing.Worker
	embedding     *indexing.Embedding
	moves         *vault.Scenarios
	current       *vault.CurrentRefresh
	out           io.Writer
	embedderError error
	closeEmbedder func() error
}

// windowCases are the use cases of one window.
type windowCases struct {
	start       windowsession.StartWindow
	show        windowsession.ShowVault
	openPlugin  windowsession.OpenPluginVault
	flush       windowsession.FlushPending
	close       windowsession.CloseWindow
	importFiles windowsession.ImportFiles
	showTab     windowsession.ShowTab
}

func windowSession(w *windowWiring, state *editor.State, ctx context.Context, parts windowParts) windowCases {
	diag := windowsession.NewDiagnostics(parts.out, state.Window)
	runner := windowsession.NewPassRunner(
		&state.Runtime, diag, ctx.Done(), newIndexSession(parts.indexer), getTextVersions(w.cfg),
		&state.Notices.ChangeEvents, wrapPasses(w, startVault(w.cfg, w.db, parts.current)), getTriggerSource(w.cfg),
		parts.indexer,
	)
	identity := vault.NewCheckIdentity(vaultIdentity(vaultOf(w.cfg)))
	session := windowsession.NewSession(&state.Runtime, parts.moves.RecordOpen, diag, parts.embedderError, runner)
	guard := &windowsession.Guard{}
	rebuild := windowsession.NewRebuildSession(&state.Runtime, guard, session)
	return windowCases{
		start: windowsession.NewStartWindow(
			diag, parts.embedderError, parts.embedding, parts.indexer, &state.Runtime, session, rebuild, w.cfg.ShouldRebuildIndex,
		),
		show: windowsession.NewShowVault(
			&state.Runtime, guard, session, identity, state.Window, state.Writes, &state.Notices.ChangeEvents,
		),
		openPlugin: windowsession.NewOpenPluginVault(guard, identity, state.Window),
		flush:      windowsession.NewFlushPending(guard, state.Window, state.Writes),
		close: windowsession.NewCloseWindow(
			state.Gate, session,
			windowsession.CloseFunc(state.ClosePlaying),
			windowsession.CloseFunc(func(context.Context) error { return parts.closeEmbedder() }),
			windowsession.CloseFunc(func(context.Context) error { return w.db.Close() }),
		),
		importFiles: windowsession.NewImportFiles(
			parts.moves.Tree.Import, state.Writes, state.Window, &state.Notices.ChangeEvents,
		),
		showTab: windowsession.NewShowTab(&state.Tabs, &state.Notices.TabIDs, 0),
	}
}
