// Command numen-flashcards is the window a person runs their cards in.
//
// It stands beside the editor and over the same core: writing cards is
// occasional and running them is daily, and the daily act is not reached
// through the application built for the other one.
//
// It writes into the vault it is sitting to — a mark for a card that carries
// none — and the answers, which go to the vault's own folder. Each write is
// levelled in the index before it returns, so what the window draws next is
// what it just wrote. A vault the index does not carry at all is read into it
// here.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/flashcardsui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func main() {
	var cfg container.Config
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
	defer db.Close()
	level := cfg.Level(db)

	running := cfg.Flashcards(db.Queries(), db.Links(), level)
	api := &flashcardsui.API{
		Registry:  registry,
		Owed:      running.Owed,
		Session:   running.Session,
		Schedules: running.Schedules,
		Log:       running.Log,
		Counted:   running.Counted,
		Joined: flashcards.Around{
			Linked: note.ShowLinks{Links: db.Links()},
			Notes:  db.Queries(),
			Reads:  note.Read{Readers: cfg.VaultReaders()},
		},
		Presets: running.Presets,
		Curves:  running.Curves,
		Notes:   db.Queries(),
		Tasking: task.New(),
		Day:     running.Day,
		Now:     time.Now,
	}

	// A vault the index does not carry is walked into it here, over the same
	// scan the editor and the command line walk one with. The reading outlives
	// the count that asked for it, so it runs for the life of the window.
	scan := cfg.Scan(db)
	api.Reading(ctx, func(ctx context.Context, v domain.Vault, got func(int64)) error {
		walk := scan
		walk.OnProgress = func(res vault.ScanResult) { got(int64(res.Indexed)) }
		_, err := walk.Execute(ctx, v)
		return err
	})

	// A card is asked about through tools on a port this window opens for
	// itself. The agent works the vault the person sat down to, so it is
	// started and stopped around a sitting.
	away := serveAgents(ctx, cfg, db, api, noAgent, os.Stderr)
	defer func() {
		if err := away(); err != nil {
			fmt.Fprintln(os.Stderr, "numen-flashcards: agents:", err)
		}
	}()

	// What the window draws from is followed while it is open, so a card
	// changed or a deck written is counted again without a person asking.
	//
	// The vaults, for the cards themselves, which are read from their files.
	held, err := registry.All()
	if err != nil {
		return err
	}
	watcher := cfg.VaultWatcher()
	for _, v := range held {
		changes, lost, err := watcher.Watch(ctx, v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "numen-flashcards: %s is not being followed: %v\n", v.Name, err)
			continue
		}
		api.Follows(ctx, drop(changes))
		api.Follows(ctx, lost)
	}

	// And the index, for which files are decks. That answer is the index's, and
	// another window writing it changes it.
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

	pages, err := flashcardsui.Pages()
	if err != nil {
		return err
	}

	app := application.New(application.Options{
		Name: "numen-flashcards",
		Assets: application.AssetOptions{
			Handler: api.Serving(pages),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — flashcards",
		Width:  760,
		Height: 720,
		URL:    "/",
	})
	return app.Run()
}

// drop is a channel of paths as a channel of nothing: what moved is not carried
// past here, because the page asks what the vaults come to whatever it was.
func drop(paths <-chan []string) <-chan struct{} {
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		for range paths {
			select {
			case out <- struct{}{}:
			default:
			}
		}
	}()
	return out
}
