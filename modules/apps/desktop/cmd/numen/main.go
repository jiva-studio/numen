// Command numen is the window: one vault, one note, and the plex around it.
//
// It is a separate binary from the command line one because a webview links
// against the system's own browser, and a binary carrying one can no longer be
// built for another machine from this one. Both are entry points onto the same
// core.
//
// It also carries the tools an agent works the vault through, unless it is
// built with the `nomcp` tag.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime/debug"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func main() {
	cfg := makeConfig(os.Stderr)
	var agentFlags agentOptions
	var scaleFlags scaleSettings
	var vault string
	var showVersion bool
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.StringVar(&vault, "vault", "",
		"the vault to open: a name, a path or an identity; the one opened last by default")
	flag.StringVar(&agentFlags.addr, "mcp-addr", defaultAgentAddr,
		"where agents reach this vault; anything but a loopback address opens it to the network")
	flag.BoolVar(&agentFlags.isOff, "no-mcp", false, "do not let agents reach this vault")
	flag.Float64Var(&scaleFlags.interfaceScale, "interface-scale", 0,
		"how large the interface is drawn, 1 being as designed; this launch alone")
	flag.Float64Var(&scaleFlags.textScale, "text-scale", 0,
		"how large the text a person reads is set, 1 being as designed; this launch alone")
	flag.BoolVar(&cfg.ShouldRebuildIndex, "rebuild-index", false,
		"read every file and put it in the index again, whatever the index remembers")
	flag.BoolVar(&showVersion, "version", false, "say what this build is and stop")
	flag.Parse()

	if showVersion {
		fmt.Println(version.GetVersionLine("numen"))
		return
	}

	if err := run(cfg, agentFlags, vault, scaleFlags); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		refuse(cfg, err)
		os.Exit(1)
	}
}

// makeConfig is what this binary starts from: what the machine supplies the
// core, and where the core says what it went wrong at and carried on past. That
// is the same place everything else this binary could not do is said.
func makeConfig(out io.Writer) container.Config {
	cfg := platform.GetConfig()
	cfg.ErrorHandler = func(err error) { fmt.Fprintln(out, "numen:", err) }
	return cfg
}

// scaleSettings are what the command line said about size: how large the interface is
// drawn, and how large the text a person reads is set. Zero is not said, and
// the settings file stands.
type scaleSettings struct{ interfaceScale, textScale float64 }

// validate is what is wrong with a number the setting it says does not take.
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

func run(cfg container.Config, mcp agentOptions, vault string, scaleFlags scaleSettings) error {
	if err := scaleFlags.validate(); err != nil {
		return err
	}

	// Before anything draws: the settings a folder dialog reads are looked for
	// once, the first time something asks for one.
	setSchemasDir()

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	chosen, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.SetSettings(chosen)
	applyMemoryLimit(cfg.ResolvedCompute.MemoryLimitMB, debug.SetMemoryLimit, os.LookupEnv)
	cfg.InterfaceScale, cfg.TextScale = scaleFlags.interfaceScale, scaleFlags.textScale

	shown, err := openInstallation(ctx, cfg, vault)
	if err != nil {
		return err
	}
	pages, opened := shown.pages, shown.opened
	loadThemes(cfg, opened)

	// What reading the settings had to tell a person goes where they are: a
	// window opened from a desktop entry has no terminal to write to.
	opened.Report(chosen.Notices)

	reachable := makeEndpoint(ctx, cfg, opened, mcp)
	going := &vaultFlush{flush: opened.FlushPending}
	behind := makeShutdown(reachable, opened, stop)

	// The vault settles before a quit is let through, so the steps behind it are
	// asked for here with nothing left owed.
	defer func() {
		going.flushVault()
		behind.Run()
	}()

	// The window is hidden before the vault is flushed and shown again when a
	// page calls the close off.
	var window *application.WebviewWindow
	seen := visibility{
		hide: func() { window.Hide() },
		show: func() { window.Show() },
	}

	app := makeApplication(opened, pages, behind, going, seen)
	window = makeWindow(app, opened)
	connectWindow(ctx, window, opened, reachable)

	// A hook runs before the window is destroyed and on a thread of its own, so
	// the page is still drawn and still answered while what it owes is written.
	// A cancelled event is where the hooks stop, and the destroy the window
	// registered for itself is one of the listeners after them.
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !closeWindow(ctx, going, seen, opened.WaitForFlush, window.Close) {
			event.Cancel()
		}
	})

	return app.Run()
}

// shownInstallation is the pages of the window and the installation they show.
type shownInstallation struct {
	pages  http.Handler
	opened *editor.Installation
}

// openInstallation reads the pages and opens the vault the window shows.
func openInstallation(
	ctx context.Context, cfg container.Config, vault string,
) (shownInstallation, error) {
	// Before the window: every page this process reads is read through the
	// runtime made here, and one made after the window reads a page as nothing.
	// A machine holding no runtime yet is told so by the first recognition.
	if err := cfg.PrepareRecogniser(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "numen: nothing to read a scan with:", err)
	}
	pages, err := editor.Pages()
	if err != nil {
		return shownInstallation{}, err
	}

	made, err := cfg.GetEditorAssembly(ctx)
	if err != nil {
		return shownInstallation{}, err
	}
	opened, err := editor.Open(ctx, made, vault, os.Stdout)
	if err != nil {
		return shownInstallation{}, err
	}
	return shownInstallation{pages: pages, opened: opened}, nil
}

// loadThemes gives the window the installation's themes. A folder that could not
// be made leaves the ones this binary ships, and a theme the settings name that
// the catalogue has not is said where the person is.
func loadThemes(cfg container.Config, opened *editor.Installation) {
	themes, wrong := cfg.Themes(opened.SayTheme)
	if wrong != nil {
		fmt.Fprintf(os.Stdout, "themes: %v\n", wrong)
	}
	opened.Services.Themes = themes
}

// makeEndpoint is the agents' endpoint on the vault in the window.
func makeEndpoint(
	ctx context.Context, cfg container.Config, opened *editor.Installation, mcp agentOptions,
) *agents.Endpoint {
	return &agents.Endpoint{
		Serve: func() (func() error, error) {
			return serveAgents(ctx, cfg, opened, mcp, os.Stdout)
		},
		GetVault:       opened.GetCurrentVault,
		SetAgent:       opened.State.SetAgent,
		SetUnreachable: func(said string) { opened.State.Unreachable.Store(said) },
		ErrorHandler:   func(err error) { fmt.Fprintln(os.Stderr, "numen:", err) },
	}
}

// makeShutdown stops the agents, closes the installation, then ends the context
// they ran under.
func makeShutdown(
	reachable *agents.Endpoint, opened *editor.Installation, stop func(),
) *shutdown.Sequence {
	return shutdown.NewSequence(
		reachable.Stop,
		func() {
			if err := opened.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "numen:", err)
			}
		},
		stop,
	)
}

// makeApplication is the application serving the page. It ends when its last
// window closes.
func makeApplication(
	opened *editor.Installation, pages http.Handler, behind *shutdown.Sequence,
	going *vaultFlush, seen visibility,
) *application.App {
	return application.New(application.Options{
		Name: "numen",
		Assets: application.AssetOptions{
			Handler: opened.Services.NewHandler(pages),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// Runs on the thread the window is drawn on, after the application has
		// stopped dispatching.
		PostShutdown: behind.Run,
		// A quit that does not come through the window flushes the vault off the
		// serving thread, then asks for the quit again. A flush that ended with a
		// question open does not ask.
		ShouldQuit: func() bool {
			return requestQuit(going, seen, func() { application.Get().Quit() })
		},
	})
}

// makeWindow is the window the vault is shown in.
func makeWindow(app *application.App, opened *editor.Installation) *application.WebviewWindow {
	return app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  getWindowTitle(opened.GetCurrentVault(), opened.State.GetOpenTabs()),
		Width:  1280,
		Height: 860,
		URL:    "/",
		// Files dragged from the desktop reach the page, which marks the drop targets.
		EnableFileDrop: true,
	})
}

// connectWindow wires what the window owns into the installation: the folder
// dialog, the title, dropped files and opening another vault.
func connectWindow(
	ctx context.Context, window *application.WebviewWindow, opened *editor.Installation,
	reachable *agents.Endpoint,
) {
	// Choosing a folder is the machine's own, and it opens over this window.
	opened.Services.SetFolderDialog(&folderDialog{window: window})

	// The window is named after what the person is looking at, and is named
	// again each time the page says what it has open.
	naming := func(open domain.OpenTabs) { window.SetTitle(getWindowTitle(opened.GetCurrentVault(), open)) }
	opened.State.NotifyOpenTabs = naming

	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		importDroppedFiles(ctx, opened, event)
	})

	// Opening another vault, as a person asks for it. The agents are told which
	// vault they are working when their session opens, so the endpoint they
	// reach it through is stopped and started again around the swap.
	opened.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		err := reachable.RunWhileStopped(func() error { return opened.Show(ctx, v) })
		naming(opened.State.GetOpenTabs())
		return err
	}

	// A tool is served where what it works through is there, so the dialog and
	// the swap above stand before the agents are let in.
	//
	// An agent nobody can reach is a panel that says so, not a window that does
	// not open. Everything else the window does is the vault, and the vault is
	// here.
	reachable.Start()
}

// importDroppedFiles copies files dropped on the window into the folder the
// drop target names, and ignores a drop on no target.
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

// droppedInto is the attribute a page marks a drop target with. Its value is
// the vault folder a dropped file is filed in, the root being the empty path.
const droppedInto = "data-file-drop-target"

// getWindowTitle is what the window is called: the application, and the file
// the person is looking at. A window with no file in front of it is called
// after the vault it is showing.
func getWindowTitle(v domain.Vault, open domain.OpenTabs) string {
	if front, held := open.GetFrontTab(); held && front.Path != "" {
		return "numen — " + path.Base(front.Path)
	}
	if v.Name == "" {
		return "numen"
	}
	return "numen — " + v.Name
}

// agentOptions is what the person said about letting agents in.
type agentOptions struct {
	addr  string
	isOff bool
}
