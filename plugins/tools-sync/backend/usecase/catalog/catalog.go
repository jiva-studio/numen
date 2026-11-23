// Package catalog lists the synced vaults of the account, connects a local vault to one and keeps its name.
package catalog

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const maxTries = 3

var (
	// ErrInvalidCatalog means the catalog of the server is not a signed, unbroken log of the devices of the account.
	ErrInvalidCatalog = errors.New("the catalog of the account is not valid")
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
	Catalog  ports.Catalog
	Chain    ports.Chain
	Crypto   ports.Crypto
	Identity ports.Identity
	Host     ports.Host
	State    ports.Links
	Journals ports.Journals
	Random   io.Reader
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
	if err := checkEntries(entries, authors); err != nil {
		return nil, err
	}
	return entries, r.keep(ctx, entries)
}

func checkEntries(entries []*syncv1.CatalogEntry, authors trust.Authors) error {
	var previous [32]byte
	for i, entry := range entries {
		if err := checkEntry(entry, uint64(i)+1, previous, authors); err != nil {
			return fmt.Errorf("%w: entry %d: %w", ErrInvalidCatalog, i+1, err)
		}
		previous = syncprotocol.GetCatalogEntryID(entry)
	}
	return nil
}

var (
	errWrongVersion  = errors.New("the version does not follow the one before")
	errWrongPrevious = errors.New("the previous hash is not the id of the entry before")
	errBadSignature  = errors.New("the signature is not valid")
	errUnknownAuthor = errors.New("the author is not a device of the account")
)

func checkEntry(entry *syncv1.CatalogEntry, version uint64, previous [32]byte, authors trust.Authors) error {
	if err := syncprotocol.CheckCatalogEntry(entry); err != nil {
		return err
	}
	if entry.GetVersion() != version {
		return errWrongVersion
	}
	if version > 1 && !bytes.Equal(entry.GetPrevHash(), previous[:]) {
		return errWrongPrevious
	}
	if len(entry.GetSignature()) != 64 || !identity.VerifyForPlugin(entry.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID,
		syncprotocol.GetCatalogEntryMessage(entry), [64]byte(entry.GetSignature())) {
		return errBadSignature
	}
	if !authors.IsTrusted(entry.GetAuthorDeviceSignKey()) {
		return errUnknownAuthor
	}
	return nil
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
			vaults = append(vaults, remote{vaultID: entry.GetVaultId(), last: nil, body: vaultlink.CatalogBody{LocalID: "", Name: "", Key: nil}})
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
	listed := make([]vaultlink.RemoteVault, len(vaults))
	for i, one := range vaults {
		id := hex.EncodeToString(one.vaultID)
		listed[i] = vaultlink.RemoteVault{RemoteID: id, Name: one.body.Name, IsConnected: connected[id]}
	}
	return listed, nil
}
