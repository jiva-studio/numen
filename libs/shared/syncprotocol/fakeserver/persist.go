package fakeserver

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver/internal/persisted"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver/internal/sharestate"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

type snapshot struct {
	Chains        map[string][]byte          `json:"chains"`
	Catalogs      map[string][][]byte        `json:"catalogs"`
	Vaults        map[string]persisted.Vault `json:"vaults"`
	Blobs         map[string]persisted.Blob  `json:"blobs"`
	Used          map[string]uint64          `json:"used"`
	Subscriptions map[string]time.Time       `json:"subscriptions"`
	Plans         map[string]string          `json:"plans"`
	Trials        map[string]bool            `json:"trials"`
	PartLog       []partLogEntry             `json:"partLog"`
	Bodies        map[string][]byte          `json:"bodies"`
	Shares        map[string]persisted.Share `json:"shares"`
}

// Snapshot returns the accounts, catalogs, vaults, blobs and subscriptions of the server as bytes that Restore
// reads, so a test process can stop and run again over the same data. Sessions and challenges are not part of it.
func (s *Server) Snapshot() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := snapshot{
		Chains: map[string][]byte{}, Catalogs: map[string][][]byte{}, Vaults: map[string]persisted.Vault{},
		Blobs: map[string]persisted.Blob{}, Used: s.used, Subscriptions: s.subscriptions,
		Plans: s.planOf, Trials: s.shop.trials, PartLog: s.partLog, Bodies: s.store.Held, Shares: map[string]persisted.Share{},
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
			Account: row.account, Vault: hex.EncodeToString([]byte(row.vault)), Size: row.size, ExpiresAt: row.expiresAt, State: int(row.state), Body: row.body,
			IsUploaded: row.isUploaded, Multipart: snapshotMultipart(row.multipart),
		}
	}
	for id, share := range s.shares {
		kept, err := snapshotShare(share)
		if err != nil {
			return nil, err
		}
		out.Shares[hex.EncodeToString([]byte(id))] = kept
	}
	return json.Marshal(out)
}

func snapshotShare(share *sharestate.State) (persisted.Share, error) {
	kept := persisted.Share{
		Account: share.Account, Vault: hex.EncodeToString([]byte(share.Vault)), Epoch: share.Epoch, IsClosed: share.IsClosed, Members: share.Members,
		Exits: map[string]uint64{},
	}
	for file, seq := range share.Exits {
		kept.Exits[hex.EncodeToString([]byte(file))] = seq
	}
	for _, entry := range share.Log {
		raw, err := proto.Marshal(entry)
		if err != nil {
			return persisted.Share{}, fmt.Errorf("keep a share log entry: %w", err)
		}
		kept.Log = append(kept.Log, raw)
	}
	return kept, nil
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
	blobs, err := restoreBlobs(in.Blobs)
	if err != nil {
		return err
	}
	shares, err := restoreShares(in.Shares)
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
		s.store.Held = in.Bodies
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

func restoreVaults(in map[string]persisted.Vault) (map[string]*vaultState, error) {
	out := map[string]*vaultState{}
	for encodedID, kept := range in {
		id, err := decodeID(encodedID)
		if err != nil {
			return nil, err
		}
		vault := newVaultState(kept.Account)
		vault.seq, vault.prunedSeq = kept.Seq, kept.PrunedSeq
		for encodedPath, encoded := range kept.Paths {
			path, err := decodeID(encodedPath)
			if err != nil {
				return nil, err
			}
			fileID, err := decodeID(encoded)
			if err != nil {
				return nil, err
			}
			vault.paths[path] = fileID
		}
		for encodedFileID, raw := range kept.Files {
			fileID, err := decodeID(encodedFileID)
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
		id, err := decodeID(encodedID)
		if err != nil {
			return nil, err
		}
		vault, err := decodeID(kept.Vault)
		if err != nil {
			return nil, err
		}
		out[id] = &blobRow{
			account: kept.Account, vault: vault, size: kept.Size, expiresAt: kept.ExpiresAt, state: blobState(kept.State),
			body: kept.Body, isUploaded: kept.IsUploaded, multipart: restoreMultipart(kept.Multipart),
		}
	}
	return out, nil
}

func decodeID(encoded string) (string, error) {
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("read an id of the snapshot: %w", err)
	}
	return string(raw), nil
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

func restoreShares(in map[string]persisted.Share) (map[string]*sharestate.State, error) {
	out := map[string]*sharestate.State{}
	for encodedID, kept := range in {
		id, err := decodeID(encodedID)
		if err != nil {
			return nil, err
		}
		vault, err := decodeID(kept.Vault)
		if err != nil {
			return nil, err
		}
		share := sharestate.New(kept.Account, vault)
		share.Epoch, share.IsClosed, share.Members = kept.Epoch, kept.IsClosed, kept.Members
		for encodedFile, seq := range kept.Exits {
			file, err := decodeID(encodedFile)
			if err != nil {
				return nil, err
			}
			share.Exits[file] = seq
		}
		for _, raw := range kept.Log {
			entry := &syncv1.ShareLogEntry{}
			if err := proto.Unmarshal(raw, entry); err != nil {
				return nil, fmt.Errorf("read a share log entry: %w", err)
			}
			share.Log = append(share.Log, entry)
		}
		out[id] = share
	}
	return out, nil
}
