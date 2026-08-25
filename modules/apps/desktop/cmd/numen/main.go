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
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

func main() {
	var cfg container.Config
	var agents agentOptions
	var said sizes
	var vault string
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.StringVar(&vault, "vault", "",
		"the vault to open: a name, a path or an identity; the one opened last by default")
	flag.StringVar(&agents.addr, "mcp-addr", defaultAgentAddr,
		"where agents reach this vault; anything but a loopback address opens it to the network")
	flag.BoolVar(&agents.off, "no-mcp", false, "do not let agents reach this vault")
	flag.Float64Var(&said.drawn, "interface", 0,
		"how large the interface is drawn, 1 being as designed; this launch alone")
	flag.Float64Var(&said.set, "font", 0,
		"how large the text a person reads is set, 1 being as designed; this launch alone")
	flag.BoolVar(&cfg.RebuildIndex, "rebuild-index", false,
		"read every file and put it in the index again, whatever the index remembers")
	flag.Parse()

	if err := run(cfg, agents, vault, said); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		refuse(cfg, err)
		os.Exit(1)
	}
}

// sizes are what the command line said about size: how large the interface is
// drawn, and how large the text a person reads is set. Zero is not said, and
// the settings file stands.
type sizes struct{ drawn, set float64 }

// check is what is wrong with a number the setting it says does not take.
func (s sizes) check() error {
	if s.drawn > 0 {
		if err := settings.InterfaceScaleBounds.Check("-interface", s.drawn); err != nil {
			return err
		}
	}
	if s.set > 0 {
		return settings.TextScaleBounds.Check("-font", s.set)
	}
	return nil
}

func run(cfg container.Config, agents agentOptions, vault string, said sizes) error {
	if err := said.check(); err != nil {
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
	cfg.InterfaceScale, cfg.TextScale = said.drawn, said.set

	// Before the window: every page this process reads is read through the
	// runtime made here, and one made after the window reads a page as nothing.
	// A machine holding no runtime yet is told so by the first reading.
	if err := cfg.PrepareRecogniser(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "numen: nothing to read a scan with:", err)
	}

	opened, err := webui.Open(ctx, cfg, vault, os.Stdout)
	if err != nil {
		return err
	}
	defer opened.Close()

	// What reading the settings had to tell a person goes where they are: a
	// window opened from a desktop entry has no terminal to write to.
	opened.Says(chosen.Said)

	// The agents' endpoint on the vault in the window, let in once the window is
	// built.
	reachable := &reaching{ctx: ctx, cfg: cfg, opened: opened, opts: agents, out: os.Stdout}
	defer reachable.off()

	pages, err := webui.Pages()
	if err != nil {
		return err
	}

	// Registered last so that it runs first: what the page owes lands, then the
	// agents are let go of, then the scan and the follower stop and the
	// database closes.
	going := &going{settle: opened.Settle}
	defer func() { going.wait() }()

	app := application.New(application.Options{
		Name: "numen",
		Assets: application.AssetOptions{
			Handler: opened.API.Serving(pages),
		},
		// A quit that does not come through the window is answered on the
		// thread the page is served on, so the settling happens off it and the
		// quit is asked for again once it is over. A settling that ended with a
		// question standing asks for nothing: the person is answering it, and
		// this refusal is the whole of what a stale goroutine may do.
		ShouldQuit: func() bool {
			return asked(going, func() { application.Get().Quit() })
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — " + opened.API.Showing().Name,
		Width:  1280,
		Height: 860,
		URL:    "/",
	})

	// Picking a folder is the machine's own, and it opens over this window.
	opened.API.Choosing = &picker{window: window}

	// Opening another vault, as a person asks for it. The agents are told which
	// vault they are working when their session opens, so the endpoint they
	// reach it through is stopped and started again around the swap.
	opened.API.Opens = func(ctx context.Context, v domain.Vault) error {
		err := reachable.around(func() error { return opened.Show(ctx, v) })
		window.SetTitle("numen — " + opened.Showing().Name)
		return err
	}

	// A tool is served where what it works through is there, so the picker and
	// the swap above stand before the agents are let in.
	//
	// An agent nobody can reach is a panel that says so, not a window that does
	// not open. Everything else the window does is the vault, and the vault is
	// here.
	reachable.on()

	// A hook runs before the window is destroyed and on a thread of its own, so
	// the page is still drawn and still answered while what it owes is written.
	// A cancelled event is where the hooks stop, and the destroy the window
	// registered for itself is one of the listeners after them.
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !closing(ctx, going, opened.Answered, window.Close) {
			event.Cancel()
		}
	})

	return app.Run()
}

// reaching is the agents' endpoint on the vault the window is showing.
//
// What an agent is told about the vault it is working is said once, when its
// session opens, so a window that changes vault stops the endpoint and starts
// it again.
type reaching struct {
	ctx    context.Context
	cfg    container.Config
	opened *webui.Opened
	opts   agentOptions
	out    io.Writer

	// turn is one swap. It is held from the endpoint stopping to the endpoint
	// being served again, so a second swap waits for the first.
	turn sync.Mutex

	mu   sync.Mutex
	shut func() error
}

// around runs one swap with the tools taken away, and serves them again on the
// vault the window then has.
//
// One swap holds this at a time, so the endpoint is started again by the swap
// that stopped it and on the vault that swap ended on.
func (r *reaching) around(swap func() error) error {
	r.turn.Lock()
	defer r.turn.Unlock()

	r.off()
	defer r.on()
	return swap()
}

// on serves the tools against the vault in the window. A window standing on no
// vault serves none.
func (r *reaching) on() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.shut != nil || r.opened.Showing().ID == "" {
		return
	}
	r.opened.API.Unreachable.Store("")
	shut, err := serveAgents(r.ctx, r.cfg, r.opened, r.opts, r.out)
	if err != nil {
		r.opened.API.Unreachable.Store(err.Error())
		fmt.Fprintln(os.Stderr, "numen: no agent:", err)
		return
	}
	r.shut = shut
}

// off stops the endpoint and the agents this window started.
func (r *reaching) off() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.shut == nil {
		return
	}
	if err := r.shut(); err != nil {
		fmt.Fprintln(os.Stderr, "numen: agents:", err)
	}
	r.shut = nil
	r.opened.API.Answers(nil)
}

// closing is the window being asked to go, and answers with whether it may.
//
// A page holding text a person has to answer for calls the close off, and the
// close is asked for again once they have answered. That wait is on a person
// and is not measured.
func closing(
	ctx context.Context,
	g *going,
	answered func(context.Context) bool,
	again func(),
) bool {
	if g.wait() {
		return true
	}
	go func() {
		if answered(ctx) {
			again()
		}
	}()
	return false
}

// asked is a quit that did not come through the window, and answers with
// whether the application may go.
//
// It is answered on the thread the page is served on, so the settling happens
// off it and the quit is asked for again once it is over. A settling that ended
// with a question standing asks for nothing: the person is answering it, and
// this refusal is the whole of what the goroutine left behind may do.
func asked(g *going, quit func()) bool {
	if g.settled() {
		return true
	}
	go func() {
		if g.wait() {
			quit()
		}
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
	over    chan struct{}
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
		this = &turn{over: make(chan struct{})}
		g.turn = this
		go g.begin(this)
	}
	g.mu.Unlock()

	<-this.over
	return this.settled
}

// settled reports whether there is nothing left owed.
func (g *going) settled() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.done
}

func (g *going) begin(this *turn) {
	defer close(this.over)

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
