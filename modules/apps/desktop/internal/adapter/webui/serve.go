package webui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

// Open puts together everything the window needs: the vault it shows, the
// questions it may ask, and a scan running behind it.
//
// The scan is started rather than waited for. A vault of a hundred thousand
// notes takes a minute and a half, and the first note is answerable long before
// that.
//
// Closing stops the scan, waits for it, and then closes the database — in that
// order, because the database is what the scan writes to.
func Open(ctx context.Context, cfg container.Config, out io.Writer) (*API, func() error, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return nil, nil, err
	}
	vaults, err := usecase.List{Registry: registry}.Execute()
	if err != nil {
		return nil, nil, err
	}
	if len(vaults) == 0 {
		return nil, nil, fmt.Errorf("no vault to open — add one with: numen vault add <path>")
	}

	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return nil, nil, err
	}

	watching, stop := context.WithCancel(ctx)
	api := &API{Vault: vaults[0], Notes: db.Queries(), Links: db.Links()}
	scan := usecase.Scan{
		Readers:     cfg.VaultReaders(),
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
		OnProgress: func(res usecase.ScanResult) {
			api.Indexed.Store(int64(res.Indexed))
		},
	}
	api.Scan = scan.Execute

	// Following the vault is a use case; this adapter only says who hears about
	// it. Whatever a change turns out to mean is decided in one place, so a
	// second way of showing a vault does not decide it again.
	follow := usecase.Follow{
		Watcher: cfg.VaultWatcher(),
		Refresh: usecase.Refresh{Readers: cfg.VaultReaders(), Notes: db.Notes()},
		Scan:    scan,
		Changed: func(m usecase.Moved) {
			api.Listeners.tell(changed{paths: m.Paths, reload: m.Reload})
		},
		Trouble: func(err error) {
			if err == nil {
				api.Failed.Store("")
				return
			}
			api.Failed.Store(err.Error())
		},
	}

	// Watching begins before the scan does, so that an edit made while the
	// vault is being read is held rather than missed.
	//
	// A vault that cannot be watched is still a vault: the application keeps
	// working, and says that changes will not appear by themselves.
	following, err := follow.Begin(watching, api.Vault)
	if err != nil {
		fmt.Fprintf(out, "not watching %s: %v\n", api.Vault.Name, err)
		api.Unwatched.Store(err.Error())
	}

	var running sync.WaitGroup
	running.Add(1)
	go func() {
		defer running.Done()

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

		// The scan goes first. It writes in groups from what it holds, so its
		// copy of a note lands last however early the note was read.
		if following != nil {
			following.Run(watching)
		}
	}()

	return api, func() error {
		stop()
		running.Wait()
		return db.Close()
	}, nil
}

// Showing is the vault the window has open.
func (a *API) Showing() domain.Vault { return a.Vault }
