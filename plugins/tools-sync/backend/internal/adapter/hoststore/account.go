package hoststore

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

const pinsFile = "pins.json"

type pinsDocument struct {
	Keys map[string][]byte `json:"keys"`
}

type journalDocument struct {
	Entries [][]byte `json:"entries"`
}

func getJournalFile(kind restore.Kind) string { return "journal-" + string(kind) + ".json" }

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

// WritePin pins the key for the address in every vault. The reads and the writes of all vaults run as one step, so two pins written at once both stay.
func (s *State) WritePin(ctx context.Context, address string, key []byte) error {
	s.pins.Lock()
	defer s.pins.Unlock()
	ids, err := s.listVaults(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errNoVault
	}
	kept := 0
	for _, id := range ids {
		var document pinsDocument
		if _, err := s.read(ctx, id, pinsFile, &document); err != nil {
			return err
		}
		if document.Keys == nil {
			document.Keys = map[string][]byte{}
		}
		document.Keys[address] = key
		isKept, err := s.writeCopy(ctx, id, pinsFile, document)
		if err != nil {
			return err
		}
		if isKept {
			kept++
		}
	}
	return checkAnyKept(kept)
}

func (s *State) writeCopy(ctx context.Context, id domain.VaultID, name string, value any) (bool, error) {
	err := s.write(ctx, id, name, value)
	if errors.Is(err, host.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func checkAnyKept(kept int) error {
	if kept == 0 {
		return errNoVault
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
	kept := 0
	for _, id := range ids {
		isKept, err := s.writeCopy(ctx, id, getJournalFile(journal.Kind), journalDocument{Entries: journal.Entries})
		if err != nil {
			return err
		}
		if isKept {
			kept++
		}
	}
	return checkAnyKept(kept)
}

var errNoVault = vaultlink.ErrNoVaults

const entrySightingsFile = "chain-sightings.json"

type entrySightingsDocument struct {
	Seen map[string]int64 `json:"seen"`
}

// ReadEntrySightings returns the earliest time any vault holds for each entry, nil when no vault holds a copy.
func (s *State) ReadEntrySightings(ctx context.Context) (map[string]int64, error) {
	ids, err := s.listVaults(ctx)
	if err != nil {
		return nil, err
	}
	var merged map[string]int64
	for _, id := range ids {
		var document entrySightingsDocument
		found, err := s.read(ctx, id, entrySightingsFile, &document)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if merged == nil {
			merged = map[string]int64{}
		}
		for entry, at := range document.Seen {
			if first, isKnown := merged[entry]; !isKnown || at < first {
				merged[entry] = at
			}
		}
	}
	return merged, nil
}

// WriteEntrySightings replaces the copy of the sightings in every vault.
func (s *State) WriteEntrySightings(ctx context.Context, seen map[string]int64) error {
	ids, err := s.listVaults(ctx)
	if err != nil {
		return err
	}
	kept := 0
	for _, id := range ids {
		isKept, err := s.writeCopy(ctx, id, entrySightingsFile, entrySightingsDocument{Seen: seen})
		if err != nil {
			return err
		}
		if isKept {
			kept++
		}
	}
	return checkAnyKept(kept)
}
