package hoststore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

const (
	uploadsFile   = "uploads.json"
	downloadsFile = "downloads.json"
	pendingFile   = "pending-deletions.json"
	answerFile    = "answer-deletions.json"
	absencesFile  = "absences.json"
)

type uploadsDocument struct {
	Uploads []transfer.UploadEntry `json:"uploads"`
}

type downloadsDocument struct {
	Downloads []transfer.DownloadEntry `json:"downloads"`
}

type absencesDocument struct {
	Absent map[string]int64 `json:"absent"`
}

func (s *State) ReadUploads(ctx context.Context, id domain.VaultID) ([]transfer.UploadEntry, error) {
	var document uploadsDocument
	_, err := s.read(ctx, id, uploadsFile, &document)
	return document.Uploads, err
}

// WriteUploads replaces the uploads of the vault that are not finished.
func (s *State) WriteUploads(ctx context.Context, id domain.VaultID, entries []transfer.UploadEntry) error {
	return s.write(ctx, id, uploadsFile, uploadsDocument{Uploads: entries})
}

func (s *State) ReadDownloads(ctx context.Context, id domain.VaultID) ([]transfer.DownloadEntry, error) {
	var document downloadsDocument
	_, err := s.read(ctx, id, downloadsFile, &document)
	return document.Downloads, err
}

// WriteDownloads replaces the downloads of the vault that are not finished.
func (s *State) WriteDownloads(ctx context.Context, id domain.VaultID, entries []transfer.DownloadEntry) error {
	return s.write(ctx, id, downloadsFile, downloadsDocument{Downloads: entries})
}

// ReadPending returns the removals of the vault that wait for an answer, or nil.
func (s *State) ReadPending(ctx context.Context, id domain.VaultID) (*transfer.PausedRemovals, error) {
	var state transfer.PausedRemovals
	found, err := s.read(ctx, id, pendingFile, &state)
	if err != nil || !found {
		return nil, err
	}
	return &state, nil
}

// WritePending keeps the removals of the vault that wait for an answer.
func (s *State) WritePending(ctx context.Context, id domain.VaultID, state transfer.PausedRemovals) error {
	return s.write(ctx, id, pendingFile, state)
}

// DeletePending drops the removals of the vault that wait for an answer.
func (s *State) DeletePending(ctx context.Context, id domain.VaultID) error {
	return s.delete(ctx, id, pendingFile)
}

// ReadApproval returns the answer to the paused removals of the vault, or nil.
func (s *State) ReadApproval(ctx context.Context, id domain.VaultID) (*transfer.Approval, error) {
	var approval transfer.Approval
	found, err := s.read(ctx, id, answerFile, &approval)
	if err != nil || !found {
		return nil, err
	}
	return &approval, nil
}

// WriteApproval keeps the answer to the paused removals of the vault.
func (s *State) WriteApproval(ctx context.Context, id domain.VaultID, approval transfer.Approval) error {
	return s.write(ctx, id, answerFile, approval)
}

// DeleteApproval drops the answer to the paused removals of the vault.
func (s *State) DeleteApproval(ctx context.Context, id domain.VaultID) error {
	return s.delete(ctx, id, answerFile)
}

// ReadAbsences returns when each synced file that is gone from the vault was first missed.
func (s *State) ReadAbsences(ctx context.Context, id domain.VaultID) (map[string]int64, error) {
	var document absencesDocument
	if _, err := s.read(ctx, id, absencesFile, &document); err != nil {
		return nil, err
	}
	if document.Absent == nil {
		return map[string]int64{}, nil
	}
	return document.Absent, nil
}

// WriteAbsences replaces when each synced file that is gone from the vault was first missed.
func (s *State) WriteAbsences(ctx context.Context, id domain.VaultID, absent map[string]int64) error {
	return s.write(ctx, id, absencesFile, absencesDocument{Absent: absent})
}

func (s *State) delete(ctx context.Context, id domain.VaultID, name string) error {
	files := s.getVault(id)
	files.mu.Lock()
	defer files.mu.Unlock()
	delete(files.digests, name)
	if err := s.host.Store(id).Delete(ctx, name); err != nil && !errors.Is(err, host.ErrNotExist) {
		return fmt.Errorf("delete %s: %w", name, err)
	}
	return nil
}

// HasPending reports whether removals of the vault wait for an answer.
func (s *State) HasPending(ctx context.Context, id domain.VaultID) (bool, error) {
	pending, err := s.ReadPending(ctx, id)
	return pending != nil, err
}
