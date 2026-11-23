package editor

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// PathDialog is the native dialog the application gives the window to choose a file or a folder. It
// answers the folder a person chose, and false when none was chosen.
type PathDialog interface {
	ChooseFolder(ctx context.Context, title, startingAt string) (string, bool, error)
	ChooseFile(ctx context.Context, title, startingAt string, accept []string) (string, bool, error)
}

// Starter opens the first vault of the window and the passes behind it.
type Starter interface {
	Execute(ctx context.Context, first domain.Vault)
}

// Shower puts a vault in the window. `open` is the plugin open whose lock the switch shares, nil for a
// user switch.
type Shower interface {
	Execute(ctx context.Context, v domain.Vault, open *windowsession.PluginOpen) error
}

// Flusher has every page write what only it holds, and reports whether none holds work that needs
// user confirmation.
type Flusher interface {
	Execute(ctx context.Context) bool
}

// Closer stops the passes behind the vault and closes what the window holds open.
type Closer interface {
	Execute(ctx context.Context) error
}

// Importer copies files dropped onto the window into a folder of the vault.
type Importer interface {
	Execute(ctx context.Context, v domain.Vault, into string, paths []string)
}

// Lifecycle switches, flushes and closes an open window.
type Lifecycle struct {
	Show  Shower
	Flush Flusher
	Close Closer
}

// Installation is a window put together and running: the pieces the services
// of its routes and anything else working the same vault need.
// What runs behind the vault is the lifecycle; the installation maps the
// calls of the application onto it.
type Installation struct {
	lifecycle Lifecycle
	imports   Importer
	pages     pageSet
}

// errNoVault is returned for a vault query on a window with no vault open.
var errNoVault = errors.New("the window has no vault")

// Option sets what a window is put together with besides its state.
type Option func(*Installation)

// WithThemes sets the themes the window puts on its pages. A window put together without them serves
// pages with no theme.
func WithThemes(themes numenv1connect.ThemeServiceHandler) Option {
	return func(o *Installation) { o.pages.themes = themes }
}

// WithPlugins sets where the calls for the services plugins serve are forwarded, and where the files
// of a plugin are read from.
func WithPlugins(services PluginServices, dir func(id string) (string, bool)) Option {
	return func(o *Installation) { o.pages.plugins, o.pages.pluginDir = services, dir }
}

// NewInstallation makes the installation of a window over its state, the lifecycle of what it shows,
// the import of dropped files and the opening of the vault files served as assets. Nothing is served and no
// vault is open until Start.
func NewInstallation(page *PageAccess, lifecycle Lifecycle, imports Importer, assets *file.OpenVaultFile, options ...Option) *Installation {
	o := &Installation{
		lifecycle: lifecycle,
		imports:   imports,
		pages:     pageSet{window: page, assets: assets},
	}
	for _, option := range options {
		option(o)
	}
	return o
}

// Mount adds the services the window answers through.
func (o *Installation) Mount(routes ...Route) { o.pages.routes = append(o.pages.routes, routes...) }

// GetThemes returns the themes the window offers.
func (o *Installation) GetThemes() numenv1connect.ThemeServiceHandler { return o.pages.themes }

// Start opens the first vault, follows the model of the embedder and serves
// recordings over the loopback socket.
func (o *Installation) Start(ctx context.Context, first domain.Vault, files *FileServer, starter Starter, out io.Writer) error {
	starter.Execute(ctx, first)

	// Recordings are served over a dedicated socket, opened after every
	// dependency it answers through is in place. When the socket cannot be opened,
	// the player has no address.
	onStop := func(why error) {
		writeDiagnostic(o.pages.window.Window.AddTask, out, "recordings will no longer play: %v\n", why)
	}
	if playing, why := Listen(ctx, files, onStop); why != nil {
		writeDiagnostic(o.pages.window.Window.AddTask, out, "recordings will not play: %v\n", why)
	} else {
		playing.setPluginServices(o.pages.plugins)
		o.pages.window.Media.playing.Store(playing)
	}
	return nil
}

// Close shuts the door on every question, stops the passes behind the vault,
// waits for them, and closes what the window holds open.
func (o *Installation) Close() error { return o.lifecycle.Close.Execute(context.Background()) }

// Show puts a vault in the window, in place of the current one or as the first. `open` is the plugin
// open whose lock the switch shares, nil for a user switch.
func (o *Installation) Show(ctx context.Context, v domain.Vault, open *windowsession.PluginOpen) error {
	return o.lifecycle.Show.Execute(ctx, v, open)
}

// FlushPending has every page write what only it holds, then waits for the
// writes already taken. It returns false when a page holds work that needs
// user confirmation.
func (o *Installation) FlushPending(ctx context.Context) bool {
	return o.lifecycle.Flush.Execute(ctx)
}

// WaitForFlush waits for every page to write its pending data. The wait is on
// ctx ended or the vault was requested again.
func (o *Installation) WaitForFlush(ctx context.Context) bool {
	return o.pages.window.Window.WaitForFlush(ctx)
}

const (
	// readingTheSettings is the task ID for a settings name that this build reads under another name.
	// Each message is appended to this ID with its position.
	readingTheSettings = "read settings"
	// readingTheFloor is the task ID for a similarity floor that is stored and cannot be read.
	readingTheFloor = "read search floor"
)

// Report records configuration messages as tasks that are notices, not failures.
func (o *Installation) Report(messages []string) {
	for item, one := range messages {
		o.pages.window.Window.AddTask(work.Task{
			ID:    fmt.Sprintf("%s %d", readingTheSettings, item),
			Title: "Read settings",
			About: one,
			Kind:  work.KindNotice,
		})
	}
}

// Imports copies files dropped onto the window into a folder of the vault; the
// root is the empty path.
// It runs where the drop reached the application, off the page's thread, and
// returns no value: results go to all listeners.
// The watcher does not report images or archives, so Imports names what arrived itself.
func (o *Installation) Imports(ctx context.Context, into string, paths []string) {
	o.imports.Execute(ctx, o.GetCurrentVault(), into, paths)
}

// GetCurrentVault returns the vault displayed in the window.
func (o *Installation) GetCurrentVault() domain.Vault {
	return o.pages.window.session.GetCurrentVault()
}

// GetOpenTabs returns the tabs reported by the displayed page.
func (o *Installation) GetOpenTabs() window.OpenTabs { return o.pages.window.session.GetOpenTabs() }
