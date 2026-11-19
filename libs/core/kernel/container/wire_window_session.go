package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// windowParts are what the use cases of one window are built over besides its state.
type windowParts struct {
	indexer       *indexing.Worker
	embedding     *indexing.Embedding
	passes        windowsession.VaultPassesFunc
	recordOpen    vault.RecordOpen
	importFiles   vault.Import
	out           io.Writer
	embedderError error
	closeEmbedder func() error
	closeIndex    func() error
	shouldRebuild bool
	startFeed     func(ctx context.Context, v domain.Vault) func()
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

// windowSession is the use cases of the lifecycle of a window, the ones beside it, and the switch a person
// shows another vault through.
type windowSession struct {
	cases      windowCases
	tools      windowTools
	userSwitch windowsession.UserSwitch
}

func newWindowSession(parts windowParts,
	pluginVault windowsession.PluginVaultPath, triggers windowsession.TriggerSource, vaultDeps vaultWiring, state *editor.Session, page *editor.PageAccess, ctx context.Context,
) windowSession {
	diag := windowsession.NewDiagnostics(parts.out, page.Window)
	runner := windowsession.NewPassRunner(windowsession.PassRunnerConfig{
		Failures: &state.Runtime, Diag: diag, Closing: ctx.Done(), NewSession: newIndexSession(parts.indexer),
		Reporter: windowsession.NewChangeReporter(&state.Notices.ChangeEvents, parts.indexer, getTextVersions(vaultDeps.TextVersions), diag),
		Passes:   parts.passes, Triggers: triggers, StartFeed: parts.startFeed,
		Report: editor.ReportVault(vaultDeps.RecordLog),
	})
	identity := vault.NewCheckIdentity(vaultIdentity(vaultDeps.Options))
	session := windowsession.NewSession(&state.Runtime, parts.recordOpen, diag, parts.embedderError, runner)
	guard := &windowsession.Guard{}
	flush := windowsession.NewPageFlush(page.Window, page.Writes)
	rebuild := windowsession.NewRebuildSession(guard, session)
	show := windowsession.NewShowVault(guard, session, identity, flush, &state.Notices.ChangeEvents)
	follow := windowsession.NewModelFollow(parts.embedding, parts.indexer, &state.Runtime, rebuild)
	switching := windowsession.NewSwitchVault(show, pluginVault, &state.Tabs, state.ShowOpenTabs)
	return windowSession{
		cases: windowCases{
			start:      windowsession.NewStartWindow(diag, parts.embedderError, follow, session, parts.shouldRebuild),
			show:       show,
			openPlugin: windowsession.NewOpenPluginVault(guard, identity, page.Window, switching),
			flush:      windowsession.NewFlushPending(guard, flush),
			close: windowsession.NewCloseWindow(
				page.Gate, session,
				page.Media.ClosePlaying,
				func(context.Context) error { return parts.closeEmbedder() },
				func(context.Context) error { return parts.closeIndex() },
			),
		},
		tools: windowTools{
			importFiles: windowsession.NewImportFiles(
				parts.importFiles, page.Writes, page.Window, &state.Notices.ChangeEvents,
			),
			showTab: windowsession.NewShowTab(&state.Tabs, &state.Notices.TabIDs, 0),
		},
		userSwitch: windowsession.NewUserSwitch(switching),
	}
}
