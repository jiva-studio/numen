// Command numen-flashcards is the window a person runs their cards in.
//
// It stands beside the editor and over the same core: writing cards is
// occasional and running them is daily, and the daily act is not reached
// through the application built for the other one.
//
// It writes into the vault its session is on — a mark for a card that carries
// none — and the answers, which go to the vault's own folder. Each write is
// levelled in the index before it returns, so what the window draws next is
// what it just wrote. A vault the index does not carry at all is read into it
// here.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/version"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

func main() {
	cfg := makeConfig(os.Stderr)
	var telling, noAgent bool
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.BoolVar(&noAgent, "no-agent", false, "do not let a card be asked about")
	flag.BoolVar(&telling, "version", false, "say what this build is and stop")
	flag.Parse()

	if telling {
		fmt.Println(version.Built("numen-flashcards"))
		return
	}

	if err := run(cfg, noAgent); err != nil {
		fmt.Fprintln(os.Stderr, "numen-flashcards:", err)
		os.Exit(1)
	}
}

// makeConfig is what this binary starts from: where the core says what it went
// wrong at and carried on past, which is the same place everything else this
// binary could not do is said.
func makeConfig(out io.Writer) container.Config {
	return container.Config{
		ErrorHandler: func(err error) { fmt.Fprintln(out, "numen-flashcards:", err) },
	}
}

// makeNotesAndCards is everything that acts on the vault's notes and cards,
// built once and in one place. The page and the tools an agent calls are served
// these and build none of their own, so a dependency named here is named for
// both.
//
// What a write touched is levelled through the opening the vault was opened
// with, which is what the walk and the watch also go through.
func makeNotesAndCards(
	cfg container.Config, db *container.Index, vaults *openVaults,
) (container.Notes, container.Cards) {
	return cfg.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), vaults.level),
		cfg.Cards(db.Queries(), db.Links(), vaults.level)
}

func run(cfg container.Config, noAgent bool) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	chosen, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.Indexing(chosen.Indexing)
	cfg.Agent = chosen.Agent

	registry, err := cfg.Registry()
	if err != nil {
		return err
	}

	// The index is opened to be written as well as read: a note this window
	// writes is levelled here, and the editor's window may be open over the same
	// file at the same time.
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}

	// Every vault this window shows is opened the way the editor opens the one
	// it shows: watched from the moment it is opened, walked into the index, and
	// levelled by the paths a write touches.
	vaults := &openVaults{cfg: cfg, db: db, under: ctx, out: os.Stderr}

	notes, cutting := makeNotesAndCards(cfg, db, vaults)

	running := cfg.Flashcards(db.Queries(), db.Links(), db.Problems(), vaults.level)
	api := &window.API{
		Registry:      registry,
		CardsDue:      running.CardsDue,
		Session:       running.Session,
		Schedules:     running.Schedules,
		Log:           running.Log,
		Counted:       running.Counted,
		Neighbourhood: flashcards.NewShowNeighbourhood(notes.Links, db.Queries(), notes.Read),
		Presets:       running.Presets,
		Notes:         db.Queries(),
		Window:        window.Watching(task.New()),
		Day:           running.Day,
		Now:           time.Now,
	}

	// A vault is walked into the index before it is counted. The walk outlives
	// the count that asked for it, so it runs for the life of the window.
	//
	// A vault that moved is counted again, and is one whose walk is worth trying
	// again where the last one failed.
	vaults.record = func(v domain.Vault) {
		api.Forget(v.ID)
		api.Moved()
	}
	api.Reading(ctx, vaults.reads)

	// A card is asked about through tools on a port this window opens for
	// itself. The agent works the vault the person sat down to, so it is
	// started and stopped around a session.
	away := serveAgents(ctx, cfg, db, notes, cutting, api, noAgent, os.Stderr)

	// What the window holds, in the order each part needs the next: the agents
	// are let go of, then the walk and the watch, which write to the index, and
	// then the index itself.
	held := shutdown.InOrder(
		func() {
			if err := away(); err != nil {
				fmt.Fprintln(os.Stderr, "numen-flashcards: agents:", err)
			}
		},
		stop,
		vaults.wait,
		func() {
			if err := db.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "numen-flashcards:", err)
			}
		},
	)
	defer held.Go()

	// What the window draws from is followed while it is open, so a card changed
	// or a deck written is counted again without a person asking. The vaults are
	// followed by the openings they were opened through.
	//
	// The index is followed here, for which files are decks. That answer is the
	// index's, and another window writing it changes it.
	moves, err := cfg.Moves(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "numen-flashcards: the index is not being followed:", err)
	} else {
		api.Follows(ctx, moves)
	}

	// The themes are the installation's, and a folder that could not be made
	// leaves the ones this binary ships.
	themes, why := cfg.Themes(func(said string) { fmt.Fprintln(os.Stderr, "themes:", said) })
	if why != nil {
		fmt.Fprintf(os.Stderr, "themes: %v\n", why)
	}
	api.Themes = themes

	pages, err := window.Pages()
	if err != nil {
		return err
	}

	app := application.New(application.Options{
		Name: "numen-flashcards",
		Assets: application.AssetOptions{
			Handler: api.Serving(pages),
		},
		// The application ends when its last window closes.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// Run on the thread the window is drawn on, once the application has
		// stopped dispatching. The walk and the watch are waited for with no
		// bound, and a window not yet destroyed holds until they are done.
		PostShutdown: held.Go,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — flashcards",
		Width:  760,
		Height: 720,
		URL:    "/",
	})
	return app.Run()
}
