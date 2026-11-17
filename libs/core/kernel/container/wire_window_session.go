package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
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
	frontmatter   *indexnote.FrontmatterChanges
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

func newWindowSession(c Config, a pluginsWiring, state *editor.State, ctx context.Context, parts windowParts) windowSession {
	diag := windowsession.NewDiagnostics(parts.out, state.Window)
	runner := windowsession.NewPassRunner(windowsession.PassRunnerConfig{
		Failures: &state.Runtime, Diag: diag, Closing: ctx.Done(), NewSession: newIndexSession(parts.indexer),
		Reporter: windowsession.NewChangeReporter(&state.Notices.ChangeEvents, parts.indexer, getTextVersions(a), diag),
		Passes:   parts.passes, Triggers: getTriggerSource(a, parts.frontmatter), StartFeed: parts.startFeed,
	})
	identity := vault.NewCheckIdentity(vaultIdentity(a.vault))
	session := windowsession.NewSession(&state.Runtime, parts.recordOpen, diag, parts.embedderError, runner)
	guard := &windowsession.Guard{}
	flush := windowsession.NewPageFlush(state.Window, state.Writes)
	rebuild := windowsession.NewRebuildSession(guard, session)
	show := windowsession.NewShowVault(guard, session, identity, flush, &state.Notices.ChangeEvents)
	follow := windowsession.NewModelFollow(parts.embedding, parts.indexer, &state.Runtime, rebuild)
	switching := windowsession.NewSwitchVault(show, a.PluginHost.Runtime, state)
	return windowSession{
		cases: windowCases{
			start:      windowsession.NewStartWindow(diag, parts.embedderError, follow, session, c.ShouldRebuildIndex),
			show:       show,
			openPlugin: windowsession.NewOpenPluginVault(guard, identity, state.Window, switching),
			flush:      windowsession.NewFlushPending(guard, flush),
			close: windowsession.NewCloseWindow(
				state.Gate, session,
				state.ClosePlaying,
				func(context.Context) error { return parts.closeEmbedder() },
				func(context.Context) error { return parts.closeIndex() },
			),
		},
		tools: windowTools{
			importFiles: windowsession.NewImportFiles(
				parts.importFiles, state.Writes, state.Window, &state.Notices.ChangeEvents,
			),
			showTab: windowsession.NewShowTab(&state.Tabs, &state.Notices.TabIDs, 0),
		},
		userSwitch: windowsession.NewUserSwitch(switching),
	}
}
