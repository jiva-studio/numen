package pluginport

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// PluginTriggerEntry is one trigger an enabled plugin declares, and the plugin that declares it.
type PluginTriggerEntry struct {
	PluginID string
	Trigger  manifest.Trigger
}

// PluginTriggers is the triggers the enabled plugins declare.
type PluginTriggers interface {
	// ListTriggers lists the triggers of every enabled plugin that the host does not hold suspended,
	// in the order of the plugins' ids and then of each manifest.
	ListTriggers() []PluginTriggerEntry
}

// TriggerFile is a file of the vault as the triggers last saw it: its size, its modification time
// and the hash of its content.
type TriggerFile struct {
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Hash    string    `json:"hash"`
}

// TriggerState is what the triggers of a vault last saw and ran: each file by its path, and for each
// trigger over a file the hash of the content it last ran on, by the key of the trigger and the path.
type TriggerState struct {
	Files map[string]TriggerFile `json:"files"`
	Runs  map[string]string      `json:"runs"`
}

// TriggerRecord is what the triggers of one vault saw and ran.
type TriggerRecord interface {
	// Read returns the state, empty where none is recorded.
	Read(ctx context.Context) (TriggerState, error)
	// Write replaces the state.
	Write(ctx context.Context, state TriggerState) error
}

// TriggerRecords opens the record of the triggers of a vault.
type TriggerRecords interface {
	Open(v domain.Vault) (TriggerRecord, error)
}
