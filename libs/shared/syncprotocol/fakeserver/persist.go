package fakeserver

import (
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/identity"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

type snapshot struct {
	Chains        map[string][]byte        `json:"chains"`
	Catalogs      map[string][][]byte      `json:"catalogs"`
	Vaults        map[string]vaultSnapshot `json:"vaults"`
	Blobs         map[string]blobSnapshot  `json:"blobs"`
	Used          map[string]uint64        `json:"used"`
	Subscriptions map[string]time.Time     `json:"subscriptions"`
	Shares        map[string]shareSnapshot `json:"shares"`
}

type shareSnapshot struct {
	Account  string            `json:"account"`
	Vault    string            `json:"vault"`
	Epoch    uint32            `json:"epoch"`
	IsClosed bool              `json:"isClosed"`
	Log      [][]byte          `json:"log"`
	Members  []*shareMember    `json:"members"`
	Exits    map[string]uint64 `json:"exits"`
}

type vaultSnapshot struct {
	Account string            `json:"account"`
	Seq     uint64            `json:"seq"`
	Files   map[string][]byte `json:"files"`
	Paths   map[string]string `json:"paths"`
}

type blobSnapshot struct {
	Account    string    `json:"account"`
	Vault      string    `json:"vault"`
	Size       uint64    `json:"size"`
	ExpiresAt  time.Time `json:"expiresAt"`
	State      int       `json:"state"`
	Body       []byte    `json:"body"`
	IsUploaded bool      `json:"isUploaded"`
}

// Snapshot returns the accounts, catalogs, vaults, blobs and subscriptions of the server as bytes that Restore
// reads, so a test process can stop and run again over the same data. Sessions and challenges are not part of it.
func (s *Server) Snapshot() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := snapshot{
		Chains: map[string][]byte{}, Catalogs: map[string][][]byte{}, Vaults: map[string]vaultSnapshot{},
		Blobs: map[string]blobSnapshot{}, Used: s.used, Subscriptions: s.subscriptions, Shares: map[string]shareSnapshot{},
	}
	for account, chain := range s.chains {
		out.Chains[account] = chain.Bytes()
	}
	for account, entries := range s.catalogs {
		for _, entry := range entries {
			raw, err := proto.Marshal(entry)
			if err != nil {
				return nil, fmt.Errorf("keep a catalog entry: %w", err)
			}
			out.Catalogs[account] = append(out.Catalogs[account], raw)
		}
	}
	for id, vault := range s.vaults {
		kept, err := snapshotVault(vault)
		if err != nil {
			return nil, err
		}
		out.Vaults[id] = kept
	}
	for id, row := range s.blobs {
		out.Blobs[id] = blobSnapshot{
			Account: row.account, Vault: row.vault, Size: row.size, ExpiresAt: row.expiresAt, State: int(row.state), Body: row.body,
			IsUploaded: row.isUploaded,
		}
	}
	for id, share := range s.shares {
		kept, err := snapshotShare(share)
		if err != nil {
			return nil, err
		}
		out.Shares[id] = kept
	}
	return json.Marshal(out)
}

func snapshotShare(share *shareState) (shareSnapshot, error) {
	kept := shareSnapshot{
		Account: share.account, Vault: share.vault, Epoch: share.epoch, IsClosed: share.isClosed, Members: share.members, Exits: share.exits,
	}
	for _, entry := range share.log {
		raw, err := proto.Marshal(entry)
		if err != nil {
			return shareSnapshot{}, fmt.Errorf("keep a share log entry: %w", err)
		}
		kept.Log = append(kept.Log, raw)
	}
	return kept, nil
}

func snapshotVault(vault *vaultState) (vaultSnapshot, error) {
	kept := vaultSnapshot{Account: vault.account, Seq: vault.seq, Files: map[string][]byte{}, Paths: vault.paths}
	for id, stored := range vault.files {
		raw, err := proto.Marshal(stored)
		if err != nil {
			return vaultSnapshot{}, fmt.Errorf("keep a revision: %w", err)
		}
		kept.Files[id] = raw
	}
	return kept, nil
}

// Restore replaces the accounts, catalogs, vaults, blobs and subscriptions of the server with what Snapshot returned.
func (s *Server) Restore(raw []byte) error {
	var in snapshot
	if err := json.Unmarshal(raw, &in); err != nil {
		return fmt.Errorf("read the snapshot: %w", err)
	}
	chains, err := restoreChains(in.Chains)
	if err != nil {
		return err
	}
	catalogs, err := restoreCatalogs(in.Catalogs)
	if err != nil {
		return err
	}
	vaults, err := restoreVaults(in.Vaults)
	if err != nil {
		return err
	}
	shares, err := restoreShares(in.Shares)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chains, s.catalogs, s.vaults, s.blobs, s.used = chains, catalogs, vaults, restoreBlobs(in.Blobs), in.Used
	if s.used == nil {
		s.used = map[string]uint64{}
	}
	s.shares = shares
	s.subscriptions = in.Subscriptions
	if s.subscriptions == nil {
		s.subscriptions = map[string]time.Time{}
	}
	return nil
}

func restoreChains(in map[string][]byte) (map[string]identity.Chain, error) {
	out := map[string]identity.Chain{}
	for account, raw := range in {
		chain, err := identity.ParseChain(raw)
		if err != nil {
			return nil, fmt.Errorf("read a chain: %w", err)
		}
		out[account] = chain
	}
	return out, nil
}

func restoreCatalogs(in map[string][][]byte) (map[string][]*syncv1.CatalogEntry, error) {
	out := map[string][]*syncv1.CatalogEntry{}
	for account, raws := range in {
		for _, raw := range raws {
			entry := &syncv1.CatalogEntry{}
			if err := proto.Unmarshal(raw, entry); err != nil {
				return nil, fmt.Errorf("read a catalog entry: %w", err)
			}
			out[account] = append(out[account], entry)
		}
	}
	return out, nil
}

func restoreVaults(in map[string]vaultSnapshot) (map[string]*vaultState, error) {
	out := map[string]*vaultState{}
	for id, kept := range in {
		vault := newVaultState(kept.Account)
		vault.seq = kept.Seq
		if kept.Paths != nil {
			vault.paths = kept.Paths
		}
		for fileID, raw := range kept.Files {
			stored := &syncv1.StoredRevision{}
			if err := proto.Unmarshal(raw, stored); err != nil {
				return nil, fmt.Errorf("read a revision: %w", err)
			}
			vault.files[fileID] = stored
		}
		out[id] = vault
	}
	return out, nil
}

func restoreBlobs(in map[string]blobSnapshot) map[string]*blobRow {
	out := map[string]*blobRow{}
	for id, kept := range in {
		out[id] = &blobRow{
			account: kept.Account, vault: kept.Vault, size: kept.Size, expiresAt: kept.ExpiresAt, state: blobState(kept.State),
			body: kept.Body, isUploaded: kept.IsUploaded,
		}
	}
	return out
}

func restoreShares(in map[string]shareSnapshot) (map[string]*shareState, error) {
	out := map[string]*shareState{}
	for id, kept := range in {
		share := newShareState(kept.Account, kept.Vault)
		share.epoch, share.isClosed, share.members = kept.Epoch, kept.IsClosed, kept.Members
		if kept.Exits != nil {
			share.exits = kept.Exits
		}
		for _, raw := range kept.Log {
			entry := &syncv1.ShareLogEntry{}
			if err := proto.Unmarshal(raw, entry); err != nil {
				return nil, fmt.Errorf("read a share log entry: %w", err)
			}
			share.log = append(share.log, entry)
		}
		out[id] = share
	}
	return out, nil
}
