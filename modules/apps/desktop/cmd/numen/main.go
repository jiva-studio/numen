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
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func main() {
	cfg := configured(os.Stderr)
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
		fmt.Println(version.Built("numen"))
		return
	}

	if err := run(cfg, letting, vault, said); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		refuse(cfg, err)
		os.Exit(1)
	}
}

// configured is what this binary starts from: what the machine supplies the
// core, and where the core says what it went wrong at and carried on past. That
// is the same place everything else this binary could not do is said.
func configured(out io.Writer) container.Config {
	cfg := platform.Config()
	cfg.Trouble = func(err error) { fmt.Fprintln(out, "numen:", err) }
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

	// Before anything draws: the settings a folder picker reads are looked for
	// once, the first time something asks for one.
	findSchemas()

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	chosen, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.Indexing(chosen.Indexing)
	cfg.Agent = chosen.Agent
	cfg.InterfaceScale, cfg.TextScale = sizes.interfaceScale, sizes.textScale

	// Before the window: every page this process reads is read through the
	// runtime made here, and one made after the window reads a page as nothing.
	// A machine holding no runtime yet is told so by the first reading.
	if err := cfg.PrepareRecogniser(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "numen: nothing to read a scan with:", err)
	}
	pages, err := webui.Pages()
	if err != nil {
		return err
	}

	opened, err := webui.Open(ctx, cfg, vault, os.Stdout)
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
		Showing:     opened.Showing,
		Handler:     opened.API.Answers,
		Unreachable: func(said string) { opened.API.Unreachable.Store(said) },
		Trouble:     func(err error) { fmt.Fprintln(os.Stderr, "numen:", err) },
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
			Handler: opened.API.Serving(pages),
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
			return asked(going, seen, func() { application.Get().Quit() })
		},
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  titled(opened.Showing(), opened.API.Attended()),
		Width:  1280,
		Height: 860,
		URL:    "/",
		// Files a person drags off their desktop reach the page, which marks
		// the places one may be let go of.
		EnableFileDrop: true,
	})

	// Picking a folder is the machine's own, and it opens over this window.
	opened.API.Vaults.Picker = &picker{window: window}

	// The window is named after what the person is looking at, and is named
	// again each time the page says what it has open.
	naming := func(open domain.Attention) { window.SetTitle(titled(opened.Showing(), open)) }
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
		opened.Brings(ctx, into, event.Context().DroppedFiles())
	})

	// Opening another vault, as a person asks for it. The agents are told which
	// vault they are working when their session opens, so the endpoint they
	// reach it through is stopped and started again around the swap.
	opened.API.Opens = func(ctx context.Context, v domain.Vault) error {
		err := reachable.Around(func() error { return opened.Show(ctx, v) })
		naming(opened.API.Attended())
		return err
	}

	// A tool is served where what it works through is there, so the picker and
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
		if !closing(ctx, going, seen, opened.Answered, window.Close) {
			event.Cancel()
		}
	})

	return app.Run()
}

// droppedInto is the attribute a page marks a drop target with. Its value is
// the folder of the vault a file let go of there is filed in, the root being
// the empty path. The name is the one the window's own drag and drop looks for.
const droppedInto = "data-file-drop-target"

// titled is what the window is called: the application, and the file the
// person is looking at. A window with no file in front of it is called after
// the vault it is showing.
func titled(v domain.Vault, open domain.Attention) string {
	if front, held := open.Fronted(); held && front.Path != "" {
		return "numen — " + path.Base(front.Path)
	}
	if v.Name == "" {
		return "numen"
	}
	return "numen — " + v.Name
}

// visibility is the window going out of sight and coming back into it. Hiding
// leaves the page drawing and answering, so what only it holds is handed over
// after the window is gone from the screen.
type visibility struct {
	hide func()
	show func()
}

// closing is the window being asked to go, and answers with whether it may.
//
// The window goes out of sight first and the vault settles behind it. A page
// holding text a person has to answer for calls the close off, the window comes
// back, and the close is asked for again once they have answered. That wait is
// on a person and is not measured.
func closing(
	ctx context.Context,
	g *going,
	s visibility,
	answered func(context.Context) bool,
	retry func(),
) bool {
	s.hide()
	if g.wait() {
		return true
	}
	s.show()
	go func() {
		if answered(ctx) {
			retry()
		}
	}()
	return false
}

// asked is a quit that did not come through the window, and answers with
// whether the application may go.
//
// It is answered on the thread the page is served on, so the window is hidden
// and the settling happens off it, and the quit is asked for again once it is
// over. A settling that ended with a question standing puts the window back and
// asks for nothing: the person is answering it, and that is the whole of what
// the goroutine left behind may do.
func asked(g *going, s visibility, quit func()) bool {
	if g.settled() {
		return true
	}
	go func() {
		s.hide()
		if g.wait() {
			quit()
			return
		}
		s.show()
	}()
	return false
}

// quitBound is how long the window waits for a page that says nothing to write
// what only it holds. A vault being changed waits under the same bound.
const quitBound = webui.HandedOverIn

// going is the vault settling, whichever way the window is asked to go.
//
// A settling that ends with a question standing leaves the vault as it was, and
// the next ask begins another one.
type going struct {
	settle func(context.Context) bool

	mu   sync.Mutex
	turn *turn
	done bool
}

// turn is one settling, and what it answered.
type turn struct {
	done    chan struct{}
	settled bool
}

// wait settles the vault and answers with whether it did. A settling already
// running is joined and its answer shared.
func (g *going) wait() bool {
	g.mu.Lock()
	if g.done {
		g.mu.Unlock()
		return true
	}
	this := g.turn
	if this == nil {
		this = &turn{done: make(chan struct{})}
		g.turn = this
		go g.begin(this)
	}
	g.mu.Unlock()

	<-this.done
	return this.settled
}

// settled reports whether there is nothing left owed.
func (g *going) settled() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.done
}

func (g *going) begin(this *turn) {
	defer close(this.done)

	ctx, cancel := context.WithTimeout(context.Background(), quitBound)
	defer cancel()
	settled := g.settle(ctx)

	g.mu.Lock()
	defer g.mu.Unlock()
	this.settled = settled
	g.done = settled
	g.turn = nil
}

// agentOptions is what the person said about letting agents in.
type agentOptions struct {
	addr string
	off  bool
}
