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
	PartLog       []partLogEntry           `json:"partLog"`
	Bodies        map[string][]byte        `json:"bodies"`
}

type multipartSnapshot struct {
	PartSize    uint64            `json:"partSize"`
	PartCount   uint32            `json:"partCount"`
	Parts       map[uint32]uint64 `json:"parts"`
	IsCompleted bool              `json:"isCompleted"`
	IsAborted   bool              `json:"isAborted"`
}

type vaultSnapshot struct {
	Account   string            `json:"account"`
	Seq       uint64            `json:"seq"`
	PrunedSeq uint64            `json:"prunedSeq"`
	Files     map[string][]byte `json:"files"`
	Paths     map[string]string `json:"paths"`
}

type blobSnapshot struct {
	Account    string             `json:"account"`
	Vault      string             `json:"vault"`
	Size       uint64             `json:"size"`
	ExpiresAt  time.Time          `json:"expiresAt"`
	State      int                `json:"state"`
	Body       []byte             `json:"body"`
	IsUploaded bool               `json:"isUploaded"`
	Multipart  *multipartSnapshot `json:"multipart"`
}

// Snapshot returns the accounts, catalogs, vaults, blobs and subscriptions of the server as bytes that Restore
// reads, so a test process can stop and run again over the same data. Sessions and challenges are not part of it.
func (s *Server) Snapshot() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := snapshot{
		Chains: map[string][]byte{}, Catalogs: map[string][][]byte{}, Vaults: map[string]vaultSnapshot{},
		Blobs: map[string]blobSnapshot{}, Used: s.used, Subscriptions: s.subscriptions, PartLog: s.partLog,
		Bodies: s.store.held,
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
			IsUploaded: row.isUploaded, Multipart: snapshotMultipart(row.multipart),
		}
	}
	return json.Marshal(out)
}

func snapshotVault(vault *vaultState) (vaultSnapshot, error) {
	kept := vaultSnapshot{Account: vault.account, Seq: vault.seq, PrunedSeq: vault.prunedSeq, Files: map[string][]byte{}, Paths: vault.paths}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chains, s.catalogs, s.vaults, s.blobs, s.used = chains, catalogs, vaults, restoreBlobs(in.Blobs), in.Used
	if s.used == nil {
		s.used = map[string]uint64{}
	}
	s.partLog = in.PartLog
	if in.Bodies != nil {
		s.store.held = in.Bodies
	}
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
		vault.seq, vault.prunedSeq = kept.Seq, kept.PrunedSeq
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
			body: kept.Body, isUploaded: kept.IsUploaded, multipart: restoreMultipart(kept.Multipart),
		}
	}
	return out
}

func snapshotMultipart(state *multipartState) *multipartSnapshot {
	if state == nil {
		return nil
	}
	return &multipartSnapshot{PartSize: state.partSize, PartCount: state.partCount, Parts: state.parts, IsCompleted: state.isCompleted, IsAborted: state.isAborted}
}

func restoreMultipart(kept *multipartSnapshot) *multipartState {
	if kept == nil {
		return nil
	}
	state := &multipartState{partSize: kept.PartSize, partCount: kept.PartCount, parts: kept.Parts, isCompleted: kept.IsCompleted, isAborted: kept.IsAborted}
	if state.parts == nil {
		state.parts = map[uint32]uint64{}
	}
	return state
}
