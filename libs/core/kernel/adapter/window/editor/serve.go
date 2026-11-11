package editor

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
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

// PluginOpener switches the window to a vault on behalf of a plugin.
type PluginOpener interface {
	Execute(ctx context.Context, v domain.Vault) error
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

// Lifecycle is what runs the window from its start to its close. NoteWritten tells the embedder that a note
// holds new prose; a nil one leaves the state as it is.
type Lifecycle struct {
	Start      Starter
	Show       Shower
	OpenPlugin PluginOpener
	Flush      Flusher
	Close      Closer

	NoteWritten func()
}

// OpenedWindow is an installation with the state it was put together over. The state is what the services
// mounted on the installation work through; the installation drives the window through its pages only.
type OpenedWindow struct {
	*Installation
	State *State
}

// Installation is a window put together and running: the pieces the services
// of its routes and anything else working the same vault need.
// What runs behind the vault is the lifecycle; the installation maps the
// calls of the application onto it.
type Installation struct {
	lifecycle Lifecycle
	imports   Importer
	pages     pageSet
	dialogs   []func(dialog PathDialog)
	out       io.Writer
}

// errNoVault is returned for a vault query on a window with no vault open.
var errNoVault = errors.New("the window has no vault")

// NewOpenedWindow makes the installation of a window over its state, the lifecycle of what it shows and
// the import of dropped files. Nothing is served and no vault is open until Start.
func NewOpenedWindow(state *State, lifecycle Lifecycle, imports Importer, out io.Writer) *OpenedWindow {
	if lifecycle.NoteWritten != nil {
		state.Writes.NotifyNoteWritten = lifecycle.NoteWritten
	}
	return &OpenedWindow{
		Installation: &Installation{
			lifecycle: lifecycle,
			imports:   imports,
			pages:     pageSet{gate: state.Gate, window: statePages{state: state}},
			out:       out,
		},
		State: state,
	}
}

// Mount adds the services the window answers through.
func (o *Installation) Mount(routes ...Route) { o.pages.routes = append(o.pages.routes, routes...) }

// SetThemes sets the themes the window offers and puts on its pages. A window
// put together without a catalog serves none.
func (o *Installation) SetThemes(themes numenv1connect.ThemeServiceHandler) { o.pages.themes = themes }

// GetThemes returns the themes the window offers.
func (o *Installation) GetThemes() numenv1connect.ThemeServiceHandler { return o.pages.themes }

// SetPlugins sets where the calls for the services plugins serve are forwarded,
// and where the files of a plugin are read from.
func (o *Installation) SetPlugins(services PluginServices, dir func(id string) (string, bool)) {
	o.pages.plugins, o.pages.pluginDir = services, dir
}

// OnFolderDialog registers what a service does with the native dialog when the
// application gives one.
func (o *Installation) OnFolderDialog(bind func(dialog PathDialog)) {
	o.dialogs = append(o.dialogs, bind)
}

// SetFolderDialog sets the native file and folder dialog used when a client
// requests one.
func (o *Installation) SetFolderDialog(dialog PathDialog) {
	for _, bind := range o.dialogs {
		bind(dialog)
	}
}

// Start opens the first vault, follows the model of the embedder and serves
// recordings over the loopback socket.
func (o *Installation) Start(ctx context.Context, first domain.Vault, files *FileServer) error {
	o.lifecycle.Start.Execute(ctx, first)

	// Recordings are served over a dedicated socket, opened after every
	// dependency it answers through is in place. When the socket cannot be opened,
	// the player has no address.
	onStop := func(why error) {
		writeDiagnostic(o.pages.window.AddTask, o.out, "recordings will no longer play: %v\n", why)
	}
	if playing, why := Listen(ctx, files, onStop); why != nil {
		writeDiagnostic(o.pages.window.AddTask, o.out, "recordings will not play: %v\n", why)
	} else {
		o.pages.window.SetPlaying(playing)
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

// OpenPluginVault switches the window to the vault on behalf of a plugin.
func (o *Installation) OpenPluginVault(ctx context.Context, v domain.Vault) error {
	return o.lifecycle.OpenPlugin.Execute(ctx, v)
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
	return o.pages.window.WaitForFlush(ctx)
}

const (
	// applyingTheme is the task ID for a theme the settings name that the catalog does not have.
	applyingTheme = "apply theme"
	// readingTheSettings is the task ID for a settings name that this build reads under another name.
	// Each message is appended to this ID with its position.
	readingTheSettings = "read settings"
	// readingTheFloor is the task ID for a similarity floor that is stored and cannot be read.
	readingTheFloor = "read search floor"
)

// RecordThemeWarning records a theme warning task, and takes the task out of the list for an empty message,
// which says the theme the settings name is found now.
func (o *Installation) RecordThemeWarning(message string) {
	if message == "" {
		o.pages.window.RemoveTask(applyingTheme)
		return
	}
	o.pages.window.AddTask(work.Task{ID: applyingTheme, Title: "Apply theme", Error: message})
}

// Report records configuration messages as tasks that are notices, not failures.
func (o *Installation) Report(messages []string) {
	for item, one := range messages {
		o.pages.window.AddTask(work.Task{
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
	o.imports.Execute(ctx, o.pages.window.GetCurrentVault(), into, paths)
}
