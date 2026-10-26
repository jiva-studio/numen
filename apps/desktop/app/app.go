// Package app provides the desktop application runner and host lifecycle.
package app

import (
	"context"
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

	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/apps/desktop/internal/plugincli"
	"github.com/jiva-studio/numen/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/apps/desktop/internal/themes"
	"github.com/jiva-studio/numen/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const appName = "numen"

// DefaultAgentAddr is the default MCP listening address.
const DefaultAgentAddr = "127.0.0.1:23579"

// Options holds command-line options for running the desktop application.
type Options struct {
	IndexPath      string
	RegistryPath   string
	Vault          string
	McpAddr        string
	NoMcp          bool
	InterfaceScale float64
	TextScale      float64
	RebuildIndex   bool
	SafeMode       bool
	ShowVersion    bool
	Debug          DebugOptions
}

// NewOptions creates default options for the application.
func NewOptions() *Options {
	return &Options{
		McpAddr: DefaultAgentAddr,
	}
}

// Run executes the application. If opts is not supplied or nil, command-line flags are parsed.
func Run(opts ...*Options) error {
	var opt *Options
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	} else {
		opt = parseFlags()
	}

	if opt.ShowVersion {
		fmt.Fprintln(os.Stdout, version.GetVersionLine(appName))
		return nil
	}

	cfg := makeConfig(os.Stderr)
	if opt.IndexPath != "" {
		cfg.IndexPath = opt.IndexPath
	}
	if opt.RegistryPath != "" {
		cfg.RegistryPath = opt.RegistryPath
	}
	cfg.ShouldRebuildIndex = opt.RebuildIndex
	cfg = cfg.SetSafeMode(opt.SafeMode)

	scaleFlags := scaleSettings{
		interfaceScale: opt.InterfaceScale,
		textScale:      opt.TextScale,
	}

	if err := run(cfg, opt.Vault, scaleFlags, &opt.Debug); err != nil {
		if !opt.Debug.isServingHTTP() {
			failWithDialog(cfg, err)
		}
		return err
	}
	return nil
}

func parseFlags() *Options {
	if len(os.Args) > 1 && os.Args[1] == "plugin" {
		os.Exit(plugincli.Run(os.Args[2:], os.Stdout, os.Stderr))
	}

	opts := NewOptions()
	fs := flag.CommandLine
	if !fs.Parsed() {
		fs.StringVar(&opts.IndexPath, "index", "", "path to the index database")
		fs.StringVar(&opts.RegistryPath, "registry", "", "path to the vault list")
		fs.StringVar(&opts.Vault, "vault", "",
			"the vault to open: a name, a path or an identity; the one opened last by default")
		fs.StringVar(&opts.McpAddr, "mcp-addr", DefaultAgentAddr,
			"where agents reach this vault; anything but a loopback address opens it to the network")
		fs.BoolVar(&opts.NoMcp, "no-mcp", false, "do not let agents reach this vault")
		fs.Float64Var(&opts.InterfaceScale, "interface-scale", 0,
			"how large the interface is drawn, 1 being as designed; this launch alone")
		fs.Float64Var(&opts.TextScale, "text-scale", 0,
			"how large the text a person reads is set, 1 being as designed; this launch alone")
		fs.BoolVar(&opts.RebuildIndex, "rebuild-index", false,
			"read every file and put it in the index again, whatever the index remembers")
		fs.BoolVar(&opts.SafeMode, "safe-mode", false, "disable plugins for emergency recovery")
		fs.BoolVar(&opts.ShowVersion, "version", false, "say what this build is and stop")
		RegisterDebugFlags(fs, opts)
		_ = fs.Parse(os.Args[1:])
	}
	return opts
}

func makeConfig(out io.Writer) container.Config {
	cfg := platform.GetConfig()
	cfg.ErrorHandler = func(err error) { fmt.Fprintln(out, "numen:", err) }
	return cfg
}

type scaleSettings struct{ interfaceScale, textScale float64 }

func (s scaleSettings) validate() error {
	if s.interfaceScale > 0 {
		if err := settings.InterfaceScaleBounds.Check("-interface-scale", s.interfaceScale); err != nil {
			return err
		}
	}
	if s.textScale > 0 {
		return settings.TextScaleBounds.Check("-text-scale", s.textScale)
	}
	return nil
}

func run(cfg container.Config, vault string, scaleFlags scaleSettings, debugOpts *DebugOptions) error {
	if err := debugOpts.apply(cfg); err != nil {
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

	setSchemasDir()

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	if err := cfg.MigratePluginSettings(); err != nil {
		return err
	}
	selected, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.SetSettings(selected)
	applyMemoryLimit(cfg.ResolvedCompute.MemoryLimitMB, debug.SetMemoryLimit, os.LookupEnv)
	cfg.InterfaceScale, cfg.TextScale = scaleFlags.interfaceScale, scaleFlags.textScale

	displayed, err := openInstallation(ctx, cfg, vault)
	if err != nil {
		return err
	}
	pages, installation := displayed.pages, displayed.opened
	installation.Services.Themes = themes.Load(cfg.Themes, installation.SayTheme, os.Stdout)

	installation.Report(selected.Notices)

	going := &vaultFlush{flush: installation.FlushPending}
	shutdownSeq := shutdown.NewSequence(
		shutdown.Report(appName, os.Stderr, "", installation.Close),
		shutdown.Report(appName, os.Stderr, "plugins", func() error {
			return cfg.StopPlugins(context.Background())
		}),
		stop,
	)

	defer func() {
		going.flushVault()
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
	window = makeWindow(app, installation)
	tracker.addWindow()
	connectWindow(ctx, app, cfg, window, installation, tracker, going)

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
	ctx context.Context, cfg container.Config, vault string,
) (shownInstallation, error) {
	pages, err := editor.Pages()
	if err != nil {
		return shownInstallation{}, err
	}
	versions, err := cfg.NewTextVersions()
	if err != nil {
		return shownInstallation{}, err
	}
	cfg.TextVersions = versions

	made, err := cfg.GetEditorAssembly(ctx)
	if err != nil {
		return shownInstallation{}, err
	}
	opened, err := editor.Open(ctx, made, vault, os.Stdout)
	if err != nil {
		return shownInstallation{}, err
	}
	cfg.SetPluginVaultPath(opened.GetCurrentVault().Path)
	getShownVault := func() (domain.Vault, bool) {
		current := opened.GetCurrentVault()
		return current, current.Path != ""
	}
	shared, err := made.OpenPluginVault(getShownVault, container.PluginVaultHooks{Written: func() {
		if wake := opened.State.NotifyNoteWritten; wake != nil {
			wake()
		}
	}})
	if err != nil {
		return shownInstallation{}, err
	}
	cfg.SetPluginVault(shared)
	cfg.SetPluginTexts(ctx, getShownVault)
	return shownInstallation{pages: pages, opened: opened}, nil
}

func makeApplication(
	opened *editor.Installation, pages http.Handler, shutdownSeq *shutdown.Sequence,
	going *vaultFlush, windowVis visibility,
) *application.App {
	return application.New(application.Options{
		Name: appName,
		Assets: application.AssetOptions{
			Handler: opened.Services.NewHandler(pages),
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
	return app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          getWindowTitle(opened.GetCurrentVault(), opened.State.GetOpenTabs()),
		Width:          1280,
		Height:         860,
		URL:            "/",
		EnableFileDrop: true,
	})
}

// makeSecondaryWindow creates a new window displaying a torn-off tab at the specified position.
func makeSecondaryWindow(
	app *application.App, opened *editor.Installation, tabID string, x, y int,
) *application.WebviewWindow {
	destURL := "/?window=secondary"
	if tabID != "" {
		destURL = "/?window=secondary&tab=" + url.QueryEscape(tabID)
	}
	opts := application.WebviewWindowOptions{
		Title:          getWindowTitle(opened.GetCurrentVault(), opened.State.GetOpenTabs()),
		Width:          1280,
		Height:         860,
		URL:            destURL,
		EnableFileDrop: true,
	}
	if x != 0 || y != 0 {
		opts.InitialPosition = application.WindowXY
		opts.X = x
		opts.Y = y
	}
	return app.Window.NewWithOptions(opts)
}

func connectWindow(
	ctx context.Context, app *application.App, cfg container.Config, window *application.WebviewWindow, opened *editor.Installation,
	tracker *windowTracker, going *vaultFlush,
) {
	opened.Services.SetFolderDialog(&folderDialog{window: window})

	naming := func(open domain.OpenTabs) { window.SetTitle(getWindowTitle(opened.GetCurrentVault(), open)) }
	opened.State.NotifyOpenTabs = naming

	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		importDroppedFiles(ctx, opened, event)
	})

	opened.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		cfg.SetPluginVaultPath(v.Path)
		err := opened.Show(ctx, v)
		naming(opened.State.GetOpenTabs())
		return err
	}

	opened.State.DetachTab = func(_ context.Context, tabID string, x, y int) error {
		tracker.addWindow()
		secWindow := makeSecondaryWindow(app, opened, tabID, x, y)
		secSeen := visibility{
			hide: func() { secWindow.Hide() },
			show: func() { secWindow.Show() },
		}
		secWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
			if !handleWindowClosing(ctx, tracker, going, secSeen, opened.WaitForFlush, secWindow.Close) {
				event.Cancel()
			}
		})
		secWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
			importDroppedFiles(ctx, opened, event)
		})
		return nil
	}
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
	opened.Imports(ctx, into, event.Context().DroppedFiles())
}

const droppedInto = "data-file-drop-target"

func getWindowTitle(v domain.Vault, open domain.OpenTabs) string {
	if front, target := open.GetFrontTab(); target && front.Path != "" {
		return "numen — " + path.Base(front.Path)
	}
	if v.Name == "" {
		return appName
	}
	return "numen — " + v.Name
}
