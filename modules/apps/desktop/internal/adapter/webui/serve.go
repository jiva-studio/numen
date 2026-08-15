package webui

import (
	"context"
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
	go func() {
		result, err := scan.Execute(ctx, api.Vault)
		if err != nil {
			fmt.Fprintf(out, "scan: %v\n", err)
		} else {
			fmt.Fprintf(out, "%s: %d notes\n", api.Vault.Name, result.Seen)
		}
		api.Indexed.Store(int64(result.Indexed))
		api.Ready.Store(true)
	}()

	return api, db.Close, nil
}

// Vault is what the window is showing.
func (a *API) Showing() domain.Vault { return a.Vault }
