// Package app provides the desktop application runner and host lifecycle.
package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime/debug"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/jiva-studio/numen/apps/desktop/internal/folderdialog"
	"github.com/jiva-studio/numen/apps/desktop/internal/frame"
	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/apps/desktop/internal/themes"
	"github.com/jiva-studio/numen/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
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

// Run executes the application. If opts is not supplied or nil, command-line flags are parsed.
func Run(opts ...*Options) error {
	var opt *Options
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	} else {
		parsed, err := parseFlags()
		if err != nil {
			return err
		}
		opt = parsed
	}

	if opt.ShowVersion {
		_, err := fmt.Fprintln(os.Stdout, version.GetVersionLine(appName))
		return err
	}

	cfg := makeConfig(os.Stderr)
	if opt.IndexPath != "" {
		cfg.IndexPath = opt.IndexPath
	}
	if opt.RegistryPath != "" {
		cfg.RegistryPath = opt.RegistryPath
	}
	cfg.ShouldRebuildIndex = opt.RebuildIndex
	cfg.SafeMode = opt.SafeMode
	cfg = container.NewConfig(cfg)

	scaleFlags := scaleSettings{
		interfaceScale: opt.InterfaceScale,
		textScale:      opt.TextScale,
	}

	err := run(cfg, opt.Vault, scaleFlags, &opt.Debug)
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

func parseFlags() (*Options, error) {
	opts := NewOptions()
	fs := flag.CommandLine
	if !fs.Parsed() {
		fs.StringVar(&opts.IndexPath, "index", "", "path to the index database")
		fs.StringVar(&opts.RegistryPath, "registry", "", "path to the vault list")
		fs.StringVar(&opts.Vault, "vault", "",
			"the vault to open: a name, a path or an identity; the one opened last by default")
		fs.Float64Var(&opts.InterfaceScale, "interface-scale", 0,
			"scale of the interface (controls and spacing); 1 is the default; this launch only")
		fs.Float64Var(&opts.TextScale, "text-scale", 0,
			"scale of note and book text; 1 is the default; this launch only")
		fs.BoolVar(&opts.RebuildIndex, "rebuild-index", false,
			"read every file again and rewrite its index entries, whatever the index already holds")
		fs.BoolVar(&opts.SafeMode, "safe-mode", false, "disable plugins for emergency recovery")
		fs.BoolVar(&opts.ShowVersion, "version", false, "print the version and exit")
		RegisterDebugFlags(fs, opts)
		if err := fs.Parse(os.Args[1:]); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

func makeConfig(out io.Writer) container.Config {
	cfg := platform.GetConfig()
	cfg.ErrorHandler = func(err error) {
		// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
		if _, writeErr := fmt.Fprintln(out, "numen:", err); writeErr != nil {
			return
		}
	}
	return cfg
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

func run(cfg container.Config, vault string, scaleFlags scaleSettings, debugOpts *DebugOptions) error {
	if err := debugOpts.apply(&cfg); err != nil {
		return err
	}
	if err := debugOpts.ensureVault(cfg, vault); err != nil {
		return err
	}
	defer shutdown.Report(appName, os.Stderr, "plugins", func() error {
		return cfg.StopPlugins(context.Background())
	})()
	if err := scaleFlags.validate(); err != nil {
		return err
	}

	if err := folderdialog.SetSchemasDir(); err != nil {
		return err
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	cfg, notices, err := cfg.ApplySettings()
	if err != nil {
		return err
	}
	applyMemoryLimit(cfg.GetMachine().ResolvedCompute.MemoryLimitMB, debug.SetMemoryLimit, os.LookupEnv)

	frame := &windowFrame{}
	displayed, cfg, err := openInstallation(ctx, cfg, vault, frame)
	if err != nil {
		return err
	}
	pages, installation := displayed.pages, displayed.opened
	scales := container.Scales{InterfaceScale: scaleFlags.interfaceScale, TextScale: scaleFlags.textScale}
	themes.Report(container.SetThemes(cfg, installation.Installation, scales, installation.RecordThemeWarning), os.Stdout)

	installation.Report(notices)

	going := &vaultFlush{flush: installation.FlushPending}
	shutdownSeq := shutdown.NewSequence(
		shutdown.Report(appName, os.Stderr, "", installation.Close),
		shutdown.Report(appName, os.Stderr, "plugins", func() error {
			return cfg.StopPlugins(context.Background())
		}),
		stop,
	)

	defer func() {
		going.flushVault(ctx)
		shutdownSeq.Run()
	}()

	if debugOpts.isServingHTTP() {
		return debugOpts.serveHTTP(ctx, cfg, installation, pages)
	}

	tracker := newWindowTracker()
	var window *application.WebviewWindow
	observed := visibility{
		hide: func() { window.Hide() },
		show: func() { window.Show() },
	}

	app := makeApplication(installation, pages, shutdownSeq, going, observed)
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
	opened *container.OpenedWindow
}

func openInstallation(
	ctx context.Context, cfg container.Config, vault string, frame editor.Frame,
) (shownInstallation, container.Config, error) {
	pages, err := editor.Pages()
	if err != nil {
		return shownInstallation{}, cfg, err
	}
	opened, cfg, err := container.OpenEditor(cfg, ctx, vault, os.Stdout, frame)
	if err != nil {
		return shownInstallation{}, cfg, err
	}
	return shownInstallation{pages: pages, opened: opened}, cfg, nil
}

func makeApplication(
	opened *container.OpenedWindow, pages http.Handler, shutdownSeq *shutdown.Sequence,
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

func makeWindow(app *application.App, opened *container.OpenedWindow) *application.WebviewWindow {
	return app.Window.NewWithOptions(getMainWindowOptions(opened))
}

func getMainWindowOptions(opened *container.OpenedWindow) application.WebviewWindowOptions {
	return frame.Window(application.WebviewWindowOptions{
		Title:          getWindowTitle(opened.State.GetCurrentVault(), opened.State.Tabs.GetOpenTabs()),
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
	app *application.App, opened *container.OpenedWindow, tab windowdomain.DetachedTab,
) *application.WebviewWindow {
	return app.Window.NewWithOptions(getSecondaryWindowOptions(opened, tab))
}

// The size of a window of a torn-off tab that gives no size of its own.
const (
	secondaryWindowWidth  = 1280
	secondaryWindowHeight = 860
)

func getSecondaryWindowOptions(opened *container.OpenedWindow, tab windowdomain.DetachedTab) application.WebviewWindowOptions {
	opts := frame.Window(application.WebviewWindowOptions{
		Title:          getWindowTitle(opened.State.GetCurrentVault(), opened.State.Tabs.GetOpenTabs()),
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
	opened *container.OpenedWindow
	detach func(tab windowdomain.DetachedTab) error
}

func (f *windowFrame) ShowOpenTabs(open windowdomain.OpenTabs) {
	if f.window != nil {
		f.window.SetTitle(getWindowTitle(f.opened.State.GetCurrentVault(), open))
	}
}

func (f *windowFrame) DetachTab(_ context.Context, tab windowdomain.DetachedTab) error {
	if f.detach == nil {
		return errNoWindow
	}
	return f.detach(tab)
}

func detachTab(
	ctx context.Context, app *application.App, opened *container.OpenedWindow, hooks windowHooks, tab windowdomain.DetachedTab,
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
	ctx context.Context, app *application.App, window *application.WebviewWindow, opened *container.OpenedWindow,
	frame *windowFrame, hooks windowHooks,
) {
	opened.SetFolderDialog(folderdialog.New(window))
	*frame = windowFrame{
		window: window, opened: opened,
		detach: func(tab windowdomain.DetachedTab) error { return detachTab(ctx, app, opened, hooks, tab) },
	}

	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		importDroppedFiles(ctx, opened, event)
	})
}

func importDroppedFiles(ctx context.Context, opened *container.OpenedWindow, event *application.WindowEvent) {
	landed := event.Context().DropTargetDetails()
	if landed == nil {
		return
	}
	into, marked := landed.Attributes[droppedInto]
	if !marked {
		return
	}
	opened.Imports(ctx, into, event.Context().DroppedFiles())
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
