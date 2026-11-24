// Package app provides the desktop application runner and host lifecycle.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime/debug"

	"github.com/jiva-studio/numen/apps/desktop/internal/folderdialog"
	"github.com/jiva-studio/numen/apps/desktop/internal/frame"
	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const appName = "numen"

// Options holds command-line options for running the desktop application.
type Options struct {
	IndexPath      string
	RegistryPath   string
	Vault          string
	InterfaceScale float64
	TextScale      float64
	RebuildIndex   bool
	SafeMode       bool
	ShowVersion    bool
	Debug          DebugOptions
}

// NewOptions creates default options for the application.
func NewOptions() *Options {
	return &Options{}
}

// Run executes the application with the options the command line gave.
func Run(opt *Options) error {
	if opt.ShowVersion {
		_, err := fmt.Fprintln(os.Stdout, version.GetVersionLine(appName))
		return err
	}

	cfg := platform.GetConfig()
	if opt.IndexPath != "" {
		cfg.IndexPath = opt.IndexPath
	}
	if opt.RegistryPath != "" {
		cfg.RegistryPath = opt.RegistryPath
	}
	cfg.ShouldRebuildIndex = opt.RebuildIndex
	cfg.SafeMode = opt.SafeMode
	applyWebviewEnvironment(cfg)
	services := makeServices(os.Stderr)
	services.Execution, services.Preferences, services.Endpoints = container.PreparePluginHost(
		services.Execution, services.Preferences, services.Endpoints, cfg.SafeMode,
	)

	scaleFlags := scaleSettings{
		interfaceScale: opt.InterfaceScale,
		textScale:      opt.TextScale,
	}

	err := run(cfg, services, opt.Vault, scaleFlags, &opt.Debug)
	if err == nil {
		return nil
	}
	if opt.Debug.isServingHTTP() {
		return err
	}
	if dialogErr := failWithDialog(cfg, err); dialogErr != nil {
		return errors.Join(err, dialogErr)
	}
	return err
}

func makeServices(out io.Writer) platform.Installation {
	services := platform.NewPluginServices(platform.TargetEditor)
	services.Resources.ErrorHandler = func(err error) {
		// gomutants:disable-next-line BRANCH_IF,CONDITIONALS_NEGATION reason="the branch returns where the function ends, so skipping it or inverting its condition ends the function at the same point"
		if _, writeErr := fmt.Fprintln(out, "numen:", err); writeErr != nil {
			return
		}
	}
	return services
}

type scaleSettings struct{ interfaceScale, textScale float64 }

func (s scaleSettings) validate() error {
	if s.interfaceScale > 0 {
		if err := container.CheckInterfaceScale("-interface-scale", s.interfaceScale); err != nil {
			return err
		}
	}
	if s.textScale > 0 {
		return container.CheckTextScale("-text-scale", s.textScale)
	}
	return nil
}

func run(cfg container.Config, services platform.Installation, vault string, scaleFlags scaleSettings, debugOpts *DebugOptions) error {
	if err := debugOpts.apply(services.Preferences.RefreshDependencies, services.Resources); err != nil {
		return err
	}
	if err := debugOpts.ensureVault(cfg, services.Resources, vault); err != nil {
		return err
	}
	pluginsStop := shutdown.NewSequence(shutdown.Report(appName, os.Stderr, "plugins", func() error {
		return container.StopPlugins(services.Execution.Watch, context.Background())
	}))
	defer pluginsStop.Run()
	if err := scaleFlags.validate(); err != nil {
		return err
	}

	if err := folderdialog.SetSchemasDir(); err != nil {
		return err
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	cfg, notices, err := cfg.ApplySettings(services.Execution, services.Preferences, services.Resources)
	if err != nil {
		return err
	}
	applyMemoryLimit(cfg.GetMachine().ResolvedCompute.MemoryLimitMB, debug.SetMemoryLimit, os.LookupEnv)

	frame := &windowFrame{}
	cfg.Scales = container.Scales{InterfaceScale: scaleFlags.interfaceScale, TextScale: scaleFlags.textScale}
	displayed, err := openInstallation(ctx, cfg, services, vault, debugOpts.getFrame(frame))
	if err != nil {
		return err
	}
	pages, installation := displayed.pages, displayed.opened
	installation.Report(notices)

	going := &vaultFlush{flush: installation.FlushPending}
	shutdownSeq := shutdown.NewSequence(
		shutdown.Report(appName, os.Stderr, "", installation.Close),
		pluginsStop.Run,
		stop,
	)

	defer func() {
		going.flushVault(ctx)
		shutdownSeq.Run()
	}()

	if debugOpts.isServingHTTP() {
		return debugOpts.serveHTTP(ctx, services.Resources, installation, pages)
	}

	tracker := newWindowTracker()
	var window *application.WebviewWindow
	observed := visibility{
		hide: func() { window.Hide() },
		show: func() { window.Show() },
	}

	app := makeApplication(installation, pages, shutdownSeq, going, observed)
	enableMediaCapture(app)
	app.Menu.SetApplicationMenu(makeApplicationMenu())
	window = makeWindow(app, installation)
	tracker.addWindow()
	connectWindow(ctx, app, window, installation, frame, windowHooks{tracker: tracker, going: going})

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !handleWindowClosing(ctx, tracker, going, observed, installation.WaitForFlush, window.Close) {
			event.Cancel()
		}
	})

	return app.Run()
}

type shownInstallation struct {
	pages  http.Handler
	opened *editor.Installation
}

func openInstallation(
	ctx context.Context, cfg container.Config, services platform.Installation, vault string, frame editor.Frame,
) (shownInstallation, error) {
	pages, err := editor.CreatePagesHandler()
	if err != nil {
		return shownInstallation{}, err
	}
	opened, err := cfg.OpenEditor(services.Execution, services.Preferences, services.Endpoints, services.Resources, ctx, container.WindowOpening{Named: vault, Out: os.Stdout, Frame: frame})
	if err != nil {
		return shownInstallation{}, err
	}
	return shownInstallation{pages: pages, opened: opened}, nil
}

func makeApplication(
	opened *editor.Installation, pages http.Handler, shutdownSeq *shutdown.Sequence,
	going *vaultFlush, windowVis visibility,
) *application.App {
	return application.New(application.Options{
		Name: appName,
		Assets: application.AssetOptions{
			Handler: opened.NewHandler(pages),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		PostShutdown: shutdownSeq.Run,
		ShouldQuit: func() bool {
			return requestQuit(going, windowVis, func() { application.Get().Quit() })
		},
	})
}

func makeWindow(app *application.App, opened *editor.Installation) *application.WebviewWindow {
	return app.Window.NewWithOptions(getMainWindowOptions(opened))
}

func getMainWindowOptions(opened *editor.Installation) application.WebviewWindowOptions {
	return frame.ApplyPlatformFrame(application.WebviewWindowOptions{
		Title:          getWindowTitle(opened.GetCurrentVault(), opened.GetOpenTabs()),
		Width:          1280,
		Height:         860,
		URL:            "/",
		EnableFileDrop: true,
	})
}

func getSecondaryWindowURL(tab windowdomain.DetachedTab) string {
	query := url.Values{"window": {"secondary"}}
	if tab.ID != "" {
		query.Set("id", tab.ID)
	}
	return "/?" + query.Encode()
}

func makeSecondaryWindow(
	app *application.App, opened *editor.Installation, tab windowdomain.DetachedTab,
) *application.WebviewWindow {
	return app.Window.NewWithOptions(getSecondaryWindowOptions(opened, tab))
}

// The size of a window of a torn-off tab that gives no size of its own.
const (
	secondaryWindowWidth  = 1280
	secondaryWindowHeight = 860
)

func getSecondaryWindowOptions(opened *editor.Installation, tab windowdomain.DetachedTab) application.WebviewWindowOptions {
	opts := frame.ApplyPlatformFrame(application.WebviewWindowOptions{
		Title:          getWindowTitle(opened.GetCurrentVault(), opened.GetOpenTabs()),
		Width:          secondaryWindowWidth,
		Height:         secondaryWindowHeight,
		URL:            getSecondaryWindowURL(tab),
		EnableFileDrop: true,
	})
	if tab.Width > 0 && tab.Height > 0 {
		opts.Width = tab.Width
		opts.Height = tab.Height
	}
	if tab.X != 0 || tab.Y != 0 {
		opts.InitialPosition = application.WindowXY
		opts.X = tab.X
		opts.Y = tab.Y
	}
	return opts
}

// windowHooks holds what a window needs to close and to follow the vault.
type windowHooks struct {
	tracker *windowTracker
	going   *vaultFlush
}

// windowFrame is the application window around the editor: it titles the main window after the open tabs and
// opens a window of its own for a torn-off tab. It is made before the editor opens and is given the windows
// by connectWindow, so it does nothing until then.
type windowFrame struct {
	window *application.WebviewWindow
	opened *editor.Installation
	dialog *folderdialog.Dialog
	detach func(tab windowdomain.DetachedTab) error
}

func (f *windowFrame) ShowOpenTabs(open windowdomain.OpenTabs) {
	if f.window != nil {
		f.window.SetTitle(getWindowTitle(f.opened.GetCurrentVault(), open))
	}
}

func (f *windowFrame) ChooseFolder(ctx context.Context, title, startingAt string) (string, bool, error) {
	if f.dialog == nil {
		return "", false, machine.ErrNoFolderDialog
	}
	return f.dialog.ChooseFolder(ctx, title, startingAt)
}

func (f *windowFrame) ChooseFile(ctx context.Context, title, startingAt string, accept []string) (string, bool, error) {
	if f.dialog == nil {
		return "", false, machine.ErrNoFolderDialog
	}
	return f.dialog.ChooseFile(ctx, title, startingAt, accept)
}

func (f *windowFrame) ChooseSaveFile(ctx context.Context, title, name string, accept []string) (string, bool, error) {
	if f.dialog == nil {
		return "", false, machine.ErrNoFolderDialog
	}
	return f.dialog.ChooseSaveFile(ctx, title, name, accept)
}

func (f *windowFrame) DetachTab(_ context.Context, tab windowdomain.DetachedTab) error {
	if f.detach == nil {
		return errNoWindow
	}
	return f.detach(tab)
}

func detachTab(
	ctx context.Context, app *application.App, opened *editor.Installation, hooks windowHooks, tab windowdomain.DetachedTab,
) error {
	hooks.tracker.addWindow()
	secWindow := makeSecondaryWindow(app, opened, tab)
	secSeen := visibility{
		hide: func() { secWindow.Hide() },
		show: func() { secWindow.Show() },
	}
	secWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !handleWindowClosing(ctx, hooks.tracker, hooks.going, secSeen, opened.WaitForFlush, secWindow.Close) {
			event.Cancel()
		}
	})
	secWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		importDroppedFiles(ctx, opened, event)
	})
	return nil
}

var errNoWindow = errors.New("the application has no window to open")

func connectWindow(
	ctx context.Context, app *application.App, window *application.WebviewWindow, opened *editor.Installation,
	frame *windowFrame, hooks windowHooks,
) {
	*frame = windowFrame{
		window: window, opened: opened, dialog: folderdialog.New(window),
		detach: func(tab windowdomain.DetachedTab) error { return detachTab(ctx, app, opened, hooks, tab) },
	}

	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		importDroppedFiles(ctx, opened, event)
	})
}

func importDroppedFiles(ctx context.Context, opened *editor.Installation, event *application.WindowEvent) {
	landed := event.Context().DropTargetDetails()
	if landed == nil {
		return
	}
	into, marked := landed.Attributes[droppedInto]
	if !marked {
		return
	}
	opened.ImportFiles(ctx, into, event.Context().DroppedFiles())
}

const droppedInto = "data-file-drop-target"

func getWindowTitle(v domain.Vault, open windowdomain.OpenTabs) string {
	if front, target := open.GetActiveTab(); target && front.Path != "" {
		return "numen — " + path.Base(front.Path)
	}
	if v.Name == "" {
		return appName
	}
	return "numen — " + v.Name
}
