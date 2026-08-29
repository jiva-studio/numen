// Command numen-review is the window a person runs their cards in.
//
// It stands beside the editor and over the same core (ADR-0030): writing cards
// is occasional and running them is daily, and the daily act is not reached
// through the application built for the other one.
//
// It reads the vault registry and the index and writes neither. The one thing
// it writes into a vault is a mark for a card typed by hand, and the answers,
// which go to the vault's own folder.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/reviewui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func main() {
	var cfg container.Config
	var telling bool
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.BoolVar(&telling, "version", false, "say what this build is and stop")
	flag.Parse()

	if telling {
		fmt.Println(built())
		return
	}

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "numen-review:", err)
		os.Exit(1)
	}
}

func run(cfg container.Config) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	chosen, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.Indexing(chosen.Indexing)

	registry, err := cfg.Registry()
	if err != nil {
		return err
	}

	// The index is opened to be read and never written: what it answers here is
	// which files of a vault are decks, and nothing else. A second writer over
	// the one database every vault shares is what that avoids.
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	running := cfg.Review(db.Queries(), db.Links(), nil)
	api := &reviewui.API{
		Registry:  registry,
		Owed:      running.Owed,
		Session:   running.Session,
		Schedules: running.Schedules,
		Log:       running.Log,
		Opens:     opens,
		Now:       time.Now,
	}

	// The themes are the installation's, and a folder that could not be made
	// leaves the ones this binary ships.
	themes, why := cfg.Themes(func(said string) { fmt.Fprintln(os.Stderr, "themes:", said) })
	if why != nil {
		fmt.Fprintf(os.Stderr, "themes: %v\n", why)
	}
	api.Themes = themes

	pages, err := reviewui.Pages()
	if err != nil {
		return err
	}

	app := application.New(application.Options{
		Name: "numen-review",
		Assets: application.AssetOptions{
			Handler: api.Serving(pages),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — review",
		Width:  760,
		Height: 720,
		URL:    "/",
	})
	return app.Run()
}

// opens brings the editor forward with a deck open at one card.
//
// Between two processes that is an invocation and not a call: the editor is
// started on the vault, and it is the editor that decides what to do with a
// window it may already have open on it.
func opens(_ context.Context, v domain.Vault, deck, card string) error {
	at, err := editor()
	if err != nil {
		return err
	}
	run := exec.Command(at, "-vault", v.ID, "-open", deck, "-card", card)
	run.Stdout, run.Stderr = os.Stdout, os.Stderr
	if err := run.Start(); err != nil {
		return err
	}
	// The editor outlives this call and is not waited for: what it does with
	// the deck is its own, and this window goes on asking cards.
	go func() { _ = run.Wait() }()
	return nil
}

// editor is where the editor stands: beside this binary, which is how the two
// are installed.
func editor() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	at := filepath.Join(filepath.Dir(self), "numen")
	if _, err := os.Stat(at); err != nil {
		return "", errors.New("the editor is not installed beside this application")
	}
	return at, nil
}
