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
	moves         *VaultScenarios
	current       *vault.CurrentRefresh
	out           io.Writer
	embedderError error
	closeEmbedder func() error
}

// windowCases are the use cases that run a window from its start to its close.
type windowCases struct {
	start      windowsession.StartWindow
	show       windowsession.ShowVault
	openPlugin windowsession.OpenPluginVault
	flush      windowsession.FlushPending
	close      windowsession.CloseWindow
}

// windowTools are the use cases a window offers beside its lifecycle.
type windowTools struct {
	importFiles windowsession.ImportFiles
	showTab     windowsession.ShowTab
}

func windowSession(w *windowWiring, state *editor.State, ctx context.Context, parts windowParts) (windowCases, windowTools) {
	diag := windowsession.NewDiagnostics(parts.out, state.Window)
	runner := windowsession.NewPassRunner(windowsession.PassRunnerConfig{
		Failures: &state.Runtime, Diag: diag, Closing: ctx.Done(), NewSession: newIndexSession(parts.indexer),
		Reporter: windowsession.NewChangeReporter(&state.Notices.ChangeEvents, parts.indexer, getTextVersions(w.cfg), diag),
		Passes:   wrapPasses(w, startVault(w.cfg, w.db, parts.current)), Triggers: getTriggerSource(w.cfg),
	})
	identity := vault.NewCheckIdentity(vaultIdentity(vaultOf(w.cfg)))
	session := windowsession.NewSession(&state.Runtime, parts.moves.RecordOpen, diag, parts.embedderError, runner)
	guard := &windowsession.Guard{}
	flush := windowsession.NewPageFlush(state.Window, state.Writes)
	rebuild := windowsession.NewRebuildSession(guard, session)
	show := windowsession.NewShowVault(guard, session, identity, flush, &state.Notices.ChangeEvents)
	follow := windowsession.NewModelFollow(parts.embedding, parts.indexer, &state.Runtime, rebuild)
	state.Switch = windowsession.NewSwitchVault(show, plugins(w.cfg).PluginHost.Runtime, state)
	cases := windowCases{
		start:      windowsession.NewStartWindow(diag, parts.embedderError, follow, session, w.cfg.ShouldRebuildIndex),
		show:       show,
		openPlugin: windowsession.NewOpenPluginVault(guard, identity, state.Window, state.GetVaultSwitch()),
		flush:      windowsession.NewFlushPending(guard, flush),
		close: windowsession.NewCloseWindow(
			state.Gate, session,
			windowsession.CloseFunc(state.ClosePlaying),
			windowsession.CloseFunc(func(context.Context) error { return parts.closeEmbedder() }),
			windowsession.CloseFunc(func(context.Context) error { return w.db.Close() }),
		),
	}
	tools := windowTools{
		importFiles: windowsession.NewImportFiles(
			parts.moves.Tree.Import, state.Writes, state.Window, &state.Notices.ChangeEvents,
		),
		showTab: windowsession.NewShowTab(&state.Tabs, &state.Notices.TabIDs, 0),
	}
	return cases, tools
}
