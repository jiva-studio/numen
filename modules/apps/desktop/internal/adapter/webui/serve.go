package webui

import (
	"context"
	"errors"
	"fmt"
	"io"

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

	scanning, stop := context.WithCancel(ctx)
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

	done := make(chan struct{})
	go func() {
		defer close(done)
		result, err := scan.Execute(scanning, api.Vault)
		api.Indexed.Store(int64(result.Indexed))
		switch {
		case err == nil:
			fmt.Fprintf(out, "%s: %d notes\n", api.Vault.Name, result.Seen)
			api.Ready.Store(true)
		case errors.Is(err, context.Canceled):
			// Asked to stop. What it stored is correct as far as it got, and
			// there is nothing to report.
		default:
			api.Failed.Store(err.Error())
		}
	}()

	return api, func() error {
		stop()
		<-done
		return db.Close()
	}, nil
}

// Showing is the vault the window has open.
func (a *API) Showing() domain.Vault { return a.Vault }
