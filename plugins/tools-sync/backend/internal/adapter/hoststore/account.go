package hoststore

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

const pinsFile = "pins.json"

type pinsDocument struct {
	Keys map[string][]byte `json:"keys"`
}

type journalDocument struct {
	Entries [][]byte `json:"entries"`
}

func getJournalFile(kind restore.Kind) string { return "journal-" + string(kind) + ".json" }

// listVaults returns the ids of the vaults of the installation. The state of the account is kept in the store
// folder of every one of them, so a vault that joins later reads it from the others.
func (s *State) listVaults(ctx context.Context) ([]domain.VaultID, error) {
	vaults, err := s.host.GetVaults(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]domain.VaultID, len(vaults))
	for i, vault := range vaults {
		ids[i] = vault.ID
	}
	return ids, nil
}

// ReadPin returns the key pinned for the server at the address, nil when none is.
func (s *State) ReadPin(ctx context.Context, address string) ([]byte, error) {
	ids, err := s.listVaults(ctx)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		var document pinsDocument
		if _, err := s.read(ctx, id, pinsFile, &document); err != nil {
			return nil, err
		}
		if key := document.Keys[address]; len(key) > 0 {
			return key, nil
		}
	}
	return nil, nil
}

// WritePin pins the key for the server at the address.
func (s *State) WritePin(ctx context.Context, address string, key []byte) error {
	ids, err := s.listVaults(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errNoVault
	}
	for _, id := range ids {
		var document pinsDocument
		if _, err := s.read(ctx, id, pinsFile, &document); err != nil {
			return err
		}
		if document.Keys == nil {
			document.Keys = map[string][]byte{}
		}
		document.Keys[address] = key
		if err := s.write(ctx, id, pinsFile, document); err != nil {
			return err
		}
	}
	return nil
}

// ReadJournal returns the longest copy of the log of the kind that any vault holds, empty when there is none.
func (s *State) ReadJournal(ctx context.Context, kind restore.Kind) (restore.Journal, error) {
	ids, err := s.listVaults(ctx)
	if err != nil {
		return restore.Journal{}, err
	}
	longest := restore.Journal{Kind: kind, Entries: nil}
	for _, id := range ids {
		var document journalDocument
		if _, err := s.read(ctx, id, getJournalFile(kind), &document); err != nil {
			return restore.Journal{}, err
		}
		if len(document.Entries) > len(longest.Entries) {
			longest.Entries = document.Entries
		}
	}
	return longest, nil
}

// WriteJournal replaces the copy of the log of the journal's kind in every vault.
func (s *State) WriteJournal(ctx context.Context, journal restore.Journal) error {
	ids, err := s.listVaults(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errNoVault
	}
	for _, id := range ids {
		if err := s.write(ctx, id, getJournalFile(journal.Kind), journalDocument{Entries: journal.Entries}); err != nil {
			return err
		}
	}
	return nil
}

var errNoVault = errors.New("the installation has no vault to keep the copy in")
