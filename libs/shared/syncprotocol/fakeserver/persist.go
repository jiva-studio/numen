package fakeserver

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver/internal/persisted"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// snapshotFormat is the layout of the saved state: metadata only; the bodies of blobs live in the body folder, or in
// Bodies when the server has no folder. Restore refuses a saved state of another layout.
const snapshotFormat = 2

// SnapshotPrefix is how every saved state begins, so a caller can tell the layout without reading the whole file.
const SnapshotPrefix = `{"format":2,`

// ErrOldFormat is returned by Restore for a saved state of another layout.
var ErrOldFormat = errors.New("the saved state has an old layout")

type snapshot struct {
	Format        int                        `json:"format"`
	Chains        map[string][]byte          `json:"chains"`
	Catalogs      map[string][][]byte        `json:"catalogs"`
	Vaults        map[string]persisted.Vault `json:"vaults"`
	Blobs         map[string]persisted.Blob  `json:"blobs"`
	Used          map[string]uint64          `json:"used"`
	Subscriptions map[string]time.Time       `json:"subscriptions"`
	Plans         map[string]string          `json:"plans"`
	Trials        map[string]bool            `json:"trials"`
	PartLog       []partLogEntry             `json:"partLog"`
	Bodies        map[string][]byte          `json:"bodies,omitempty"`
	Shares        map[string]persisted.Share `json:"shares"`
}

// Snapshot returns the accounts, catalogs, vaults, blobs and subscriptions of the server as bytes that Restore
// reads, so a test process can stop and run again over the same data. Sessions and challenges are not part of it.
func (s *Server) Snapshot() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := snapshot{
		Format: snapshotFormat,
		Chains: map[string][]byte{}, Catalogs: map[string][][]byte{}, Vaults: map[string]persisted.Vault{},
		Blobs: map[string]persisted.Blob{}, Used: s.used, Subscriptions: s.subscriptions,
		Plans: s.planOf, Trials: s.shop.trials, PartLog: s.partLog, Bodies: s.store.GetHeld(), Shares: map[string]persisted.Share{},
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
		out.Vaults[hex.EncodeToString([]byte(id))] = kept
	}
	for id, row := range s.blobs {
		out.Blobs[hex.EncodeToString([]byte(id))] = persisted.Blob{
			Account: row.account, Vault: hex.EncodeToString([]byte(row.vault)), Size: row.size, ExpiresAt: row.expiresAt, State: int(row.state),
			IsUploaded: row.isUploaded, Multipart: snapshotMultipart(row.multipart),
		}
	}
	for id, share := range s.shares {
		kept, err := persisted.EncodeShare(share)
		if err != nil {
			return nil, err
		}
		out.Shares[hex.EncodeToString([]byte(id))] = kept
	}
	return json.Marshal(out)
}

func snapshotVault(vault *vaultState) (persisted.Vault, error) {
	kept := persisted.Vault{Account: vault.account, Seq: vault.seq, PrunedSeq: vault.prunedSeq, Files: map[string][]byte{}, Paths: map[string]string{}}
	for path, id := range vault.paths {
		kept.Paths[hex.EncodeToString([]byte(path))] = hex.EncodeToString([]byte(id))
	}
	for id, stored := range vault.files {
		raw, err := proto.Marshal(stored)
		if err != nil {
			return persisted.Vault{}, fmt.Errorf("keep a revision: %w", err)
		}
		kept.Files[hex.EncodeToString([]byte(id))] = raw
	}
	return kept, nil
}

// Restore replaces the accounts, catalogs, vaults, blobs and subscriptions of the server with what Snapshot returned.
func (s *Server) Restore(raw []byte) error {
	var in snapshot
	if err := json.Unmarshal(raw, &in); err != nil {
		return fmt.Errorf("read the snapshot: %w", err)
	}
	if in.Format != snapshotFormat {
		return ErrOldFormat
	}
	chains, err := persisted.DecodeChains(in.Chains)
	if err != nil {
		return err
	}
	catalogs, err := persisted.DecodeCatalogs(in.Catalogs)
	if err != nil {
		return err
	}
	vaults, err := restoreVaults(in.Vaults)
	if err != nil {
		return err
	}
	blobs, err := restoreBlobs(in.Blobs)
	if err != nil {
		return err
	}
	shares, err := persisted.DecodeShares(in.Shares)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chains, s.catalogs, s.vaults, s.blobs, s.used = chains, catalogs, vaults, blobs, in.Used
	if s.used == nil {
		s.used = map[string]uint64{}
	}
	s.partLog = in.PartLog
	if in.Bodies != nil {
		s.store.SetHeld(in.Bodies)
	}
	s.shares = shares
	s.subscriptions = in.Subscriptions
	if s.subscriptions == nil {
		s.subscriptions = map[string]time.Time{}
	}
	s.restoreShop(in)
	return nil
}

func (s *Server) restoreShop(in snapshot) {
	s.planOf, s.shop.trials = in.Plans, in.Trials
	if s.planOf == nil {
		s.planOf = map[string]string{}
	}
	if s.shop.trials == nil {
		s.shop.trials = map[string]bool{}
	}
}

func restoreVaults(in map[string]persisted.Vault) (map[string]*vaultState, error) {
	out := map[string]*vaultState{}
	for encodedID, kept := range in {
		id, err := persisted.DecodeID(encodedID)
		if err != nil {
			return nil, err
		}
		vault := newVaultState(kept.Account)
		vault.seq, vault.prunedSeq = kept.Seq, kept.PrunedSeq
		for encodedPath, encoded := range kept.Paths {
			path, err := persisted.DecodeID(encodedPath)
			if err != nil {
				return nil, err
			}
			fileID, err := persisted.DecodeID(encoded)
			if err != nil {
				return nil, err
			}
			vault.paths[path] = fileID
		}
		for encodedFileID, raw := range kept.Files {
			fileID, err := persisted.DecodeID(encodedFileID)
			if err != nil {
				return nil, err
			}
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

func restoreBlobs(in map[string]persisted.Blob) (map[string]*blobRow, error) {
	out := map[string]*blobRow{}
	for encodedID, kept := range in {
		id, err := persisted.DecodeID(encodedID)
		if err != nil {
			return nil, err
		}
		vault, err := persisted.DecodeID(kept.Vault)
		if err != nil {
			return nil, err
		}
		out[id] = &blobRow{
			account: kept.Account, vault: vault, size: kept.Size, expiresAt: kept.ExpiresAt, state: blobState(kept.State),
			isUploaded: kept.IsUploaded, multipart: restoreMultipart(kept.Multipart),
		}
	}
	return out, nil
}

func snapshotMultipart(state *multipartState) *persisted.Multipart {
	if state == nil {
		return nil
	}
	return &persisted.Multipart{PartSize: state.partSize, PartCount: state.partCount, Parts: state.parts, IsCompleted: state.isCompleted, IsAborted: state.isAborted}
}

func restoreMultipart(kept *persisted.Multipart) *multipartState {
	if kept == nil {
		return nil
	}
	state := &multipartState{partSize: kept.PartSize, partCount: kept.PartCount, parts: kept.Parts, isCompleted: kept.IsCompleted, isAborted: kept.IsAborted}
	if state.parts == nil {
		state.parts = map[uint32]uint64{}
	}
	return state
}
