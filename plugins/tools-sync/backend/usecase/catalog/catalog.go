// Package catalog lists the synced vaults of the account, connects a local vault to one and keeps its name.
package catalog

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const maxTries = 3

var (
	// ErrInvalidCatalog means the catalog of the server is not a signed, unbroken log of the devices of the account.
	ErrInvalidCatalog = vaultlink.ErrInvalidCatalog
	// ErrUnknownRemoteVault means the account has no synced vault of the id.
	ErrUnknownRemoteVault = errors.New("the account has no such synced vault")
	// ErrCopyOfConnectedVault means a vault is a copy of a folder that is connected already.
	ErrCopyOfConnectedVault = errors.New("a copy of a connected vault")
	// errAlreadyConnected means the vault is connected.
	errAlreadyConnected = errors.New("the vault is connected already")
	// ErrCatalogBusy means other devices appended to the catalog first, every time.
	ErrCatalogBusy = errors.New("the catalog changed on another device again and again")
	// errUnknownMode means a vault is to be connected in a way that is not known.
	errUnknownMode = errors.New("the mode of the connection is not known")
)

// Deps are what the use cases of the catalog need.
type Deps struct {
	Catalog    ports.Catalog
	Chain      ports.Chain
	Crypto     ports.Crypto
	Identity   ports.Identity
	Host       ports.Host
	State      ports.Links
	Journals   ports.Journals
	Cloud      ports.CloudVaults
	Challenges ports.DeleteChallenges
	Forget     ports.Forgetter
	Health     ports.Health
	Random     io.Reader
	Locker     ports.VaultLocker
}

type remote struct {
	vaultID []byte
	last    *syncv1.CatalogEntry
	body    vaultlink.CatalogBody
}

type reader struct{ deps Deps }

func (r reader) read(ctx context.Context) ([]*syncv1.CatalogEntry, error) {
	own, err := r.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	stored, err := r.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return nil, err
	}
	authors, err := trust.NewAuthors(own.Chain, stored)
	if err != nil {
		return nil, err
	}
	entries, err := r.deps.Catalog.GetCatalog(ctx, 0)
	if err != nil {
		return nil, err
	}
	if err := vaultlink.CheckEntries(entries, authors); err != nil {
		return nil, err
	}
	return entries, r.keep(ctx, entries)
}

func (r reader) mergeEntries(ctx context.Context, entries []*syncv1.CatalogEntry) ([]remote, error) {
	var vaults []remote
	positions := map[string]int{}
	for _, entry := range entries {
		key := string(entry.GetVaultId())
		at, known := positions[key]
		if !known {
			at = len(vaults)
			positions[key] = at
			vaults = append(vaults, remote{vaultID: entry.GetVaultId(), last: nil, body: vaultlink.CatalogBody{LocalID: "", Name: "", Key: nil, Keys: nil, Removed: false}})
		}
		vaults[at].last = entry
	}
	for i := range vaults {
		body, err := r.openBody(ctx, vaults[i].last)
		if err != nil {
			return nil, err
		}
		vaults[i].body = body
	}
	return vaults, nil
}

func (r reader) openBody(ctx context.Context, entry *syncv1.CatalogEntry) (vaultlink.CatalogBody, error) {
	plain, err := r.deps.Crypto.OpenData(ctx, vaultlink.GetCatalogContext(entry.GetVaultId(), entry.GetVersion()), entry.GetEncBody())
	if err != nil {
		return vaultlink.CatalogBody{}, fmt.Errorf("open the body of catalog entry %d: %w", entry.GetVersion(), err)
	}
	return vaultlink.ParseCatalogBody(plain)
}

func (r reader) readLinks(ctx context.Context) (map[domain.VaultID]vaultlink.Link, error) {
	vaults, err := r.deps.Host.GetVaults(ctx)
	if err != nil {
		return nil, err
	}
	links := map[domain.VaultID]vaultlink.Link{}
	for _, vault := range vaults {
		link, err := r.deps.State.ReadLink(ctx, vault.ID)
		if errors.Is(err, vaultlink.ErrNotConnected) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if link.Kind == vaultlink.KindReceived {
			continue
		}
		links[vault.ID] = link
	}
	return links, nil
}

// ListRemoteVaults lists the synced vaults of the account.
type ListRemoteVaults struct{ reader reader }

// NewListRemoteVaults returns the use case over the dependencies.
func NewListRemoteVaults(deps Deps) *ListRemoteVaults {
	return &ListRemoteVaults{reader: reader{deps: deps}}
}

// Execute returns the synced vaults with the last name of each and whether a vault of this device is
// connected to it.
func (u *ListRemoteVaults) Execute(ctx context.Context) ([]vaultlink.RemoteVault, error) {
	entries, err := u.reader.read(ctx)
	if err != nil {
		return nil, err
	}
	vaults, err := u.reader.mergeEntries(ctx, entries)
	if err != nil {
		return nil, err
	}
	links, err := u.reader.readLinks(ctx)
	if err != nil {
		return nil, err
	}
	connected := map[string]bool{}
	for _, link := range links {
		connected[link.RemoteVaultID] = true
	}
	usage, err := u.reader.readUsage(ctx)
	if err != nil {
		return nil, err
	}
	listed := make([]vaultlink.RemoteVault, 0, len(vaults))
	for _, one := range vaults {
		if one.body.Removed {
			continue
		}
		id := hex.EncodeToString(one.vaultID)
		listed = append(listed, vaultlink.RemoteVault{RemoteID: id, Name: one.body.Name, IsConnected: connected[id], SizeBytes: usage[id]})
	}
	return listed, nil
}

func (r reader) readUsage(ctx context.Context) (map[string]uint64, error) {
	rows, err := r.deps.Cloud.GetVaultUsage(ctx)
	if err != nil {
		return nil, err
	}
	usage := make(map[string]uint64, len(rows))
	for _, row := range rows {
		usage[hex.EncodeToString(row.GetVaultId())] = row.GetUsedBytes()
	}
	return usage, nil
}
