package derivedfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// TriggerRecords keeps the record of the triggers of a vault as one JSON file in the core's folder of
// the vault's store.
type TriggerRecords struct {
	Options vaultconfig.Options
	Folder  string
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

// triggerStateFile is the record as the file holds it.
type triggerStateFile struct {
	Files map[string]triggerFileFile `json:"files"`
	Runs  map[string]string          `json:"runs"`
}

// triggerFileFile is one file of the record as the file holds it.
type triggerFileFile struct {
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Hash    string    `json:"hash"`
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
	var stored triggerStateFile
	if err := json.Unmarshal(raw, &stored); err != nil {
		return pluginport.TriggerState{}, err
	}
	state := pluginport.TriggerState{Files: make(map[string]pluginport.TriggerFile, len(stored.Files)), Runs: stored.Runs}
	for path, one := range stored.Files {
		state.Files[path] = pluginport.TriggerFile{Size: one.Size, ModTime: one.ModTime, Hash: one.Hash}
	}
	return state, nil
}

func (r triggerRecord) Write(ctx context.Context, state pluginport.TriggerState) error {
	stored := triggerStateFile{Files: make(map[string]triggerFileFile, len(state.Files)), Runs: state.Runs}
	for path, one := range state.Files {
		stored.Files[path] = triggerFileFile{Size: one.Size, ModTime: one.ModTime, Hash: one.Hash}
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return r.store.Write(ctx, r.name, raw)
}
