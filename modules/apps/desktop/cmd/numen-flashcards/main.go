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
	"net/http"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/version"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func main() {
	cfg := makeConfig(os.Stderr)
	var showVersion, noAgent bool
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.BoolVar(&noAgent, "no-agent", false, "do not let a card be asked about")
	flag.BoolVar(&showVersion, "version", false, "say what this build is and stop")
	flag.Parse()

	if showVersion {
		fmt.Println(version.GetVersionLine("numen-flashcards"))
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
	return cfg.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), vaults.updateIndex),
		cfg.Cards(db.Queries(), db.Links(), vaults.updateIndex)
}

func run(cfg container.Config, noAgent bool) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	chosen, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.SetSettings(chosen)

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
	vaults := &openVaults{cfg: cfg, db: db, lifetime: ctx, out: os.Stderr}

	notes, cutting := makeNotesAndCards(cfg, db, vaults)
	api := makeWindowAPI(cfg, registry, db, notes, vaults)
	bindVaultIndexer(ctx, api, vaults)

	// The agent reaches the cards through tools on a port this window opens, and
	// is started and stopped around a session.
	away := serveAgents(ctx, cfg, db, notes, cutting, api, noAgent, os.Stderr)

	held := makeShutdown(away, stop, vaults, db)
	defer held.Run()

	followChanges(ctx, cfg, api)
	loadThemes(cfg, api)

	pages, err := window.Pages()
	if err != nil {
		return err
	}
	return runApplication(api, pages, held)
}

// makeWindowAPI is the page's API over the cards, the notes and the registry.
func makeWindowAPI(
	cfg container.Config, registry port.VaultRegistry, db *container.Index,
	notes container.Notes, vaults *openVaults,
) *window.API {
	running := cfg.Flashcards(db.Queries(), db.Links(), db.Problems(), vaults.updateIndex)
	return &window.API{
		Registry:      registry,
		CardsDue:      running.CardsDue,
		Session:       running.Session,
		Schedules:     running.Schedules,
		Log:           running.Log,
		Counted:       running.Counted,
		Neighbourhood: cfg.ShowNeighbourhood(notes.Links, db.Queries(), notes.Read),
		Presets:       running.Presets,
		Notes:         db.Queries(),
		Window:        window.NewWindow(nil),
		Day:           running.Day,
		Now:           time.Now,
	}
}

// bindVaultIndexer connects the walk of a vault into the index to the API.
// The walk outlives the count that asked for it, and a vault that moved is
// counted again and walked again where the last walk failed.
func bindVaultIndexer(ctx context.Context, api *window.API, vaults *openVaults) {
	vaults.recordVault = func(v domain.Vault) {
		api.ClearIndexFailure(v.ID)
		api.ReportChange()
	}
	api.SetVaultIndexer(ctx, vaults.indexVault)
}

// makeShutdown stops the agents, then the walk and the watch, then closes the
// index they write to.
func makeShutdown(
	away func() error, stop func(), vaults *openVaults, db *container.Index,
) *shutdown.Sequence {
	return shutdown.NewSequence(
		func() {
			if err := away(); err != nil {
				fmt.Fprintln(os.Stderr, "numen-flashcards: agents:", err)
			}
		},
		stop,
		vaults.closeAndWait,
		func() {
			if err := db.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "numen-flashcards:", err)
			}
		},
	)
}

// followChanges counts again what the window draws from when it changes, so a
// card changed or a deck written is counted without a person asking. The index
// is followed for which files are decks, and another window writing it changes
// that answer.
func followChanges(ctx context.Context, cfg container.Config, api *window.API) {
	moves, err := cfg.Moves(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "numen-flashcards: the index is not being followed:", err)
		return
	}
	api.ReportChangesFrom(ctx, moves)
}

// loadThemes gives the API the installation's themes. A folder that could not
// be made leaves the ones this binary ships.
func loadThemes(cfg container.Config, api *window.API) {
	themes, why := cfg.Themes(func(said string) { fmt.Fprintln(os.Stderr, "themes:", said) })
	if why != nil {
		fmt.Fprintf(os.Stderr, "themes: %v\n", why)
	}
	api.Themes = themes
}

// runApplication opens the window and runs until it closes.
func runApplication(api *window.API, pages http.Handler, held *shutdown.Sequence) error {
	app := application.New(application.Options{
		Name: "numen-flashcards",
		Assets: application.AssetOptions{
			Handler: api.NewHandler(pages),
		},
		// The application ends when its last window closes.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// Run on the thread the window is drawn on, once the application has
		// stopped dispatching. The walk and the watch are waited for with no
		// bound, and a window not yet destroyed holds until they are done.
		PostShutdown: held.Run,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — flashcards",
		Width:  760,
		Height: 720,
		URL:    "/",
	})
	return app.Run()
}
