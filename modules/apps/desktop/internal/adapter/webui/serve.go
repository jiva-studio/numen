package webui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
)

// Opened is a vault put together and running: the questions a client may ask,
// and the pieces anything else working the same vault needs.
type Opened struct {
	API   *API
	Vault domain.Vault
	Index *container.Index
	// Refresh brings named notes up to date. Whatever changes a note calls it,
	// so that what changed is findable before the change is reported done.
	Refresh usecase.Refresh
	// Embedder turns text into vectors, for filling the index and for turning a
	// query into one. It is the same embedder for both, so a query's vector and
	// the stored vectors come from one model. Nil for an installation with none,
	// and then every search is answered by words alone.
	Embedder port.Embedder
	// Close stops the scan, waits for it, and closes the index.
	Close func() error
}

// Open puts together everything the window needs: the vault it shows, the
// questions it may ask, and a scan running behind it.
//
// The scan is started and left running. A vault of a hundred thousand notes
// takes a minute and a half, and the first note is answerable long before that.
//
// Closing stops the scan, waits for it, and then closes the database — in that
// order, because the database is what the scan writes to.
func Open(ctx context.Context, cfg container.Config, out io.Writer) (*Opened, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return nil, err
	}
	vaults, err := usecase.List{Registry: registry}.Execute()
	if err != nil {
		return nil, err
	}
	if len(vaults) == 0 {
		return nil, fmt.Errorf("no vault to open — add one with: numen-cli vault add <path>")
	}

	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return nil, err
	}

	// Opened once, for as long as the window is. A local model holds a session
	// that takes seconds to build, and both filling the index and answering a
	// query need it.
	embedder, closeEmbedder, why := cfg.Embedder()
	if why != nil {
		fmt.Fprintf(out, "not embedding %s: %v\n", vaults[0].Name, why)
	}
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}

	// wanted carries one nudge: something that is not a note changed, and the
	// vault's books are to be read again.
	wanted := make(chan struct{}, 1)

	watching, stop := context.WithCancel(ctx)
	api := &API{
		Vault:     vaults[0],
		Notes:     db.Queries(),
		Links:     db.Links(),
		Listeners: following(),
		Watching:  focusing(),
		Progress:  db.Progress(),
	}
	// Named before anything is read: it is what decides whether a chunk already
	// carries a vector, and what tells the window that something is going to
	// embed what was cut.
	if embedder != nil {
		model := embedder.Model()
		// The vector index is built for one width. A model of another width
		// rebuilds it, and what it held is embedded again.
		if err := db.FitVectors(ctx, model.Dimensions); err != nil {
			fmt.Fprintf(out, "not embedding %s: %v\n", vaults[0].Name, err)
			embedder = nil
		} else {
			api.Model.Store(model.String())
		}
	}
	scan := usecase.Scan{
		Readers:      cfg.VaultReaders(),
		Vaults:       db.Vaults(),
		Notes:        db.Notes(),
		Known:        db.Queries(),
		Maintenance:  db.Maintenance(),
		RebuildIndex: cfg.RebuildIndex,
		OnProgress: func(res usecase.ScanResult) {
			api.Indexed.Store(int64(res.Indexed))
		},
	}
	api.Scan = scan.Execute

	// Following the vault is a use case; this adapter only says who hears about
	// it. Whatever a change turns out to mean is decided in one place, so a
	// second way of showing a vault does not decide it again.
	refresh := usecase.Refresh{Readers: cfg.VaultReaders(), Notes: db.Notes()}
	follow := usecase.Follow{
		Watcher: cfg.VaultWatcher(),
		Refresh: refresh,
		Scan:    scan,
		Changed: func(m usecase.Moved) {
			api.Listeners.tell(changed{paths: m.Paths, reload: m.Reload})
			if m.Sources {
				// A book dropped into an open vault is read without anybody
				// asking. One nudge is enough: what owes work is asked of the
				// index, so several changes at once are one reading.
				select {
				case wanted <- struct{}{}:
				default:
				}
			}
		},
		Trouble: func(err error) {
			if err == nil {
				api.Failed.Store("")
				return
			}
			api.Failed.Store(err.Error())
		},
	}

	// Watching begins before the scan does, so an edit made while the vault is
	// being read is held.
	//
	// A vault that cannot be watched is still a vault: the application keeps
	// working, and says that changes will not appear by themselves.
	following, beginErr := follow.Begin(watching, api.Vault)
	if beginErr != nil {
		fmt.Fprintf(out, "not watching %s: %v\n", api.Vault.Name, beginErr)
		api.Unwatched.Store(beginErr.Error())
	}

	// Set before the goroutine starts, so that a client asking between opening
	// and the first read is told there is more to come.
	api.Busy.Store(true)

	var running sync.WaitGroup
	running.Add(1)
	go func() {
		defer running.Done()
		// Set false on every way out of the reading, including the ways that
		// return early.
		defer api.Busy.Store(false)

		result, err := scan.Execute(watching, api.Vault)
		api.Indexed.Store(int64(result.Indexed))
		switch {
		case err == nil:
			fmt.Fprintf(out, "%s: %d notes\n", api.Vault.Name, result.Seen)
			api.Ready.Store(true)
		case errors.Is(err, context.Canceled):
			// Asked to stop. What it stored is correct as far as it got, and
			// there is nothing to report.
			return
		default:
			api.Failed.Store(err.Error())
			return
		}

		// Reading the sources comes after the notes: a vault is useful the
		// moment its notes answer, and a library takes minutes to cut and hours
		// to embed. Neither stops the window, and neither has to finish: an
		// index is a cache.
		readSources(watching, cfg, db, api, embedder, out)
		// Everything this vault owed is read. Watching it goes on for as long as
		// the window is open, and a change to a file is reported as a change.
		api.Busy.Store(false)

		// The scan goes first. It writes in groups from what it holds, so its
		// copy of a note lands last however early the note was read.
		if following != nil {
			running.Add(1)
			go func() {
				defer running.Done()
				following.Run(watching)
			}()
		}

		// A book that arrives while the window is open is read where the first
		// reading was: one at a time, and never while another is running.
		for {
			select {
			case <-watching.Done():
				return
			case <-wanted:
				api.Busy.Store(true)
				readSources(watching, cfg, db, api, embedder, out)
				api.Busy.Store(false)
			}
		}
	}()

	return &Opened{
		API:      api,
		Vault:    api.Vault,
		Index:    db,
		Refresh:  refresh,
		Embedder: embedder,
		Close: func() error {
			stop()
			running.Wait()
			// The embedder goes after the work that uses it and before the
			// database, which is the order they depend on each other in.
			err := closeEmbedder()
			if closed := db.Close(); err == nil {
				err = closed
			}
			return err
		},
	}, nil
}

// readSources takes the text out of every book in the vault and then embeds what
// was cut, reporting what it is reading as it goes.
//
// Both halves are allowed to fail without the window minding. A book that will
// not parse is one book; an embedder that is not configured is the ordinary case,
// and search answers on words alone until one is.
func readSources(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	api *API,
	embedder port.Embedder,
	out io.Writer,
) {
	// A window is cut under the limit of the model that will read it. Without a
	// model the default bound stands: what is cut now is what a model of any width
	// is later given.
	sizes := window.Sizes{}
	if embedder != nil {
		sizes.Limit = window.Under(embedder.Model().MaxTokens)
	}

	extract := source.Extract{
		Readers:      cfg.VaultReaders(),
		Sources:      db.Sources(),
		Owing:        db.SourcesKnown(),
		Sizes:        sizes,
		RebuildIndex: cfg.RebuildIndex,
		OnProgress: func(res source.ExtractResult) {
			api.Reading.Store(res.Reading)
			api.Books.Store(int64(res.Seen))
			// Every book the walk found leaves this pass one of four ways, and
			// all four count as done.
			api.BooksRead.Store(int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished))
		},
	}
	if res, err := extract.Execute(ctx, api.Vault); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintf(out, "reading the sources of %s: %v\n", api.Vault.Name, err)
		}
	} else if res.Extracted > 0 {
		fmt.Fprintf(out, "%s: %d books, %d chunks\n", api.Vault.Name, res.Extracted, res.Chunks)
	}
	api.Reading.Store("")

	if embedder == nil {
		return
	}

	embed := source.Embed{
		Readers:  cfg.VaultReaders(),
		Chunks:   db.VectorsOwing(),
		Vectors:  db.Vectors(),
		Embedder: embedder,
		OnProgress: func(res source.EmbedResult) {
			api.Reading.Store(res.Reading)
		},
	}
	api.Learning.Store(true)
	if _, err := embed.Execute(ctx, api.Vault); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "embedding %s: %v\n", api.Vault.Name, err)
	}
	api.Learning.Store(false)
	api.Reading.Store("")
}

// Showing is the vault the window has open.
func (a *API) Showing() domain.Vault { return a.Vault }
