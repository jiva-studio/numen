// Package hoststore keeps what Numen Sync knows of a vault in the store folder of the plugin in that vault, as
// whole JSON files.
package hoststore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	linkFile      = "link.json"
	recordsFile   = "files.json"
	statusFile    = "status.json"
	sightingsFile = "notes.json"
)

// State is the state of each vault in its store folder.
type State struct {
	host ports.Host
}

// New returns the state kept through the host.
func New(vaults ports.Host) *State { return &State{host: vaults} }

type recordsDocument struct {
	Files []note.FileRecord `json:"files"`
}

type statusDocument struct {
	LastSyncedAt  time.Time `json:"lastSyncedAt"`
	ConflictCount int       `json:"conflictCount"`
}

type sightingsDocument struct {
	Notes map[string]note.Seen `json:"notes"`
}

func (s *State) read(ctx context.Context, id domain.VaultID, name string, into any) (bool, error) {
	raw, err := s.host.Store(id).Read(ctx, name)
	if errors.Is(err, host.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return false, fmt.Errorf("read %s: %w", name, err)
	}
	return true, nil
}

func (s *State) write(ctx context.Context, id domain.VaultID, name string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := s.host.Store(id).Write(ctx, name, raw); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
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

// ReadRecords returns what is known of the synced notes of the vault.
func (s *State) ReadRecords(ctx context.Context, id domain.VaultID) ([]note.FileRecord, error) {
	var document recordsDocument
	_, err := s.read(ctx, id, recordsFile, &document)
	return document.Files, err
}

// WriteRecords replaces what is known of the synced notes of the vault.
func (s *State) WriteRecords(ctx context.Context, id domain.VaultID, records []note.FileRecord) error {
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
