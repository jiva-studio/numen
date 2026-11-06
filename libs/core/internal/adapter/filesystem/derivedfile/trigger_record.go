package derivedfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// TriggerRecords keeps the record of the triggers of a vault as one JSON file in the core's folder of
// the vault's store.
type TriggerRecords struct {
	Options vaultconfig.Options
	// Folder is the core's folder of the store.
	Folder string
}

// triggerRecordName is the name of the record in the core's folder.
const triggerRecordName = "triggers.json"

// Open opens the record of the vault.
func (t TriggerRecords) Open(v domain.Vault) (pluginport.TriggerRecord, error) {
	store, err := OpenDerived(v.Path, t.Options, t.Folder)
	if err != nil {
		return nil, err
	}
	return triggerRecord{store: store, name: t.Folder + "/" + triggerRecordName}, nil
}

type triggerRecord struct {
	store *DerivedStore
	name  string
}

func (r triggerRecord) Read(ctx context.Context) (pluginport.TriggerState, error) {
	raw, err := r.store.Read(ctx, r.name)
	if errors.Is(err, fs.ErrNotExist) {
		return pluginport.TriggerState{}, nil
	}
	if err != nil {
		return pluginport.TriggerState{}, err
	}
	var state pluginport.TriggerState
	return state, json.Unmarshal(raw, &state)
}

func (r triggerRecord) Write(ctx context.Context, state pluginport.TriggerState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return r.store.Write(ctx, r.name, raw)
}
