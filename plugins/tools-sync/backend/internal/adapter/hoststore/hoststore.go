// Package hoststore keeps what Numen Sync knows of a vault in the store folder of the plugin in that vault, as
// whole JSON files.
package hoststore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	linkFile      = "link.json"
	recordsFile   = "files.json"
	statusFile    = "status.json"
	sightingsFile = "notes.json"
	sharesFile    = "shares.json"
)

// State is the state of each vault in its store folder. It is the only writer of those files: every read and write of
// a vault runs under the lock of that vault, a write whose bytes equal the bytes last read or written is skipped, and
// UpdateLink reads, changes and writes the link as one step.
type State struct {
	host ports.Host

	mu     sync.Mutex
	vaults map[domain.VaultID]*vaultFiles
}

type vaultFiles struct {
	mu      sync.Mutex
	pass    chan struct{}
	digests map[string][sha256.Size]byte
}

// New returns the state kept through the host.
func New(vaults ports.Host) *State {
	return &State{host: vaults, mu: sync.Mutex{}, vaults: map[domain.VaultID]*vaultFiles{}}
}

func (s *State) getVault(id domain.VaultID) *vaultFiles {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, isKnown := s.vaults[id]
	if !isKnown {
		files = &vaultFiles{mu: sync.Mutex{}, pass: make(chan struct{}, 1), digests: map[string][sha256.Size]byte{}}
		s.vaults[id] = files
	}
	return files
}

// LockVault waits until no other holder has the vault and returns the function that releases it, or the error of ctx.
// A holder is one pass over the vault; it does not block the reads and writes of the state.
func (s *State) LockVault(ctx context.Context, id domain.VaultID) (func(), error) {
	files := s.getVault(id)
	select {
	case files.pass <- struct{}{}:
		return func() { <-files.pass }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type recordsDocument struct {
	Files []transfer.FileRecord `json:"files"`
}

type statusDocument struct {
	LastSyncedAt  time.Time `json:"lastSyncedAt"`
	ConflictCount int       `json:"conflictCount"`
}

type sightingsDocument struct {
	Notes map[string]note.Seen `json:"notes"`
}

type sharesDocument struct {
	Shares []share.Progress `json:"shares"`
}

func (s *State) read(ctx context.Context, id domain.VaultID, name string, into any) (bool, error) {
	files := s.getVault(id)
	files.mu.Lock()
	defer files.mu.Unlock()
	return s.readUnderLock(ctx, id, files, name, into)
}

func (s *State) readUnderLock(ctx context.Context, id domain.VaultID, files *vaultFiles, name string, into any) (bool, error) {
	raw, err := s.host.Store(id).Read(ctx, name)
	if errors.Is(err, host.ErrNotExist) {
		delete(files.digests, name)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return false, fmt.Errorf("read %s: %w", name, err)
	}
	files.digests[name] = sha256.Sum256(raw)
	return true, nil
}

func (s *State) write(ctx context.Context, id domain.VaultID, name string, value any) error {
	files := s.getVault(id)
	files.mu.Lock()
	defer files.mu.Unlock()
	return s.writeUnderLock(ctx, id, files, name, value)
}

func (s *State) writeUnderLock(ctx context.Context, id domain.VaultID, files *vaultFiles, name string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	digest := sha256.Sum256(raw)
	if last, isKnown := files.digests[name]; isKnown && last == digest {
		return nil
	}
	if err := s.host.Store(id).Write(ctx, name, raw); err != nil {
		delete(files.digests, name)
		return fmt.Errorf("write %s: %w", name, err)
	}
	files.digests[name] = digest
	return nil
}

// ReadLink returns the link of the vault, or vaultlink.ErrNotConnected when it has none.
func (s *State) ReadLink(ctx context.Context, id domain.VaultID) (vaultlink.Link, error) {
	var link vaultlink.Link
	found, err := s.read(ctx, id, linkFile, &link)
	if err != nil {
		return vaultlink.Link{}, err
	}
	if !found {
		return vaultlink.Link{}, vaultlink.ErrNotConnected
	}
	return link, nil
}

func (s *State) WriteLink(ctx context.Context, id domain.VaultID, link vaultlink.Link) error {
	return s.write(ctx, id, linkFile, link)
}

// UpdateLink reads the link of the vault, runs change on it and writes the result as one step, so no other write of the
// link falls between the read and the write. It returns vaultlink.ErrNotConnected when the vault has no link.
func (s *State) UpdateLink(ctx context.Context, id domain.VaultID, change func(*vaultlink.Link) error) error {
	files := s.getVault(id)
	files.mu.Lock()
	defer files.mu.Unlock()
	var link vaultlink.Link
	found, err := s.readUnderLock(ctx, id, files, linkFile, &link)
	if err != nil {
		return err
	}
	if !found {
		return vaultlink.ErrNotConnected
	}
	if err := change(&link); err != nil {
		return err
	}
	return s.writeUnderLock(ctx, id, files, linkFile, link)
}

// ReadRecords returns what is known of the synced notes of the vault.
func (s *State) ReadRecords(ctx context.Context, id domain.VaultID) ([]transfer.FileRecord, error) {
	var document recordsDocument
	_, err := s.read(ctx, id, recordsFile, &document)
	return document.Files, err
}

// WriteRecords replaces what is known of the synced notes of the vault.
func (s *State) WriteRecords(ctx context.Context, id domain.VaultID, records []transfer.FileRecord) error {
	return s.write(ctx, id, recordsFile, recordsDocument{Files: records})
}

// ReadStatus returns how the last pass over the vault ended.
func (s *State) ReadStatus(ctx context.Context, id domain.VaultID) (ports.Status, error) {
	var document statusDocument
	if _, err := s.read(ctx, id, statusFile, &document); err != nil {
		return ports.Status{}, err
	}
	return ports.Status{LastSyncedAt: document.LastSyncedAt, ConflictCount: document.ConflictCount}, nil
}

// WriteStatus replaces how the last pass over the vault ended.
func (s *State) WriteStatus(ctx context.Context, id domain.VaultID, status ports.Status) error {
	return s.write(ctx, id, statusFile, statusDocument{LastSyncedAt: status.LastSyncedAt, ConflictCount: status.ConflictCount})
}

// ReadSightings returns the notes of the vault as the last look found them.
func (s *State) ReadSightings(ctx context.Context, id domain.VaultID) (map[string]note.Seen, error) {
	var document sightingsDocument
	if _, err := s.read(ctx, id, sightingsFile, &document); err != nil {
		return nil, err
	}
	if document.Notes == nil {
		return map[string]note.Seen{}, nil
	}
	return document.Notes, nil
}

// WriteSightings replaces the notes of the vault as the last look found them.
func (s *State) WriteSightings(ctx context.Context, id domain.VaultID, seen map[string]note.Seen) error {
	return s.write(ctx, id, sightingsFile, sightingsDocument{Notes: seen})
}

// ReadShareStates returns where this device stands with the shares it made in the vault.
func (s *State) ReadShareStates(ctx context.Context, id domain.VaultID) ([]share.Progress, error) {
	var document sharesDocument
	_, err := s.read(ctx, id, sharesFile, &document)
	return document.Shares, err
}

// WriteShareStates replaces where this device stands with the shares it made in the vault.
func (s *State) WriteShareStates(ctx context.Context, id domain.VaultID, states []share.Progress) error {
	return s.write(ctx, id, sharesFile, sharesDocument{Shares: states})
}

// ForgetVault deletes the files of the vault in its store folder that belong to the vault itself, its link last, so a
// failure in between leaves a vault that is still linked. The pinned keys and the journals of the account stay.
func (s *State) ForgetVault(ctx context.Context, id domain.VaultID) error {
	for _, name := range []string{
		recordsFile, statusFile, sightingsFile, uploadsFile, downloadsFile, pendingFile, answerFile, absencesFile, linkFile,
	} {
		if err := s.delete(ctx, id, name); err != nil {
			return err
		}
	}
	return nil
}
