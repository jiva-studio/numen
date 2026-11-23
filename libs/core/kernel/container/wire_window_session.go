package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
)

// windowParts are what the use cases of one window are built over besides its state.
type windowParts struct {
	lookup    vaultport.VaultLookup
	indexer   *indexing.Worker
	embedding *indexing.Embedding

	recordOpen    vaultport.VaultRecorder
	importFiles   vault.Import
	diagnostics   *windowsession.Diagnostics
	embedderError error
	closeEmbedder func() error
	closeIndex    func() error
	shouldRebuild bool
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
	importFiles vault.ImportFiles
	showTab     windowsession.ShowTab
}

// windowSession is the use cases of the lifecycle of a window, the ones beside it, and the switch a person
// shows another vault through.
type windowSession struct {
	cases windowCases
	tools windowTools
	open  *windowsession.OpenVault
}

func newWindowSession(
	parts windowParts, pluginVault windowsession.PluginVaultPath, runner *windowsession.PassRunner, vaultDeps vaultWiring, state *editor.Session, page *editor.PageAccess,
) windowSession {
	identity := windowsession.NewCheckIdentity(createVaultIdentity(vaultDeps.Options))
	session := windowsession.NewSession(&state.Runtime, parts.recordOpen, parts.diagnostics, parts.embedderError, runner)
	guard := &windowsession.Guard{}
	flush := windowsession.NewPageFlush(page.Window, page.Writes)
	rebuild := windowsession.NewRebuildSession(guard, session)
	show := windowsession.NewShowVault(guard, session, identity, flush, &state.Notices.ChangeEvents)
	follow := windowsession.NewModelFollow(parts.embedding, parts.indexer, &state.Runtime, rebuild)
	switching := windowsession.NewSwitchVault(show, pluginVault, &state.Tabs, state.ShowOpenTabs)
	return windowSession{
		cases: windowCases{
			start:      windowsession.NewStartWindow(parts.diagnostics, parts.embedderError, follow, session, parts.shouldRebuild),
			show:       show,
			openPlugin: windowsession.NewOpenPluginVault(guard, parts.lookup, identity, page.Window, switching),
			flush:      windowsession.NewFlushPending(guard, flush),
			close: windowsession.NewCloseWindow(
				page.Gate, session,
				page.Media.ClosePlaying,
				func(context.Context) error { return parts.closeEmbedder() },
				func(context.Context) error { return parts.closeIndex() },
			),
		},
		tools: windowTools{
			importFiles: vault.NewImportFiles(
				parts.importFiles, page.Writes, page.Window, &state.Notices.ChangeEvents,
			),
			showTab: windowsession.NewShowTab(&state.Tabs, &state.Notices.TabIDs, 0),
		},
		open: windowsession.NewOpenVault(parts.lookup, switching),
	}
}
