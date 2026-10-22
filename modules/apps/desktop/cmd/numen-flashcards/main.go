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

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/themes"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/version"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "numen-flashcards:", err)
		os.Exit(1)
	}
}

// makeConfig is what this binary starts from: where the core says what it went
// wrong at and carried on past, which is the same place everything else this
// binary could not do is said.
func makeConfig(out io.Writer) container.Config {
	cfg := platform.GetConfigFor(platform.TargetFlashcards)
	cfg.ErrorHandler = func(err error) { fmt.Fprintln(out, "numen-flashcards:", err) }
	return cfg
}

// makeNotes is everything that acts on the vault's notes, built once and in one place. The page and
// the tools an agent calls are served these and build none of their own, so a dependency named here
// is named for both.
//
// What a write touched is levelled through the opening the vault was opened with, which is what the
// walk and the watch also go through.
func makeNotes(cfg container.Config, db *container.Index, vaults *container.OpenVaults) container.Notes {
	return cfg.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), vaults.UpdateIndex)
}

func run(cfg container.Config) error {
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
	vaults := &container.OpenVaults{
		Config: cfg, Index: db, Lifetime: ctx, Out: os.Stderr, Label: "numen-flashcards",
	}

	notes := makeNotes(cfg, db, vaults)
	api := makeWindowAPI(cfg, notes)
	bindVaultIndexer(ctx, api, vaults)
	lent, err := openPluginVault(cfg, registry, db, api)
	if err != nil {
		return err
	}
	cfg.SetPluginVault(lent)

	// The walk and the watch, then the index they write to closes.
	target := shutdown.NewSequence(
		stop,
		vaults.CloseAndWait,
		shutdown.Report("numen-flashcards", os.Stderr, "", db.Close),
	)
	defer target.Run()

	followChanges(ctx, cfg, api)
	api.Themes = themes.Load(cfg.Themes, func(response string) { fmt.Fprintln(os.Stderr, "themes:", response) }, os.Stderr)

	pages, err := window.Pages()
	if err != nil {
		return err
	}
	return runApplication(api, pages, target)
}

// makeWindowAPI is the window's API: the notes it answers about and the plugins' services it forwards.
func makeWindowAPI(cfg container.Config, notes container.Notes) *window.API {
	return &window.API{
		IsVaultIndexed: notes.IsVaultIndexed,
		Window:         window.NewWindow(nil),
		PluginServices: cfg.GetPluginServices(),
	}
}

// openPluginVault is the vault the plugins work through the host. The window shows none of its own,
// so a plugin names the vault of each call; the host reads a vault into the index when it is asked
// about one the index does not carry, and tells the plugin when something moved.
func openPluginVault(
	cfg container.Config, registry port.VaultRegistry, db *container.Index, api *window.API,
) (port.PluginVault, error) {
	return cfg.OpenPluginVault(registry, db,
		func() (domain.Vault, bool) { return domain.Vault{}, false },
		container.PluginVaultHooks{
			IndexState: api.EnsureIndexed,
			Moved:      api.WatchMoves,
			InUse: func(ctx context.Context, v domain.Vault) {
				if openedVault := api.Opened; openedVault != nil {
					openedVault(ctx, v)
				}
			},
		})
}

// bindVaultIndexer connects the walk of a vault into the index to the API.
// The walk outlives the count that asked for it, and a vault that moved is
// counted again and walked again where the last walk failed.
func bindVaultIndexer(ctx context.Context, api *window.API, vaults *container.OpenVaults) {
	vaults.RecordVault = func(v domain.Vault) {
		api.ClearIndexFailure(v.ID)
		api.ReportChange()
	}
	api.SetVaultIndexer(ctx, vaults.IndexVault)
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

// runApplication opens the window and runs until it closes.
func runApplication(api *window.API, pages http.Handler, target *shutdown.Sequence) error {
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
		PostShutdown: target.Run,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — flashcards",
		Width:  760,
		Height: 720,
		URL:    "/",
	})
	return app.Run()
}
