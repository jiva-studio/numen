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
	"os"
	"path"

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
	var letting agentOptions
	var said sizes
	var vault string
	var telling bool
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.StringVar(&vault, "vault", "",
		"the vault to open: a name, a path or an identity; the one opened last by default")
	flag.StringVar(&letting.addr, "mcp-addr", defaultAgentAddr,
		"where agents reach this vault; anything but a loopback address opens it to the network")
	flag.BoolVar(&letting.off, "no-mcp", false, "do not let agents reach this vault")
	flag.Float64Var(&said.interfaceScale, "interface-scale", 0,
		"how large the interface is drawn, 1 being as designed; this launch alone")
	flag.Float64Var(&said.textScale, "text-scale", 0,
		"how large the text a person reads is set, 1 being as designed; this launch alone")
	flag.BoolVar(&cfg.RebuildIndex, "rebuild-index", false,
		"read every file and put it in the index again, whatever the index remembers")
	flag.BoolVar(&telling, "version", false, "say what this build is and stop")
	flag.Parse()

	if telling {
		fmt.Println(version.GetVersionLine("numen"))
		return
	}

	if err := run(cfg, letting, vault, said); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		refuse(cfg, err)
		os.Exit(1)
	}
}

// makeConfig is what this binary starts from: what the machine supplies the
// core, and where the core says what it went wrong at and carried on past. That
// is the same place everything else this binary could not do is said.
func makeConfig(out io.Writer) container.Config {
	cfg := platform.Config()
	cfg.ErrorHandler = func(err error) { fmt.Fprintln(out, "numen:", err) }
	return cfg
}

// sizes are what the command line said about size: how large the interface is
// drawn, and how large the text a person reads is set. Zero is not said, and
// the settings file stands.
type sizes struct{ interfaceScale, textScale float64 }

// check is what is wrong with a number the setting it says does not take.
func (s sizes) check() error {
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

func run(cfg container.Config, mcp agentOptions, vault string, sizes sizes) error {
	if err := sizes.check(); err != nil {
		return err
	}

	// Before anything draws: the settings a folder dialog reads are looked for
	// once, the first time something asks for one.
	findSchemas()

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	chosen, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.SetIndexing(chosen.Indexing)
	cfg.Agent = chosen.Agent
	cfg.Importing = chosen.Importing
	cfg.InterfaceScale, cfg.TextScale = sizes.interfaceScale, sizes.textScale

	// Before the window: every page this process reads is read through the
	// runtime made here, and one made after the window reads a page as nothing.
	// A machine holding no runtime yet is told so by the first reading.
	if err := cfg.PrepareRecogniser(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "numen: nothing to read a scan with:", err)
	}
	pages, err := editor.Pages()
	if err != nil {
		return err
	}

	opened, err := editor.Open(ctx, cfg, vault, os.Stdout)
	if err != nil {
		return err
	}

	// What reading the settings had to tell a person goes where they are: a
	// window opened from a desktop entry has no terminal to write to.
	opened.Says(chosen.Said)

	// The agents' endpoint on the vault in the window, let in once the window is
	// built.
	reachable := &agents.Endpoint{
		Serve: func() (func() error, error) {
			return serveAgents(ctx, cfg, opened, mcp, os.Stdout)
		},
		Showing:      opened.GetShownVault,
		Handler:      opened.API.Answers,
		Unreachable:  func(said string) { opened.API.Unreachable.Store(said) },
		ErrorHandler: func(err error) { fmt.Fprintln(os.Stderr, "numen:", err) },
	}

	going := &going{settle: opened.Settle}

	// Everything behind the settling, in the order each part needs the next: the
	// agents are let go of, then the scan and the follower stop and the database
	// closes, then what they ran under ends.
	behind := shutdown.InOrder(
		reachable.Off,
		func() {
			if err := opened.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "numen:", err)
			}
		},
		stop,
	)

	// The vault settles before a quit is let through, so the steps behind it are
	// asked for here with nothing left owed.
	defer func() {
		going.wait()
		behind.Go()
	}()

	// The window is taken out of sight before the settling begins, and put back
	// where a page calls the close off: the question is asked on the screen the
	// person is looking at.
	var window *application.WebviewWindow
	seen := visibility{
		hide: func() { window.Hide() },
		show: func() { window.Show() },
	}

	app := application.New(application.Options{
		Name: "numen",
		Assets: application.AssetOptions{
			Handler: opened.API.NewHandler(pages),
		},
		// The application ends when its last window closes.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// Run on the thread the window is drawn on, once the application has
		// stopped dispatching. The door on the vault's questions is shut in the
		// step that closes it, and a request arriving after that is refused.
		PostShutdown: behind.Go,
		// A quit that does not come through the window is answered on the
		// thread the page is served on, so the settling happens off it and the
		// quit is asked for again once it is over. A settling that ended with a
		// question standing asks for nothing: the person is answering it, and
		// this refusal is the whole of what a stale goroutine may do.
		ShouldQuit: func() bool {
			return requestQuit(going, seen, func() { application.Get().Quit() })
		},
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  getWindowTitle(opened.GetShownVault(), opened.API.GetOpenTabs()),
		Width:  1280,
		Height: 860,
		URL:    "/",
		// Files a person drags off their desktop reach the page, which marks
		// the places one may be let go of.
		EnableFileDrop: true,
	})

	// Choosing a folder is the machine's own, and it opens over this window.
	opened.API.Vaults.FolderDialog = &folderDialog{window: window}

	// The window is named after what the person is looking at, and is named
	// again each time the page says what it has open.
	naming := func(open domain.OpenTabs) { window.SetTitle(getWindowTitle(opened.GetShownVault(), open)) }
	opened.API.Attends = naming

	// Files let go of over the window, copied into the folder the mark under
	// the pointer names. A drop that landed on no mark is not this window's.
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		landed := event.Context().DropTargetDetails()
		if landed == nil {
			return
		}
		into, marked := landed.Attributes[droppedInto]
		if !marked {
			return
		}
		opened.Imports(ctx, into, event.Context().DroppedFiles())
	})

	// Opening another vault, as a person asks for it. The agents are told which
	// vault they are working when their session opens, so the endpoint they
	// reach it through is stopped and started again around the swap.
	opened.API.Opens = func(ctx context.Context, v domain.Vault) error {
		err := reachable.Around(func() error { return opened.Show(ctx, v) })
		naming(opened.API.GetOpenTabs())
		return err
	}

	// A tool is served where what it works through is there, so the dialog and
	// the swap above stand before the agents are let in.
	//
	// An agent nobody can reach is a panel that says so, not a window that does
	// not open. Everything else the window does is the vault, and the vault is
	// here.
	reachable.On()

	// A hook runs before the window is destroyed and on a thread of its own, so
	// the page is still drawn and still answered while what it owes is written.
	// A cancelled event is where the hooks stop, and the destroy the window
	// registered for itself is one of the listeners after them.
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !closeWindow(ctx, going, seen, opened.WaitForAnswers, window.Close) {
			event.Cancel()
		}
	})

	return app.Run()
}

// droppedInto is the attribute a page marks a drop target with. Its value is
// the folder of the vault a file let go of there is filed in, the root being
// the empty path. The name is the one the window's own drag and drop looks for.
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
	addr string
	off  bool
}
